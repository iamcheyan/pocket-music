package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func multipartFile(t *testing.T, name, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return v
}

func TestFavoritesLifecycleAndHomeSections(t *testing.T) {
	s := testServer(t)
	home := decode[struct {
		FavoriteTracks []Track          `json:"favorite_tracks"`
		FavoriteAlbums []map[string]any `json:"favorite_albums"`
		RecentlyAdded  []Track          `json:"recently_added"`
		MostPlayed     []Track          `json:"most_played"`
	}](t, call(t, s, "GET", "home", ""))
	if len(home.FavoriteTracks) == 0 || len(home.FavoriteAlbums) == 0 {
		t.Fatalf("seeded favourites missing: %d %d", len(home.FavoriteTracks), len(home.FavoriteAlbums))
	}
	if len(home.RecentlyAdded) == 0 || len(home.MostPlayed) == 0 {
		t.Fatalf("home sections missing: %d %d", len(home.RecentlyAdded), len(home.MostPlayed))
	}

	if w := call(t, s, "POST", "favorites", `{"kind":"track","id":"mock-240"}`); w.Code != 200 {
		t.Fatalf("add track: %d %s", w.Code, w.Body)
	}
	if w := call(t, s, "POST", "favorites", `{"kind":"album","id":"1"}`); w.Code != 200 {
		t.Fatalf("add album: %d %s", w.Code, w.Body)
	}
	if w := call(t, s, "POST", "favorites", `{"kind":"artist","id":"1"}`); w.Code != 200 {
		t.Fatalf("add artist: %d %s", w.Code, w.Body)
	}
	if w := call(t, s, "POST", "favorites", `{"kind":"bogus","id":"1"}`); w.Code != 400 {
		t.Fatalf("invalid kind accepted: %d", w.Code)
	}
	if w := call(t, s, "POST", "favorites", `{"kind":"track","id":"does-not-exist"}`); w.Code != 404 {
		t.Fatalf("missing entity accepted: %d", w.Code)
	}

	got := decode[struct {
		Tracks []Track             `json:"tracks"`
		Albums []map[string]any    `json:"albums"`
		Artist []map[string]any    `json:"artists"`
		IDs    map[string][]string `json:"ids"`
	}](t, call(t, s, "GET", "favorites", ""))
	if len(got.Albums) == 0 || len(got.Artist) == 0 || len(got.Tracks) == 0 {
		t.Fatalf("fav groups empty: %+v", got)
	}
	found := false
	for _, id := range got.IDs["track"] {
		if id == "mock-240" {
			found = true
		}
	}
	if !found {
		t.Fatalf("added favourite missing from ids: %v", got.IDs["track"])
	}

	if w := call(t, s, "DELETE", "favorites", `{"kind":"track","id":"mock-240"}`); w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	after := decode[struct {
		IDs map[string][]string `json:"ids"`
	}](t, call(t, s, "GET", "favorites", ""))
	for _, id := range after.IDs["track"] {
		if id == "mock-240" {
			t.Fatal("favourite not removed")
		}
	}

	// The library projection must flag favourites so rows can render a heart.
	track := decode[Track](t, call(t, s, "GET", "tracks/mock-007", ""))
	if !track.Favorite {
		t.Fatalf("seeded favourite not flagged: %+v", track)
	}
	if track.AlbumID == 0 || track.ArtistID == 0 {
		t.Fatalf("track projection missing album/artist ids: %+v", track)
	}
}

