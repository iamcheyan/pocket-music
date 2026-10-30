package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Track is the public representation of a library item. AlbumID and ArtistID
// are exposed so clients can request cover art and navigate to detail views
// without re-querying by name.
type Track struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	Genre      string  `json:"genre"`
	Year       int     `json:"year"`
	Duration   float64 `json:"duration"`
	Path       string  `json:"path,omitempty"`
	AddedAt    string  `json:"added_at"`
	AlbumID    int     `json:"album_id"`
	ArtistID   int     `json:"artist_id"`
	Favorite   bool    `json:"favorite"`
	PlayCount  int     `json:"play_count"`
	SkipCount  int     `json:"skip_count"`
	TrackNo    int     `json:"track_number"`
	LyricsHint bool    `json:"has_lyrics"`
}

// Progress reports a scan in flight.
type Progress struct {
	Scanning bool   `json:"scanning"`
	Found    int    `json:"found"`
	Total    int    `json:"total"`
	Skipped  int    `json:"skipped"`
	Current  string `json:"current,omitempty"`
	Error    string `json:"error,omitempty"`
	Done     bool   `json:"done"`
}

// Playback modes supported by the queue engine.
const (
	ModeNormal    = "normal"
	ModeRepeatAll = "repeat_all"
	ModeRepeatOne = "repeat_one"
	ModeShuffle   = "shuffle"
)

var playbackModes = map[string]bool{
	ModeNormal:    true,
	ModeRepeatAll: true,
	ModeRepeatOne: true,
	ModeShuffle:   true,
}

