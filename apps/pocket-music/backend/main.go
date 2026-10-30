package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Track struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Genre    string  `json:"genre"`
	Year     int     `json:"year"`
	Duration float64 `json:"duration"`
	Path     string  `json:"path,omitempty"`
	AddedAt  string  `json:"added_at"`
}
type Progress struct {
	Scanning bool   `json:"scanning"`
	Found    int    `json:"found"`
	Total    int    `json:"total"`
	Current  string `json:"current,omitempty"`
	Error    string `json:"error,omitempty"`
}
type Server struct {
	db         *sql.DB
	root       string
	mock       bool
	provider   MusicProvider
	mu         sync.Mutex
	scan       Progress
	scanCancel context.CancelFunc
	playerMu   sync.Mutex
	player     Player
	mpv        *mpvClient
}
type Player struct {
	Track    *Track  `json:"track"`
	Playing  bool    `json:"playing"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Volume   float64 `json:"volume"`
	Mode     string  `json:"mode"`
	Queue    []Track `json:"queue"`
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func main() {
	s, e := NewServer()
	if e != nil {
		log.Fatal(e)
	}
	addr := env("MUSICD_ADDR", ":8765")
	log.Printf("musicd listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}
func NewServer() (*Server, error) {
	db, e := sql.Open("sqlite", env("MUSICD_DB", "musicd.db"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Server{db: db, root: env("MUSICD_MUSIC_DIR", filepath.Join(os.Getenv("HOME"), "Music")), mock: strings.EqualFold(env("MUSICD_MOCK", "false"), "true"), player: Player{Volume: 100, Mode: "normal", Queue: []Track{}}, mpv: newMPV()}
	if e = s.migrate(); e != nil {
		return nil, e
	}
	local := &LocalProvider{server: s}
	if s.mock {
		s.provider = &MockProvider{local: local}
		s.seedMock()
	} else {
		s.provider = local
	}
	return s, nil
}
func (s *Server) migrate() error {
	stmts := []string{`CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL)`, `INSERT INTO schema_version SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM schema_version)`, `CREATE TABLE IF NOT EXISTS artists(id INTEGER PRIMARY KEY,name TEXT UNIQUE NOT NULL)`, `CREATE TABLE IF NOT EXISTS albums(id INTEGER PRIMARY KEY,title TEXT NOT NULL,artist_id INTEGER,year INTEGER,cover TEXT,UNIQUE(title,artist_id))`, `CREATE TABLE IF NOT EXISTS tracks(id TEXT PRIMARY KEY,title TEXT,artist_id INTEGER,album_id INTEGER,genre TEXT,year INTEGER,duration REAL,path TEXT UNIQUE NOT NULL,mtime INTEGER,size INTEGER,added_at TEXT DEFAULT CURRENT_TIMESTAMP)`, `CREATE INDEX IF NOT EXISTS tracks_title_idx ON tracks(title)`, `CREATE INDEX IF NOT EXISTS tracks_artist_idx ON tracks(artist_id)`, `CREATE INDEX IF NOT EXISTS tracks_album_idx ON tracks(album_id)`, `CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(track_id UNINDEXED,title,artist,album,genre)`, `CREATE TABLE IF NOT EXISTS playlists(id INTEGER PRIMARY KEY,name TEXT NOT NULL,created_at TEXT DEFAULT CURRENT_TIMESTAMP)`, `CREATE TABLE IF NOT EXISTS playlist_items(playlist_id INTEGER,position INTEGER,track_id TEXT,PRIMARY KEY(playlist_id,position))`, `CREATE TABLE IF NOT EXISTS favorites(kind TEXT,entity_id TEXT,created_at TEXT DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(kind,entity_id))`, `CREATE TABLE IF NOT EXISTS history(track_id TEXT PRIMARY KEY,last_played TEXT,play_count INTEGER DEFAULT 0,skip_count INTEGER DEFAULT 0,completion_rate REAL DEFAULT 0)`, `CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT)`}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, q := range stmts {
		if _, e = tx.Exec(q); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(`DELETE FROM tracks_fts WHERE track_id NOT IN (SELECT id FROM tracks)`); e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE schema_version SET version=1 WHERE version<1`); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	defaults := map[string]string{"language": "en", "theme": "system", "music_folder": s.root, "cache_size": "512", "auto_scan": "false", "audio_output": "null", "proxy": ""}
	for k, v := range defaults {
		if _, e = s.db.Exec(`INSERT OR IGNORE INTO settings(key,value) VALUES(?,?)`, k, v); e != nil {
			return e
		}
	}
	if value, err := s.setting("music_folder"); err == nil && value != "" {
		s.root = value
	}
	if value, err := s.setting("audio_output"); err == nil && value != "" {
		s.mpv.setOutput(value)
	}
	if value, err := s.setting("auto_scan"); err == nil && strings.EqualFold(value, "true") {
		go func() { s.startScan(httptest.NewRecorder()) }()
	}
	return nil
}
func (s *Server) setting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&value)
	return value, err
}
func (s *Server) seedMock() {
	var n int
	s.db.QueryRow(`SELECT count(*) FROM tracks`).Scan(&n)
	if n > 0 {
		return
	}
	tx, e := s.db.Begin()
	if e != nil {
		return
	}
	defer tx.Rollback()
	for i := 1; i <= 64; i++ {
		artist := fmt.Sprintf("Artist %02d", (i-1)%8+1)
		album := fmt.Sprintf("Album %02d", (i-1)/8+1)
		tx.Exec(`INSERT OR IGNORE INTO artists(name) VALUES(?)`, artist)
		var aid int
		tx.QueryRow(`SELECT id FROM artists WHERE name=?`, artist).Scan(&aid)
		tx.Exec(`INSERT INTO albums(title,artist_id,year) SELECT ?,?,2024 WHERE NOT EXISTS(SELECT 1 FROM albums WHERE title=? AND artist_id=?)`, album, aid, album, aid)
		var alid int
		tx.QueryRow(`SELECT id FROM albums WHERE title=? AND artist_id=?`, album, aid).Scan(&alid)
		id := fmt.Sprintf("mock-%03d", i)
		title := fmt.Sprintf("Demo Track %03d", i)
		tx.Exec(`INSERT OR IGNORE INTO tracks(id,title,artist_id,album_id,genre,year,duration,path) VALUES(?,?,?,?,?,?,?,?)`, id, title, aid, alid, "Electronic", 2024, 180+float64(i), "mock:"+id)
		tx.Exec(`INSERT OR IGNORE INTO tracks_fts(track_id,title,artist,album,genre) VALUES(?,?,?,?,?)`, id, title, artist, album, "Electronic")
	}
	tx.Commit()
}
func (s *Server) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/", s.api)
	return m
}
func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if p == "health" {
		providerName := "local"
		if s.provider != nil {
			providerName = s.provider.Name()
		}
		s.json(w, map[string]any{"provider": providerName, "mock": s.mock, "audio": map[string]any{"status": "ready", "ao": env("MUSICD_AO", "null")}})
		return
	}
	if p == "home" {
		a, _ := s.list("added_at", 4)
		b, _ := s.list("last_played", 4)
		c, _ := s.list("play_count", 4)
		s.json(w, map[string]any{"recently_added": a, "recently_played": b, "most_played": c})
		return
	}
	if p == "library/songs" {
		s.songs(w, r)
		return
	}
	if p == "library/albums" || p == "library/artists" || p == "library/genres" || p == "library/folders" {
		s.collections(w, r, p)
		return
	}
	if p == "search" {
		s.search(w, r)
		return
	}
	if p == "player/state" {
		s.syncPlayerState()
		s.playerMu.Lock()
		state := s.player
		s.playerMu.Unlock()
		s.json(w, state)
		return
	}
	if p == "player/pause" || p == "player/resume" || p == "player/next" || p == "player/prev" || p == "player/seek" || p == "player/volume" {
		s.playerAction(w, r, p)
		return
	}
	if p == "play" {
		s.play(w, r)
		return
	}
	if p == "queue" {
		s.queue(w, r)
		return
	}
	if p == "settings" {
		s.settings(w, r)
		return
	}
	if p == "scan" {
		if r.Method == "POST" {
			s.startScan(w)
			return
		}
		s.mu.Lock()
		status := s.scan
		s.mu.Unlock()
		s.json(w, status)
		return
	}
	if p == "scan/status" {
		s.mu.Lock()
		status := s.scan
		s.mu.Unlock()
		s.json(w, status)
		return
	}
	if p == "scan/cancel" && r.Method == "POST" {
		s.mu.Lock()
		if s.scanCancel != nil {
			s.scanCancel()
		}
		s.mu.Unlock()
		s.json(w, map[string]bool{"cancelled": true})
		return
	}
	if p == "playlists" {
		s.playlists(w, r)
		return
	}
	if p == "favorites" {
		s.favorites(w, r)
		return
	}
	if p == "stats" {
		var playTime float64
		s.db.QueryRow(`SELECT COALESCE(SUM(t.duration * h.play_count),0) FROM tracks t JOIN history h ON h.track_id=t.id`).Scan(&playTime)
		artists := []map[string]any{}
		rows, err := s.db.Query(`SELECT ar.id,ar.name,COALESCE(SUM(h.play_count),0) AS plays FROM artists ar JOIN tracks t ON t.artist_id=ar.id JOIN history h ON h.track_id=t.id GROUP BY ar.id ORDER BY plays DESC,ar.name LIMIT 10`)
		if err == nil {
			for rows.Next() {
				var id, plays int
				var name string
				if rows.Scan(&id, &name, &plays) == nil {
					artists = append(artists, map[string]any{"id": id, "name": name, "play_count": plays})
				}
			}
			rows.Close()
		}
		albums := []map[string]any{}
		rows, err = s.db.Query(`SELECT al.id,al.title,COALESCE(ar.name,''),COALESCE(SUM(h.play_count),0) AS plays FROM albums al LEFT JOIN artists ar ON ar.id=al.artist_id JOIN tracks t ON t.album_id=al.id JOIN history h ON h.track_id=t.id GROUP BY al.id ORDER BY plays DESC,al.title LIMIT 10`)
		if err == nil {
			for rows.Next() {
				var id, plays int
				var title, artist string
				if rows.Scan(&id, &title, &artist, &plays) == nil {
					albums = append(albums, map[string]any{"id": id, "title": title, "artist": artist, "play_count": plays})
				}
			}
			rows.Close()
		}
		s.json(w, map[string]any{"play_time": playTime, "most_played": mustList(s, "play_count", 10), "top_artists": artists, "top_albums": albums})
		return
	}

	if len(parts) == 2 && parts[0] == "tracks" {
		t, e := s.track(parts[1])
		if e != nil {
			s.fail(w, 404, "not_found", e.Error())
			return
		}
		s.json(w, t)
		return
	}
	if len(parts) == 2 && parts[0] == "media" {
		s.media(w, r, parts[1])
		return
	}
	if len(parts) == 2 && parts[0] == "lyrics" {
		s.lyrics(w, parts[1])
		return
	}
	if len(parts) == 2 && parts[0] == "cover" {
		s.cover(w, r, parts[1])
		return
	}
	if p == "playlists/import" && r.Method == http.MethodPost {
		s.importPlaylist(w, r)
		return
	}
	if len(parts) == 2 && (parts[0] == "albums" || parts[0] == "artists") {
		s.detail(w, r, parts[0], parts[1])
		return
	}
	if len(parts) == 2 && parts[0] == "playlists" {
		s.playlistByID(w, r, parts[1])
		return
	}
	if len(parts) == 3 && parts[0] == "playlists" && parts[2] == "items" {
		s.playlistItems(w, r, parts[1])
		return
	}
	if len(parts) == 3 && parts[0] == "playlists" && parts[2] == "export" {
		s.exportPlaylist(w, parts[1], r.URL.Query().Get("format"))
		return
	}
	s.fail(w, 404, "not_found", "unknown endpoint")
}
func (s *Server) json(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func (s *Server) fail(w http.ResponseWriter, status int, code, msg string) {
	w.WriteHeader(status)
	s.json(w, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
func (s *Server) list(order string, n int) ([]Track, error) {
	q := `SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM tracks t LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id ORDER BY t.added_at DESC LIMIT ?`
	if order == "last_played" || order == "play_count" {
		q = `SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM tracks t LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id LEFT JOIN history h ON h.track_id=t.id ORDER BY h.` + order + ` DESC LIMIT ?`
	}
	rows, e := s.db.Query(q, n)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Track{}
	for rows.Next() {
		var t Track
		rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt)
		out = append(out, t)
	}
	return out, rows.Err()
}
func mustList(s *Server, o string, n int) []Track { x, _ := s.list(o, n); return x }
func (s *Server) songs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 50
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	where := ""
	args := []any{}
	if q != "" {
		where = `WHERE t.id IN (SELECT track_id FROM tracks_fts WHERE tracks_fts MATCH ?)`
		args = append(args, `"`+strings.ReplaceAll(q, `"`, `""`)+`"`)
	}
	var total int
	s.db.QueryRow(`SELECT count(*) FROM tracks t `+where, args...).Scan(&total)
	args = append(args, limit, (page-1)*limit)
	rows, e := s.db.Query(`SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM tracks t LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id `+where+` ORDER BY t.title LIMIT ? OFFSET ?`, args...)
	if e != nil {
		s.fail(w, 500, "query", e.Error())
		return
	}
	defer rows.Close()
	items := []Track{}
	for rows.Next() {
		var t Track
		rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt)
		items = append(items, t)
	}
	s.json(w, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}
func (s *Server) collections(w http.ResponseWriter, r *http.Request, p string) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	like := "%" + q + "%"
	var rows *sql.Rows
	var e error
	switch p {
	case "library/albums":
		rows, e = s.db.Query(`SELECT al.id,al.title,COALESCE(ar.name,''),COALESCE(al.year,0),count(t.id) FROM albums al LEFT JOIN artists ar ON ar.id=al.artist_id LEFT JOIN tracks t ON t.album_id=al.id WHERE al.title LIKE ? GROUP BY al.id ORDER BY al.title`, like)
	case "library/artists":
		rows, e = s.db.Query(`SELECT ar.id,ar.name,count(t.id) FROM artists ar LEFT JOIN tracks t ON t.artist_id=ar.id WHERE ar.name LIKE ? GROUP BY ar.id ORDER BY ar.name`, like)
	case "library/genres":
		rows, e = s.db.Query(`SELECT DISTINCT genre FROM tracks WHERE genre<>'' ORDER BY genre`)
	case "library/folders":
		rows, e = s.db.Query(`SELECT DISTINCT path FROM tracks ORDER BY path`)
	}
	if e != nil {
		s.fail(w, 500, "query", e.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		if p == "library/albums" {
			var id, year, count int
			var title, artist string
			rows.Scan(&id, &title, &artist, &year, &count)
			out = append(out, map[string]any{"id": id, "title": title, "artist": artist, "year": year, "track_count": count})
		} else if p == "library/artists" {
			var id, count int
			var name string
			rows.Scan(&id, &name, &count)
			out = append(out, map[string]any{"id": id, "name": name, "track_count": count})
		} else {
			var v string
			rows.Scan(&v)
			out = append(out, map[string]any{"name": v})
		}
	}
	s.json(w, map[string]any{"items": out, "total": len(out)})
}
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kind := r.URL.Query().Get("type")
	songs := []Track{}
	if q != "" && (kind == "" || kind == "all" || kind == "song") {
		songs, _ = s.listQuery(Query{Text: q, Page: 1, Limit: 50})
	}
	albums := []map[string]any{}
	artists := []map[string]any{}
	playlists := []map[string]any{}
	if q != "" && (kind == "" || kind == "all" || kind == "album") {
		rows, _ := s.db.Query(`SELECT id,title FROM albums WHERE title LIKE ? ORDER BY title LIMIT 50`, "%"+q+"%")
		if rows != nil {
			for rows.Next() {
				var id int
				var title string
				rows.Scan(&id, &title)
				albums = append(albums, map[string]any{"id": id, "title": title})
			}
			rows.Close()
		}
	}
	if q != "" && (kind == "" || kind == "all" || kind == "artist") {
		rows, _ := s.db.Query(`SELECT id,name FROM artists WHERE name LIKE ? ORDER BY name LIMIT 50`, "%"+q+"%")
		if rows != nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				artists = append(artists, map[string]any{"id": id, "name": name})
			}
			rows.Close()
		}
	}
	if q != "" && (kind == "" || kind == "all" || kind == "playlist") {
		rows, _ := s.db.Query(`SELECT id,name FROM playlists WHERE name LIKE ? ORDER BY name LIMIT 50`, "%"+q+"%")
		if rows != nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				playlists = append(playlists, map[string]any{"id": id, "name": name})
			}
			rows.Close()
		}
	}
	s.json(w, map[string]any{"songs": songs, "albums": albums, "artists": artists, "playlists": playlists})
}
func (s *Server) track(id string) (Track, error) {
	var t Track
	e := s.db.QueryRow(`SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM tracks t LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id WHERE t.id=?`, id).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt)
	return t, e
}
func (s *Server) play(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ID   string `json:"track_id"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		s.fail(w, 400, "invalid_json", err.Error())
		return
	}
	t, err := s.track(b.ID)
	if err != nil {
		s.fail(w, 404, "track_not_found", "track not found")
		return
	}
	if b.Mode != "" && b.Mode != "normal" && b.Mode != "repeat_all" && b.Mode != "repeat_one" && b.Mode != "shuffle" {
		s.fail(w, 400, "invalid_mode", "unsupported playback mode")
		return
	}
	s.playerMu.Lock()
	s.player.Track, s.player.Playing, s.player.Position, s.player.Duration = &t, true, 0, t.Duration
	if b.Mode != "" {
		s.player.Mode = b.Mode
	}
	s.playerMu.Unlock()
	s.db.Exec(`INSERT INTO history(track_id,last_played,play_count) VALUES(?,CURRENT_TIMESTAMP,1) ON CONFLICT(track_id) DO UPDATE SET last_played=CURRENT_TIMESTAMP,play_count=play_count+1`, t.ID)
	if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
		if err := s.mpv.start(t.Path); err != nil {
			s.playerMu.Lock()
			s.player.Playing = false
			s.playerMu.Unlock()
		}
	}
	s.playerMu.Lock()
	state := s.player
	s.playerMu.Unlock()
	s.json(w, state)
}
func (s *Server) syncPlayerState() {
	s.playerMu.Lock()
	if s.player.Track == nil || s.player.Track.Path == "" || strings.HasPrefix(s.player.Track.Path, "mock:") {
		s.playerMu.Unlock()
		return
	}
	s.playerMu.Unlock()
	position, duration, paused, err := s.mpv.status()
	if err != nil {
		return
	}
	s.playerMu.Lock()
	if s.player.Track != nil {
		if s.player.Track != nil && s.player.Duration > 0 && position >= s.player.Duration-0.25 {
			s.completeTrack()
		} else {
			s.player.Position = position
			if duration > 0 {
				s.player.Duration = duration
			}
			s.player.Playing = !paused
		}
	}
	s.playerMu.Unlock()
}
func (s *Server) completeTrack() {
	if s.player.Track == nil {
		return
	}
	t := *s.player.Track
	s.db.Exec(`UPDATE history SET completion_rate=1 WHERE track_id=?`, t.ID)
	s.advance(1)
}
func (s *Server) advance(delta int) {
	if delta < 0 && s.player.Track != nil {
		t := *s.player.Track
		s.player.Position = 0
		s.player.Playing = true
		if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
			_ = s.mpv.start(t.Path)
		}
		return
	}
	if s.player.Mode == "repeat_one" && s.player.Track != nil {
		t := *s.player.Track
		s.player.Position = 0
		s.player.Playing = true
		if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
			_ = s.mpv.start(t.Path)
		}
		return
	}
	if len(s.player.Queue) == 0 {
		if s.player.Mode == "repeat_all" && s.player.Track != nil {
			s.player.Queue = append(s.player.Queue, *s.player.Track)
		} else {
			s.player.Playing = false
			return
		}
	}
	index := 0
	if s.player.Mode == "shuffle" && len(s.player.Queue) > 1 {
		index = int(time.Now().UnixNano() % int64(len(s.player.Queue)))
	}
	t := s.player.Queue[index]
	s.player.Queue = append(s.player.Queue[:index], s.player.Queue[index+1:]...)
	s.player.Track, s.player.Position, s.player.Duration, s.player.Playing = &t, 0, t.Duration, true
	s.db.Exec(`INSERT INTO history(track_id,last_played,play_count) VALUES(?,CURRENT_TIMESTAMP,1) ON CONFLICT(track_id) DO UPDATE SET last_played=CURRENT_TIMESTAMP,play_count=play_count+1`, t.ID)
	if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
		if e := s.mpv.start(t.Path); e != nil {
			s.player.Playing = false
		}
	}
}
func (s *Server) playerAction(w http.ResponseWriter, r *http.Request, p string) {
	s.playerMu.Lock()
	switch p {
	case "player/pause":
		s.player.Playing = false
		_ = s.mpv.property("pause", true)
	case "player/resume":
		s.player.Playing = true
		_ = s.mpv.property("pause", false)
	case "player/next":
		s.advance(1)
	case "player/prev":
		s.advance(-1)
	case "player/seek":
		var b struct {
			Position float64 `json:"position"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		s.player.Position = b.Position
		_ = s.mpv.command("seek", b.Position, "absolute")
	case "player/volume":
		var b struct {
			Volume float64 `json:"volume"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		s.player.Volume = b.Volume
		_ = s.mpv.property("volume", b.Volume)
	}
	state := s.player
	s.playerMu.Unlock()
	s.json(w, state)
}
func (s *Server) queue(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Action  string `json:"action"`
		TrackID string `json:"track_id"`
		Index   int    `json:"index"`
		To      int    `json:"to"`
	}
	if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
		s.fail(w, 400, "invalid_json", e.Error())
		return
	}
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	if b.TrackID != "" {
		t, e := s.track(b.TrackID)
		if e != nil {
			s.fail(w, 404, "track_not_found", "track not found")
			return
		}
		if b.Action == "play_next" {
			s.player.Queue = append(s.player.Queue, Track{})
			copy(s.player.Queue[1:], s.player.Queue[:len(s.player.Queue)-1])
			s.player.Queue[0] = t
		} else {
			s.player.Queue = append(s.player.Queue, t)
		}
	}
	if b.Action == "remove" && b.Index >= 0 && b.Index < len(s.player.Queue) {
		s.player.Queue = append(s.player.Queue[:b.Index], s.player.Queue[b.Index+1:]...)
	}
	if b.Action == "reorder" && b.Index >= 0 && b.Index < len(s.player.Queue) && b.To >= 0 && b.To < len(s.player.Queue) {
		t := s.player.Queue[b.Index]
		s.player.Queue = append(s.player.Queue[:b.Index], s.player.Queue[b.Index+1:]...)
		s.player.Queue = append(s.player.Queue, Track{})
		copy(s.player.Queue[b.To+1:], s.player.Queue[b.To:])
		s.player.Queue[b.To] = t
	}
	s.json(w, map[string]any{"queue": s.player.Queue})
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		var b struct {
			Settings map[string]string `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			s.fail(w, 400, "invalid_json", err.Error())
			return
		}
		allowed := map[string]bool{"language": true, "theme": true, "music_folder": true, "cache_size": true, "auto_scan": true, "audio_output": true, "proxy": true}
		for k, v := range b.Settings {
			if !allowed[k] {
				s.fail(w, 400, "invalid_setting", "unsupported setting: "+k)
				return
			}
			if _, err := s.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, k, v); err != nil {
				s.fail(w, 500, "settings", err.Error())
				return
			}
			if k == "music_folder" {
				s.root = v
			}
			if k == "audio_output" {
				s.mpv.setOutput(v)
			}
			if k == "auto_scan" && strings.EqualFold(v, "true") {
				go func() { s.startScan(httptest.NewRecorder()) }()
			}
		}
	}
	rows, err := s.db.Query(`SELECT key,value FROM settings`)
	if err != nil {
		s.fail(w, 500, "settings", err.Error())
		return
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			s.fail(w, 500, "settings", err.Error())
			return
		}
		m[k] = v
	}
	if err := rows.Err(); err != nil {
		s.fail(w, 500, "settings", err.Error())
		return
	}
	s.json(w, map[string]any{"settings": m})
}
func (s *Server) startScan(w http.ResponseWriter) {
	s.mu.Lock()
	if s.scanCancel != nil {
		s.mu.Unlock()
		s.fail(w, 409, "scan_running", "scan already running")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.scanCancel = cancel
	s.scan = Progress{Scanning: true}
	s.mu.Unlock()
	go func() {
		defer func() { s.mu.Lock(); s.scan.Scanning = false; s.scanCancel = nil; s.mu.Unlock() }()
		_, err := s.provider.Scan(ctx, func(p ScanProgress) {
			s.mu.Lock()
			s.scan.Found = p.Found
			s.scan.Total = p.Total
			s.scan.Current = p.Current
			s.mu.Unlock()
		})
		if err != nil && err != context.Canceled {
			s.mu.Lock()
			s.scan.Error = err.Error()
			s.mu.Unlock()
		}
	}()
	s.json(w, map[string]bool{"started": true})
}
func (s *Server) importTrack(path string, info os.FileInfo) {
	var title, artist, album, genre string
	var year int
	var duration float64
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration:format_tags=title,artist,album,genre,date,year", "-of", "json", path)
	out, e := cmd.Output()
	if e == nil {
		var data struct {
			Format struct {
				Duration string            `json:"duration"`
				Tags     map[string]string `json:"tags"`
			} `json:"format"`
		}
		if json.Unmarshal(out, &data) == nil {
			duration, _ = strconv.ParseFloat(data.Format.Duration, 64)
			for k, v := range data.Format.Tags {
				switch strings.ToLower(k) {
				case "title":
					title = v
				case "artist":
					artist = v
				case "album":
					album = v
				case "genre":
					genre = v
				case "date", "year":
					if len(v) >= 4 {
						year, _ = strconv.Atoi(v[:4])
					}
				}
			}
		}
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = "Unknown Album"
	}
	s.db.Exec(`INSERT OR IGNORE INTO artists(name) VALUES(?)`, artist)
	var aid int
	s.db.QueryRow(`SELECT id FROM artists WHERE name=?`, artist).Scan(&aid)
	s.db.Exec(`INSERT INTO albums(title,artist_id,year) SELECT ?,?,? WHERE NOT EXISTS(SELECT 1 FROM albums WHERE title=? AND artist_id=?)`, album, aid, year, album, aid)
	var alid int
	s.db.QueryRow(`SELECT id FROM albums WHERE title=? AND artist_id=?`, album, aid).Scan(&alid)
	id := fmt.Sprintf("%x", []byte(path))
	tx, e := s.db.Begin()
	if e != nil {
		return
	}
	defer tx.Rollback()
	var oldID string
	tx.QueryRow(`SELECT id FROM tracks WHERE path=?`, path).Scan(&oldID)
	if oldID != "" {
		tx.Exec(`DELETE FROM tracks_fts WHERE track_id=?`, oldID)
	}
	tx.Exec(`INSERT INTO tracks(id,title,artist_id,album_id,genre,year,duration,path,mtime,size) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(path) DO UPDATE SET id=excluded.id,title=excluded.title,artist_id=excluded.artist_id,album_id=excluded.album_id,genre=excluded.genre,year=excluded.year,duration=excluded.duration,mtime=excluded.mtime,size=excluded.size`, id, title, aid, alid, genre, year, duration, path, info.ModTime().UnixNano(), info.Size())
	tx.Exec(`INSERT INTO tracks_fts(track_id,title,artist,album,genre) VALUES(?,?,?,?,?)`, id, title, artist, album, genre)
	tx.Commit()
}
func (s *Server) media(w http.ResponseWriter, r *http.Request, id string) {
	t, e := s.track(id)
	if e != nil {
		s.fail(w, 404, "track_not_found", "track not found")
		return
	}
	f, e := os.Open(t.Path)
	if e != nil {
		s.fail(w, 404, "media_missing", "media unavailable")
		return
	}
	defer f.Close()
	http.ServeContent(w, r, filepath.Base(t.Path), time.Time{}, f)
}
func parseLRC(data string) []map[string]any {
	lines := []map[string]any{}
	for _, raw := range strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n") {
		text := raw
		times := []float64{}
		for strings.HasPrefix(text, "[") {
			end := strings.IndexByte(text, ']')
			if end < 0 {
				break
			}
			stamp := text[1:end]
			colon := strings.IndexByte(stamp, ':')
			if colon < 1 {
				break
			}
			minutes, err1 := strconv.Atoi(stamp[:colon])
			seconds, err2 := strconv.ParseFloat(stamp[colon+1:], 64)
			if err1 != nil || err2 != nil || minutes < 0 || seconds < 0 || seconds >= 60 {
				break
			}
			times = append(times, float64(minutes)*60+seconds)
			text = text[end+1:]
		}
		for _, at := range times {
			lines = append(lines, map[string]any{"time": at, "text": text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i]["time"].(float64) < lines[j]["time"].(float64) })
	return lines
}
func (s *Server) lyrics(w http.ResponseWriter, id string) {
	t, e := s.track(id)
	if e != nil {
		s.fail(w, 404, "track_not_found", "track not found")
		return
	}
	b, e := os.ReadFile(strings.TrimSuffix(t.Path, filepath.Ext(t.Path)) + ".lrc")
	if e != nil {
		cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format_tags:stream_tags", "-of", "json", t.Path)
		if output, err := cmd.Output(); err == nil {
			var metadata struct {
				Format struct {
					Tags map[string]string `json:"tags"`
				} `json:"format"`
				Streams []struct {
					Tags map[string]string `json:"tags"`
				} `json:"streams"`
			}
			if json.Unmarshal(output, &metadata) == nil {
				for key, value := range metadata.Format.Tags {
					if strings.EqualFold(key, "lyrics") || strings.EqualFold(key, "unsyncedlyrics") {
						b = []byte(value)
						break
					}
				}
				if len(b) == 0 {
					for _, stream := range metadata.Streams {
						for key, value := range stream.Tags {
							if strings.EqualFold(key, "lyrics") || strings.EqualFold(key, "unsyncedlyrics") {
								b = []byte(value)
								break
							}
						}
						if len(b) > 0 {
							break
						}
					}
				}
			}
		}
	}
	s.json(w, map[string]any{"lines": parseLRC(string(b))})
}
func (s *Server) playlists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var b struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		res, e := s.db.Exec(`INSERT INTO playlists(name) VALUES(?)`, b.Name)
		if e != nil {
			s.fail(w, 400, "playlist", e.Error())
			return
		}
		id, _ := res.LastInsertId()
		s.json(w, map[string]any{"id": id, "name": b.Name})
		return
	}
	rows, _ := s.db.Query(`SELECT id,name,created_at FROM playlists ORDER BY name`)
	defer rows.Close()
	o := []map[string]any{}
	for rows.Next() {
		var id int
		var n, c string
		rows.Scan(&id, &n, &c)
		o = append(o, map[string]any{"id": id, "name": n, "created_at": c})
	}
	s.json(w, map[string]any{"items": o})
}
func (s *Server) favorites(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Kind    string `json:"kind"`
		ID      string `json:"id"`
		TrackID string `json:"track_id"`
	}
	if r.Method == http.MethodPost {
		json.NewDecoder(r.Body).Decode(&b)
		if b.ID == "" {
			b.ID = b.TrackID
		}
		s.db.Exec(`INSERT OR IGNORE INTO favorites(kind,entity_id) VALUES(?,?)`, b.Kind, b.ID)
		s.json(w, map[string]bool{"ok": true})
		return
	}
	json.NewDecoder(r.Body).Decode(&b)
	s.db.Exec(`DELETE FROM favorites WHERE kind=? AND entity_id=?`, b.Kind, b.ID)
	w.WriteHeader(204)
}