func TestPlaybackModesQueueOperationsAndSkipAccounting(t *testing.T) {
	s := testServer(t)
	if w := call(t, s, "PUT", "player/mode", `{"mode":"nope"}`); w.Code != 400 {
		t.Fatalf("invalid mode accepted: %d", w.Code)
	}
	if w := call(t, s, "PUT", "player/mode", `{"mode":"repeat_all"}`); w.Code != 200 {
		t.Fatalf("valid mode rejected: %d %s", w.Code, w.Body)
	}

	// play_next puts the track at the head of the queue.
	call(t, s, "POST", "queue", `{"action":"add","track_id":"mock-002"}`)
	call(t, s, "POST", "queue", `{"action":"play_next","track_id":"mock-003"}`)
	queue := decode[struct {
		Queue []Track `json:"queue"`
	}](t, call(t, s, "POST", "queue", `{"action":"add","track_id":"mock-004"}`))
	if len(queue.Queue) != 3 || queue.Queue[0].ID != "mock-003" || queue.Queue[1].ID != "mock-002" {
		t.Fatalf("queue order: %+v", queue.Queue)
	}

	// Reorder moves the item and keeps every entry.
	reordered := decode[struct {
		Queue []Track `json:"queue"`
	}](t, call(t, s, "POST", "queue", `{"action":"reorder","index":0,"to":2}`))
	if len(reordered.Queue) != 3 || reordered.Queue[2].ID != "mock-003" {
		t.Fatalf("reorder: %+v", reordered.Queue)
	}
	call(t, s, "POST", "queue", `{"action":"remove","index":1}`)
	if q := decode[struct {
		Queue []Track `json:"queue"`
	}](t, call(t, s, "POST", "queue", `{"action":"clear"}`)); len(q.Queue) != 0 {
		t.Fatalf("clear: %+v", q.Queue)
	}

	// Skipping half way through a track records a skip and a partial completion.
	if w := call(t, s, "POST", "play", `{"track_id":"mock-010","queue":["mock-011"]}`); w.Code != 200 {
		t.Fatalf("play: %d %s", w.Code, w.Body)
	}
	s.playerMu.Lock()
	s.player.Position = s.player.Duration * 0.4
	s.playerMu.Unlock()
	call(t, s, "POST", "play", `{"track_id":"mock-012"}`)

	var skips int
	var completion float64
	s.db.QueryRow(`SELECT skip_count,completion_rate FROM history WHERE track_id='mock-010'`).Scan(&skips, &completion)
	if skips != 1 {
		t.Fatalf("skip_count=%d", skips)
	}
	if completion < 0.35 || completion > 0.45 {
		t.Fatalf("completion_rate=%v", completion)
	}

	// Natural completion records rate 1 and advances across the queue.
	call(t, s, "POST", "play", `{"track_id":"mock-013","queue":["mock-014"],"mode":"normal"}`)
	s.playerMu.Lock()
	s.player.Duration = 0.01
	s.player.Position = 0.02
	s.playerMu.Unlock()
	s.syncPlayerState()
	state := decode[Player](t, call(t, s, "GET", "player/state", ""))
	if state.Track == nil || state.Track.ID != "mock-014" {
		t.Fatalf("queue did not advance: %+v", state.Track)
	}
	var rate float64
	var plays int
	s.db.QueryRow(`SELECT completion_rate FROM history WHERE track_id='mock-013'`).Scan(&rate)
	s.db.QueryRow(`SELECT play_count FROM history WHERE track_id='mock-013'`).Scan(&plays)
	if rate < 0.99 {
		t.Fatalf("completion_rate=%v", rate)
	}
	if plays < 1 {
		t.Fatalf("play_count=%d", plays)
	}
}

func TestCoverLyricsAndArtistAlbumDetail(t *testing.T) {
	s := testServer(t)
	w := call(t, s, "GET", "cover/1", "")
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("placeholder cover: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "<svg") || !strings.Contains(w.Body.String(), "<text") {
		t.Fatalf("cover body: %s", w.Body.String())
	}

	lyrics := decode[struct {
		Lines  []lyricLine `json:"lines"`
		Synced bool        `json:"synced"`
	}](t, call(t, s, "GET", "lyrics/mock-001", ""))
	if len(lyrics.Lines) == 0 || !lyrics.Synced {
		t.Fatalf("demo lyrics: %+v", lyrics)
	}
	if lyrics.Lines[0].Time != 0 || lyrics.Lines[1].Time <= lyrics.Lines[0].Time {
		t.Fatalf("lyrics not ordered: %+v", lyrics.Lines)
	}

	album := decode[struct {
		Title  string  `json:"title"`
		Tracks []Track `json:"tracks"`
		Cover  string  `json:"cover"`
	}](t, call(t, s, "GET", "albums/1", ""))
	if album.Title == "" || len(album.Tracks) == 0 || album.Cover != "/api/cover/1" {
		t.Fatalf("album detail: %+v", album)
	}
	artist := decode[struct {
		Name   string           `json:"name"`
		Tracks []Track          `json:"tracks"`
		Albums []map[string]any `json:"albums"`
	}](t, call(t, s, "GET", "artists/1", ""))
	if artist.Name == "" || len(artist.Tracks) == 0 || len(artist.Albums) == 0 {
		t.Fatalf("artist detail: %+v", artist)
	}
}