// Player is the serialised playback state returned by the service.
type Player struct {
	Track    *Track  `json:"track"`
	Playing  bool    `json:"playing"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Volume   float64 `json:"volume"`
	Mode     string  `json:"mode"`
	Queue    []Track `json:"queue"`
	Source   string  `json:"source"`
}

// Server owns the database, the provider and the playback engine. Every API
// handler is a method on it, so the service stays usable as a library in tests.
type Server struct {
	db             *sql.DB
	root           string
	cacheDir       string
	webDir         string
	mock           bool
	provider       MusicProvider
	mu             sync.Mutex
	scan           Progress
	scanCancel     context.CancelFunc
	playerMu       sync.Mutex
	player         Player
	lastTick       time.Time
	trackCompleted bool
	mpv            *mpvClient
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
	log.Printf("musicd listening on %s (music=%s mock=%v)", addr, s.root, s.mock)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}

func NewServer() (*Server, error) {
	db, e := sql.Open("sqlite", env("MUSICD_DB", "musicd.db"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	cacheDir := env("MUSICD_CACHE", filepath.Join(os.TempDir(), "musicd-cache"))
	s := &Server{
		db:       db,
		root:     env("MUSICD_MUSIC_DIR", filepath.Join(os.Getenv("HOME"), "Music")),
		cacheDir: cacheDir,
		webDir:   env("MUSICD_WEB_DIR", ""),
		mock:     strings.EqualFold(env("MUSICD_MOCK", "false"), "true"),
		player:   Player{Volume: 100, Mode: ModeNormal, Queue: []Track{}, Source: "local"},
		mpv:      newMPV(),
	}
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

const schemaVersion = 2

func (s *Server) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL)`,
		`INSERT INTO schema_version SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM schema_version)`,
		`CREATE TABLE IF NOT EXISTS artists(id INTEGER PRIMARY KEY,name TEXT UNIQUE NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS albums(id INTEGER PRIMARY KEY,title TEXT NOT NULL,artist_id INTEGER,year INTEGER,cover TEXT,UNIQUE(title,artist_id))`,
		`CREATE TABLE IF NOT EXISTS tracks(id TEXT PRIMARY KEY,title TEXT,artist_id INTEGER,album_id INTEGER,genre TEXT,year INTEGER,duration REAL,path TEXT UNIQUE NOT NULL,mtime INTEGER,size INTEGER,track_no INTEGER DEFAULT 0,added_at TEXT DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS tracks_title_idx ON tracks(title)`,
		`CREATE INDEX IF NOT EXISTS tracks_artist_idx ON tracks(artist_id)`,
		`CREATE INDEX IF NOT EXISTS tracks_album_idx ON tracks(album_id)`,
		`CREATE INDEX IF NOT EXISTS tracks_added_idx ON tracks(added_at)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(track_id UNINDEXED,title,artist,album,genre)`,
		`CREATE TABLE IF NOT EXISTS playlists(id INTEGER PRIMARY KEY,name TEXT NOT NULL,created_at TEXT DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS playlist_items(playlist_id INTEGER,position INTEGER,track_id TEXT,PRIMARY KEY(playlist_id,position))`,
		`CREATE TABLE IF NOT EXISTS favorites(kind TEXT,entity_id TEXT,created_at TEXT DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(kind,entity_id))`,
		`CREATE TABLE IF NOT EXISTS history(track_id TEXT PRIMARY KEY,last_played TEXT,play_count INTEGER DEFAULT 0,skip_count INTEGER DEFAULT 0,completion_rate REAL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT)`,
	}
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
	// Additive migrations for databases created by earlier builds.
	for _, q := range []string{`ALTER TABLE tracks ADD COLUMN track_no INTEGER DEFAULT 0`, `ALTER TABLE tracks ADD COLUMN mtime INTEGER`, `ALTER TABLE tracks ADD COLUMN size INTEGER`} {
		tx.Exec(q) // duplicate-column errors are expected on fresh databases
	}
	if _, e = tx.Exec(`DELETE FROM tracks_fts WHERE track_id NOT IN (SELECT id FROM tracks)`); e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE schema_version SET version=? WHERE version<?`, schemaVersion, schemaVersion); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	defaults := map[string]string{
		"language": "en", "theme": "system", "music_folder": s.root,
		"cache_size": "512", "auto_scan": "false", "audio_output": "null", "proxy": "",
	}
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
		go func() { s.startScan(nil) }()
	}
	return nil
}

func (s *Server) setting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&value)
	return value, err
}

func (s *Server) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/", s.api)
	m.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		s.json(w, map[string]any{"ok": true, "mock": s.mock})
	})
	if s.webDir != "" {
		m.HandleFunc("/", s.static)
	}
	return m
}

// static serves the built frontend for single-binary desktop deployments and
// falls back to index.html so client-side routes survive a reload.
func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.webDir, filepath.Clean("/"+r.URL.Path))
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	index := filepath.Join(s.webDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, index)
}

func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(strings.Trim(p, "/"), "/")
	switch {
	case p == "health":
		s.health(w)
	case p == "home":
		s.home(w)
	case p == "library/songs":
		s.songs(w, r)
	case p == "library/albums", p == "library/artists", p == "library/genres", p == "library/folders":
		s.collections(w, r, p)
	case p == "search":
		s.search(w, r)
	case p == "player/state":
		s.playerState(w)
	case p == "player/mode":
		s.setMode(w, r)
	case p == "player/pause", p == "player/resume", p == "player/next", p == "player/prev", p == "player/seek", p == "player/volume":
		s.playerAction(w, r, p)
	case p == "play":
		s.play(w, r)
	case p == "queue":
		s.queue(w, r)
	case p == "settings":
		s.settings(w, r)
	case p == "scan":
		s.scanRoute(w, r)
	case p == "scan/status":
		s.scanStatus(w)
	case p == "scan/cancel":
		s.scanCancelRoute(w, r)
	case p == "playlists":
		s.playlists(w, r)
	case p == "playlists/import":
		s.importPlaylist(w, r)
	case p == "favorites":
		s.favorites(w, r)
	case p == "stats":
		s.stats(w)
	case len(parts) == 2 && parts[0] == "tracks":
		s.trackRoute(w, parts[1])
	case len(parts) == 2 && parts[0] == "media":
		s.media(w, r, parts[1])
	case len(parts) == 2 && parts[0] == "lyrics":
		s.lyrics(w, parts[1])
	case len(parts) == 2 && parts[0] == "cover":
		s.cover(w, r, parts[1])
	case len(parts) == 2 && (parts[0] == "albums" || parts[0] == "artists"):
		s.detail(w, r, parts[0], parts[1])
	case len(parts) == 2 && parts[0] == "playlists":
		s.playlistByID(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "playlists" && parts[2] == "items":
		s.playlistItems(w, r, parts[1])
	case len(parts) == 3 && parts[0] == "playlists" && parts[2] == "export":
		s.exportPlaylist(w, parts[1], r.URL.Query().Get("format"))
	default:
		s.fail(w, 404, "not_found", "unknown endpoint")
	}
}

func (s *Server) json(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) fail(w http.ResponseWriter, status int, code, msg string) {
	w.WriteHeader(status)
	s.json(w, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func (s *Server) health(w http.ResponseWriter) {
	providerName := "local"
	if s.provider != nil {
		providerName = s.provider.Name()
	}
	var tracks, albums, artists int
	s.db.QueryRow(`SELECT count(*) FROM tracks`).Scan(&tracks)
	s.db.QueryRow(`SELECT count(*) FROM albums`).Scan(&albums)
	s.db.QueryRow(`SELECT count(*) FROM artists`).Scan(&artists)
	s.json(w, map[string]any{
		"provider": providerName,
		"mock":     s.mock,
		"root":     s.root,
		"library":  map[string]int{"tracks": tracks, "albums": albums, "artists": artists},
		"audio": map[string]any{
			"status": "ready", "ao": s.mpv.outputName(), "engine": "mpv",
		},
	})
}
