package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("MUSICD_DB", filepath.Join(t.TempDir(), "test.db"))
	t.Setenv("MUSICD_MUSIC_DIR", t.TempDir())
	t.Setenv("MUSICD_MOCK", "true")
	s, e := NewServer()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.db.Close() })
	return s
}
func call(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/api/"+path, strings.NewReader(body))
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	return w
}
func TestMigrationAndCoreAPI(t *testing.T) {
	s := testServer(t)
	for _, table := range []string{"tracks", "albums", "artists", "playlists", "playlist_items", "favorites", "history", "settings", "tracks_fts"} {
		var name string
		if e := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE name=?`, table).Scan(&name); e != nil {
			t.Fatalf("schema missing %s: %v", table, e)
		}
	}
	if w := call(t, s, "GET", "health", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"mock":true`) {
		t.Fatalf("health: %d %s", w.Code, w.Body)
	}
	var songs map[string]any
	if e := json.Unmarshal(call(t, s, "GET", "library/songs?limit=20", "").Body.Bytes(), &songs); e != nil || int(songs["total"].(float64)) != mockTrackCount {
		t.Fatalf("library: %v", e)
	}
	if w := call(t, s, "GET", "search?q=Opening&type=song", ""); !strings.Contains(w.Body.String(), "Opening") {
		t.Fatalf("search: %s", w.Body)
	}
}
func TestPlaylistQueueSettingsFavorites(t *testing.T) {
	s := testServer(t)
	w := call(t, s, "POST", "playlists", `{"name":"Test"}`)
	var pl map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &pl); e != nil {
		t.Fatal(e)
	}
	id := int(pl["id"].(float64))
	w = call(t, s, "POST", "playlists/"+strconv.Itoa(id)+"/items", `{"action":"add","track_id":"mock-001"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "mock-001") {
		t.Fatalf("item %s", w.Body)
	}
	w = call(t, s, "GET", "playlists/"+strconv.Itoa(id)+"/export?format=m3u", "")
	if !strings.Contains(w.Body.String(), "#EXTM3U") {
		t.Fatalf("export %s", w.Body)
	}
	if w = call(t, s, "POST", "queue", `{"action":"add","track_id":"mock-001"}`); !strings.Contains(w.Body.String(), "mock-001") {
		t.Fatalf("queue %s", w.Body)
	}
	w = call(t, s, "PUT", "settings", `{"settings":{"language":"ja"}}`)
	if !strings.Contains(w.Body.String(), `"language":"ja"`) {
		t.Fatalf("settings %s", w.Body)
	}
	newFolder := t.TempDir()
	w = call(t, s, "PUT", "settings", `{"settings":{"music_folder":"`+newFolder+`","audio_output":"alsa"}}`)
	if w.Code != http.StatusOK || s.root != newFolder || s.mpv.output != "alsa" {
		t.Fatalf("runtime settings: %d root=%q output=%q", w.Code, s.root, s.mpv.output)
	}
	if w := call(t, s, "POST", "favorites", `{"kind":"track","id":"mock-001"}`); w.Code != http.StatusOK {
		t.Fatalf("favorite add: %d %s", w.Code, w.Body)
	}
	var n int
	s.db.QueryRow(`SELECT count(*) FROM favorites WHERE kind='track' AND entity_id='mock-001'`).Scan(&n)
	if n != 1 {
		t.Fatal("favorite not stored")
	}
}
func TestPlaylistM3URoundTripAndMalformedRollback(t *testing.T) {
	s := testServer(t)
	first := filepath.Join(t.TempDir(), "first.mp3")
	second := filepath.Join(t.TempDir(), "second.mp3")
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('m3u-first',?,'First')`, first)
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('m3u-second',?,'Second')`, second)
	playlist := call(t, s, "POST", "playlists", `{"name":"Round trip"}`)
	var created map[string]any
	if e := json.Unmarshal(playlist.Body.Bytes(), &created); e != nil {
		t.Fatal(e)
	}
	id := int(created["id"].(float64))
	for _, track := range []string{"m3u-first", "m3u-second"} {
		if w := call(t, s, "POST", "playlists/"+strconv.Itoa(id)+"/items", `{"action":"add","track_id":"`+track+`"}`); w.Code != http.StatusOK {
			t.Fatalf("add %s: %d %s", track, w.Code, w.Body)
		}
	}
	data := call(t, s, "GET", "playlists/"+strconv.Itoa(id)+"/export?format=m3u", "").Body.String()
	if !strings.Contains(data, first) || !strings.Contains(data, second) {
		t.Fatalf("export missing paths: %s", data)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, e := mw.CreateFormFile("file", "roundtrip.m3u")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = part.Write([]byte(data)); e != nil {
		t.Fatal(e)
	}
	if e = mw.Close(); e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/playlists/import", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"count":2`) {
		t.Fatalf("import: %d %s", w.Code, w.Body)
	}
	var before int
	s.db.QueryRow(`SELECT count(*) FROM playlists`).Scan(&before)
	body.Reset()
	mw = multipart.NewWriter(&body)
	part, _ = mw.CreateFormFile("file", "invalid.m3u")
	part.Write([]byte("#EXTM3U\n" + strings.Repeat("x", 70*1024)))
	mw.Close()
	r = httptest.NewRequest(http.MethodPost, "/api/playlists/import", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w = httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	var after int
	s.db.QueryRow(`SELECT count(*) FROM playlists`).Scan(&after)

	if w.Code != http.StatusBadRequest || after != before {
		t.Fatalf("malformed import: status=%d playlists=%d before=%d", w.Code, after, before)
	}
}
func TestScannerImportsGeneratedWAV(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MUSICD_DB", filepath.Join(t.TempDir(), "wav.db"))
	t.Setenv("MUSICD_MUSIC_DIR", dir)
	t.Setenv("MUSICD_MOCK", "false")
	s, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Close()
	path := filepath.Join(dir, "tone.wav")
	if err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-metadata", "title=Generated Tone", "-y", path).Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.provider.Scan(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	var title string
	var duration float64
	if err := s.db.QueryRow(`SELECT title,duration FROM tracks WHERE path=?`, path).Scan(&title, &duration); err != nil {
		t.Fatal(err)
	}
	if title != "Generated Tone" || duration <= 0 {
		t.Fatalf("metadata title=%q duration=%f", title, duration)
	}
}