func TestLyricsFileTakesPrecedence(t *testing.T) {
	s := testServer(t)
	dir := t.TempDir()
	audio := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(audio, []byte("not-audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	lrc := "[00:01.00]first\n[00:03.50]second\n[ar:X]\n[00:05.00][00:07.00]twice\n"
	if err := os.WriteFile(filepath.Join(dir, "song.lrc"), []byte(lrc), 0o600); err != nil {
		t.Fatal(err)
	}
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('lrc-track',?,'Song')`, audio)
	got := decode[struct {
		Lines []lyricLine `json:"lines"`
	}](t, call(t, s, "GET", "lyrics/lrc-track", ""))
	want := []string{"first", "second", "twice", "twice"}
	if len(got.Lines) != len(want) {
		t.Fatalf("lines: %+v", got.Lines)
	}
	for i, text := range want {
		if got.Lines[i].Text != text {
			t.Fatalf("line %d = %q", i, got.Lines[i].Text)
		}
	}
	if got.Lines[3].Time != 7 {
		t.Fatalf("multi-timestamp line time: %v", got.Lines[3].Time)
	}
}

func TestSearchGroupsAndFilters(t *testing.T) {
	s := testServer(t)
	all := decode[struct {
		Songs     []Track          `json:"songs"`
		Artists   []map[string]any `json:"artists"`
		Albums    []map[string]any `json:"albums"`
		Playlists []map[string]any `json:"playlists"`
		Total     int              `json:"total"`
	}](t, call(t, s, "GET", "search?q=Nova&type=all", ""))
	if len(all.Artists) == 0 || len(all.Songs) == 0 || all.Total == 0 {
		t.Fatalf("grouped search: %+v", all)
	}
	onlyArtists := decode[struct {
		Songs   []Track          `json:"songs"`
		Artists []map[string]any `json:"artists"`
	}](t, call(t, s, "GET", "search?q=Nova&type=artist", ""))
	if len(onlyArtists.Songs) != 0 || len(onlyArtists.Artists) == 0 {
		t.Fatalf("artist filter: %+v", onlyArtists)
	}
	// Punctuation must not break the FTS expression.
	if w := call(t, s, "GET", "search?q=%22%2A%28Nova", ""); w.Code != 200 {
		t.Fatalf("punctuation search: %d %s", w.Code, w.Body)
	}
	if p := decode[struct {
		Playlists []map[string]any `json:"playlists"`
	}](t, call(t, s, "GET", "search?q=Night&type=playlist", "")); len(p.Playlists) == 0 {
		t.Fatalf("playlist search empty")
	}
}

func TestScanRemovesDeletedFilesAndReportsProgress(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MUSICD_DB", filepath.Join(t.TempDir(), "scan2.db"))
	t.Setenv("MUSICD_MUSIC_DIR", dir)
	t.Setenv("MUSICD_MOCK", "false")
	s, e := NewServer()
	if e != nil {
		t.Fatal(e)
	}
	defer s.db.Close()
	audio := filepath.Join(dir, "keep.wav")
	if e := os.WriteFile(audio, make([]byte, 64), 0o600); e != nil {
		t.Fatal(e)
	}
	gone := filepath.Join(dir, "gone.wav")
	if e := os.WriteFile(gone, make([]byte, 64), 0o600); e != nil {
		t.Fatal(e)
	}
	if e := os.Remove(gone); e != nil {
		t.Fatal(e)
	}
	// Indices look wrong on purpose: the scan must prune rows for files that no
	// longer exist.
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('stale',?,'Gone')`, gone)
	s.db.Exec(`INSERT INTO tracks_fts(track_id,title,artist,album,genre) VALUES('stale','Gone','','','')`)
	result, e := s.provider.Scan(context.Background(), func(ScanProgress) {})
	if e != nil {
		t.Fatal(e)
	}
	if result.Removed != 1 || result.Found != 1 {
		t.Fatalf("scan result: %+v", result)
	}
	var stale int
	s.db.QueryRow(`SELECT count(*) FROM tracks WHERE id='stale'`).Scan(&stale)
	if stale != 0 {
		t.Fatal("stale row survived the scan")
	}
	if status := decode[Progress](t, call(t, s, "GET", "scan/status", "")); status.Error != "" {
		t.Fatalf("status error: %+v", status)
	}
}

func TestSettingsValidationAndStaticServing(t *testing.T) {
	s := testServer(t)
	for _, body := range []string{
		`{"settings":{"theme":"neon"}}`,
		`{"settings":{"language":"fr"}}`,
		`{"settings":{"audio_output":"oss"}}`,
		`{"settings":{"cache_size":"999999"}}`,
		`{"settings":{"unknown_key":"1"}}`,
	} {
		if w := call(t, s, "PUT", "settings", body); w.Code != 400 {
			t.Fatalf("%s accepted: %d", body, w.Code)
		}
	}
	ok := decode[struct {
		Settings map[string]string `json:"settings"`
	}](t, call(t, s, "PUT", "settings", `{"settings":{"theme":"dark","language":"zh","audio_output":"pulse","cache_size":"256","auto_scan":"false"}}`))
	if ok.Settings["theme"] != "dark" || ok.Settings["audio_output"] != "pulse" || ok.Settings["cache_size"] != "256" {
		t.Fatalf("settings round trip: %+v", ok.Settings)
	}
	if s.mpv.outputName() != "pulse" {
		t.Fatalf("mpv output not applied: %q", s.mpv.outputName())
	}

	web := t.TempDir()
	os.WriteFile(filepath.Join(web, "index.html"), []byte("<!doctype html><title>Pocket Music</title>"), 0o600)
	os.WriteFile(filepath.Join(web, "app.js"), []byte("console.log(1)"), 0o600)
	s.webDir = web
	r := httptest.NewRequest("GET", "/library/deep/link", nil)
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Pocket Music") {
		t.Fatalf("spa fallback: %d %s", w.Code, w.Body)
	}
	r = httptest.NewRequest("GET", "/app.js", nil)
	w = httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "console.log") {
		t.Fatalf("static asset: %d %s", w.Code, w.Body)
	}
}

func TestImportM3U8WithBOMAndRelativePaths(t *testing.T) {
	s := testServer(t)
	dir := t.TempDir()
	first := filepath.Join(dir, "one.mp3")
	second := filepath.Join(dir, "two.mp3")
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('imp-1',?,'One')`, first)
	s.db.Exec(`INSERT INTO tracks(id,path,title) VALUES('imp-2',?,'Two')`, second)
	body, contentType := multipartFile(t, "list.m3u8", "\ufeff#EXTM3U\n#EXTINF:100,A - One\none.mp3\n#EXTINF:120,A - Two\ntwo.mp3\n")
	r := httptest.NewRequest(http.MethodPost, "/api/playlists/import", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"count":2`) {
		t.Fatalf("import: %d %s", w.Code, w.Body)
	}
}

func TestStatsAggregatesHistory(t *testing.T) {
	s := testServer(t)
	s.db.Exec(`UPDATE history SET play_count=5, skip_count=2, completion_rate=0.5`)
	stats := decode[struct {
		Library    map[string]int   `json:"library"`
		PlayTime   float64          `json:"play_time"`
		PlayCount  int              `json:"play_count"`
		SkipCount  int              `json:"skip_count"`
		TopArtists []map[string]any `json:"top_artists"`
		TopAlbums  []map[string]any `json:"top_albums"`
		MostPlayed []Track          `json:"most_played"`
	}](t, call(t, s, "GET", "stats", ""))
	if stats.Library["tracks"] != mockTrackCount {
		t.Fatalf("library: %+v", stats.Library)
	}
	if stats.PlayTime <= 0 || stats.PlayCount <= 0 || stats.SkipCount <= 0 {
		t.Fatalf("aggregates: %+v", stats)
	}
	if len(stats.TopArtists) == 0 || len(stats.TopAlbums) == 0 || len(stats.MostPlayed) == 0 {
		t.Fatalf("tops: %+v", stats)
	}
}