func TestQueueRepeatOneAndRepeatAll(t *testing.T) {
	s := testServer(t)
	if w := call(t, s, "POST", "play", `{"track_id":"mock-001","mode":"repeat_one"}`); w.Code != 200 {
		t.Fatal(w.Body)
	}
	s.playerMu.Lock()
	original := s.player.Track.ID
	s.advance(1)
	repeated := s.player.Track.ID
	s.playerMu.Unlock()
	if repeated != original {
		t.Fatalf("repeat_one changed track %s -> %s", original, repeated)
	}
	call(t, s, "POST", "play", `{"track_id":"mock-001","mode":"repeat_all"}`)
	s.playerMu.Lock()
	s.player.Queue = nil
	s.advance(1)
	wrapped := s.player.Track.ID
	s.playerMu.Unlock()
	if wrapped != "mock-001" {
		t.Fatalf("repeat_all did not wrap: %q", wrapped)
	}
}

func BenchmarkSearchAndLibrary10K(b *testing.B) {
	b.Setenv("MUSICD_DB", filepath.Join(b.TempDir(), "perf.db"))
	b.Setenv("MUSICD_MOCK", "false")
	s, err := NewServer()
	if err != nil {
		b.Fatal(err)
	}
	defer s.db.Close()
	tx, err := s.db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	for i := range 10000 {
		id := fmt.Sprintf("perf-%05d", i)
		title := fmt.Sprintf("Performance Track %05d", i)
		tx.Exec(`INSERT INTO tracks(id,title,path) VALUES(?,?,?)`, id, title, "mock:"+id)
		tx.Exec(`INSERT INTO tracks_fts(track_id,title,artist,album,genre) VALUES(?,?,?,?,?)`, id, title, "Perf Artist", "Perf Album", "Test")
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.listQuery(Query{Text: "Performance", Page: 1, Limit: 50}); err != nil {
			b.Fatal(err)
		}
		if _, err := s.listQuery(Query{Page: 1, Limit: 50}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestScanIncrementalAndCancellation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MUSICD_DB", filepath.Join(t.TempDir(), "scan.db"))
	t.Setenv("MUSICD_MUSIC_DIR", dir)
	t.Setenv("MUSICD_MOCK", "false")
	s, e := NewServer()
	if e != nil {
		t.Fatal(e)
	}
	defer s.db.Close()
	audio := filepath.Join(dir, "one.wav")
	if e = os.WriteFile(audio, bytes.Repeat([]byte{0}, 32), 0600); e != nil {
		t.Fatal(e)
	}
	info, _ := os.Stat(audio)
	s.importTrack(audio, info)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = s.provider.Scan(ctx, nil); e == nil {
		t.Fatal("expected cancelled scan")
	}
	var count int
	s.db.QueryRow(`SELECT count(*) FROM tracks WHERE path=?`, audio).Scan(&count)
	if count != 1 {
		t.Fatalf("track indexed count=%d", count)
	}
}
func TestMediaRangeAndLyrics(t *testing.T) {
	s := testServer(t)
	f := filepath.Join(t.TempDir(), "sample.wav")
	os.WriteFile(f, []byte("0123456789"), 0600)
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('range',?,'Range')`, f)
	r := httptest.NewRequest("GET", "/api/media/range", nil)
	r.Header.Set("Range", "bytes=2-5")
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	if w.Code != http.StatusPartialContent || w.Body.String() != "2345" {
		t.Fatalf("range %d %q", w.Code, w.Body.String())
	}
	_ = http.MethodGet
}