// seed10k builds a 10,000 track library for the API benchmarks.
func seed10k(b *testing.B) (*Server, http.Handler) {
	b.Helper()
	b.Setenv("MUSICD_DB", filepath.Join(b.TempDir(), "api.db"))
	b.Setenv("MUSICD_MOCK", "false")
	s, err := NewServer()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { s.db.Close() })
	s.db.Exec(`INSERT OR IGNORE INTO artists(name) VALUES('Perf Artist')`)
	var artistID int
	s.db.QueryRow(`SELECT id FROM artists WHERE name='Perf Artist'`).Scan(&artistID)
	tx, err := s.db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	for i := range 200 {
		tx.Exec(`INSERT OR IGNORE INTO albums(title,artist_id,year) VALUES(?,?,2024)`, fmt.Sprintf("Perf Album %03d", i), artistID)
	}
	for i := range 10000 {
		id := fmt.Sprintf("perf-%05d", i)
		title := fmt.Sprintf("Performance Track %05d", i)
		var albumID int
		tx.QueryRow(`SELECT id FROM albums WHERE title=?`, fmt.Sprintf("Perf Album %03d", i%200)).Scan(&albumID)
		tx.Exec(`INSERT INTO tracks(id,title,artist_id,album_id,genre,year,duration,path,track_no) VALUES(?,?,?,?,?,?,?,?,?)`,
			id, title, artistID, albumID, "Test", 2024, 200, "mock:"+id, i%20)
		tx.Exec(`INSERT INTO tracks_fts(track_id,title,artist,album,genre) VALUES(?,?,?,?,?)`,
			id, title, "Perf Artist", fmt.Sprintf("Perf Album %03d", i%200), "Test")
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	return s, s.routes()
}

func benchGET(b *testing.B, handler http.Handler, path string) {
	b.Helper()
	r := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		b.Fatalf("%s: %d %s", path, w.Code, w.Body)
	}
}

// The three endpoints below cover the interactive paths the UI polls: search,
// a deep library page, and the album grid.
func BenchmarkAPISearch10K(b *testing.B) {
	_, handler := seed10k(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGET(b, handler, "/api/search?q=Performance&type=all")
	}
}

func BenchmarkAPISongQueryWorstCase10K(b *testing.B) {
	_, handler := seed10k(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Every track matches, which is the worst case for the paginated list.
		benchGET(b, handler, "/api/library/songs?q=Performance&page=1&limit=50")
	}
}

func BenchmarkAPILibraryPage10K(b *testing.B) {
	_, handler := seed10k(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGET(b, handler, "/api/library/songs?page=60&limit=50")
	}
}

func BenchmarkAPIAlbums10K(b *testing.B) {
	_, handler := seed10k(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGET(b, handler, "/api/library/albums")
	}
}

type songPageResult struct {
	Items []Track `json:"items"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
}

func TestFoldersGroupingAndFiltering(t *testing.T) {
	s := testServer(t)
	folders := decode[struct {
		Items []struct {
			Name  string `json:"name"`
			Count int    `json:"track_count"`
		} `json:"items"`
	}](t, call(t, s, "GET", "library/folders", ""))
	if len(folders.Items) != mockArtistCount*mockAlbumsPerArtist {
		t.Fatalf("folder groups = %d, want %d", len(folders.Items), mockArtistCount*mockAlbumsPerArtist)
	}
	name := folders.Items[0].Name
	if !strings.Contains(name, "/") {
		t.Fatalf("folder group not artist/album shaped: %q", name)
	}
	for _, folder := range folders.Items {
		if folder.Count != mockTracksPerAlbum {
			t.Fatalf("folder %q count=%d", folder.Name, folder.Count)
		}
	}
	page := decode[songPageResult](t, call(t, s, "GET", "library/songs?limit=500&folder="+url.QueryEscape(name), ""))
	if page.Total != mockTracksPerAlbum || len(page.Items) != mockTracksPerAlbum {
		t.Fatalf("folder filter total=%d items=%d", page.Total, len(page.Items))
	}
	for _, track := range page.Items {
		if !strings.HasPrefix(track.Path, "mock:") || !strings.Contains(track.Path, name) {
			t.Fatalf("folder filter leaked track %q", track.Path)
		}
	}
	// A folder filter combined with a text query must intersect, not union.
	both := decode[songPageResult](t, call(t, s, "GET", "library/songs?limit=500&folder="+url.QueryEscape(name)+"&q=Drift", ""))
	if both.Total >= page.Total && both.Total != 0 {
		t.Fatalf("query+folder did not narrow results: %d vs %d", both.Total, page.Total)
	}
	// Paging keeps a stable total and shrinks the page size.
	first := decode[songPageResult](t, call(t, s, "GET", "library/songs?page=1&limit=10", ""))
	second := decode[songPageResult](t, call(t, s, "GET", "library/songs?page=2&limit=10", ""))
	if first.Total != mockTrackCount || len(first.Items) != 10 || len(second.Items) != 10 {
		t.Fatalf("paging: total=%d first=%d second=%d", first.Total, len(first.Items), len(second.Items))
	}
	if first.Items[0].ID == second.Items[0].ID {
		t.Fatal("page 2 repeated page 1")
	}
}
