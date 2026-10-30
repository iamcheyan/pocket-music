package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) detail(w http.ResponseWriter, r *http.Request, kind, id string) {
	switch kind {
	case "albums":
		var title, artist, cover string
		var artistID, year, fav int
		e := s.db.QueryRow(`SELECT al.title,COALESCE(ar.name,''),COALESCE(al.artist_id,0),COALESCE(al.year,0),
			CASE WHEN f.entity_id IS NULL THEN 0 ELSE 1 END
			FROM albums al LEFT JOIN artists ar ON ar.id=al.artist_id
			LEFT JOIN favorites f ON f.kind='album' AND f.entity_id=CAST(al.id AS TEXT)
			WHERE al.id=?`, id).Scan(&title, &artist, &artistID, &year, &fav)
		if e != nil {
			s.fail(w, 404, "not_found", "album not found")
			return
		}
		cover = "/api/cover/" + id
		tracks, e := s.queryTracks(`WHERE t.album_id=? ORDER BY COALESCE(t.track_no,0),t.title`, id)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		var duration float64
		s.db.QueryRow(`SELECT COALESCE(SUM(duration),0) FROM tracks WHERE album_id=?`, id).Scan(&duration)
		s.json(w, map[string]any{"id": id, "title": title, "artist": artist, "artist_id": artistID,
			"year": year, "cover": cover, "favorite": fav == 1, "tracks": tracks, "duration": duration})
	case "artists":
		var name string
		var fav int
		e := s.db.QueryRow(`SELECT ar.name, CASE WHEN f.entity_id IS NULL THEN 0 ELSE 1 END
			FROM artists ar LEFT JOIN favorites f ON f.kind='artist' AND f.entity_id=CAST(ar.id AS TEXT)
			WHERE ar.id=?`, id).Scan(&name, &fav)
		if e != nil {
			s.fail(w, 404, "not_found", "artist not found")
			return
		}
		tracks, e := s.queryTracks(`WHERE t.artist_id=? ORDER BY COALESCE(al.year,0) DESC,COALESCE(t.track_no,0),t.title`, id)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		albums := []map[string]any{}
		rows, e := s.db.Query(`SELECT al.id,al.title,COALESCE(al.year,0),count(t.id) FROM albums al
			LEFT JOIN tracks t ON t.album_id=al.id WHERE al.artist_id=? GROUP BY al.id ORDER BY al.year DESC,al.title`, id)
		if e == nil {
			for rows.Next() {
				var aid, year, count int
				var title string
				rows.Scan(&aid, &title, &year, &count)
				albums = append(albums, map[string]any{"id": aid, "title": title, "year": year, "track_count": count, "cover": fmt.Sprintf("/api/cover/%d", aid)})
			}
			rows.Close()
		}
		s.json(w, map[string]any{"id": id, "name": name, "favorite": fav == 1, "tracks": tracks, "albums": albums})
	}
}

var favoriteKinds = map[string]bool{"track": true, "album": true, "artist": true}

func (s *Server) favoriteTracks(limit int) ([]Track, error) {
	return s.queryTracks(`WHERE fv.entity_id IS NOT NULL ORDER BY fv.created_at DESC,t.title LIMIT ?`, limit)
}

func (s *Server) favoriteAlbums(limit int) ([]map[string]any, error) {
	rows, e := s.db.Query(`SELECT al.id,al.title,COALESCE(ar.name,''),COALESCE(al.year,0),count(t.id)
		FROM favorites f JOIN albums al ON CAST(al.id AS TEXT)=f.entity_id
		LEFT JOIN artists ar ON ar.id=al.artist_id
		LEFT JOIN tracks t ON t.album_id=al.id
		WHERE f.kind='album' GROUP BY al.id ORDER BY f.created_at DESC LIMIT ?`, limit)
	if e != nil {
		return []map[string]any{}, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, year, count int
		var title, artist string
		rows.Scan(&id, &title, &artist, &year, &count)
		out = append(out, map[string]any{"id": id, "title": title, "artist": artist, "year": year, "track_count": count, "cover": fmt.Sprintf("/api/cover/%d", id)})
	}
	return out, rows.Err()
}

func (s *Server) favorites(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Kind    string `json:"kind"`
		ID      string `json:"id"`
		TrackID string `json:"track_id"`
	}
	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		if r.Method == http.MethodDelete {
			json.NewDecoder(r.Body).Decode(&b)
		}
		if b.Kind == "" {
			b.Kind = r.URL.Query().Get("kind")
		}
		if b.ID == "" {
			b.ID = r.URL.Query().Get("id")
		}
	default:
		if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
			s.fail(w, 400, "invalid_json", e.Error())
			return
		}
	}
	if b.ID == "" {
		b.ID = b.TrackID
	}
	if b.Kind != "" && !favoriteKinds[b.Kind] {
		s.fail(w, 400, "invalid_kind", "kind must be track, album or artist")
		return
	}
	switch r.Method {
	case http.MethodGet:
		tracks, _ := s.favoriteTracks(500)
		albums, _ := s.favoriteAlbums(500)
		artists := []map[string]any{}
		rows, e := s.db.Query(`SELECT ar.id,ar.name,count(t.id) FROM favorites f
			JOIN artists ar ON CAST(ar.id AS TEXT)=f.entity_id
			LEFT JOIN tracks t ON t.artist_id=ar.id WHERE f.kind='artist' GROUP BY ar.id ORDER BY ar.name`)
		if e == nil {
			for rows.Next() {
				var id, count int
				var name string
				rows.Scan(&id, &name, &count)
				artists = append(artists, map[string]any{"id": id, "name": name, "track_count": count, "favorite": true})
			}
			rows.Close()
		}
		ids := map[string][]string{"track": {}, "album": {}, "artist": {}}
		rows, e = s.db.Query(`SELECT kind,entity_id FROM favorites`)
		if e == nil {
			for rows.Next() {
				var kind, entity string
				rows.Scan(&kind, &entity)
				ids[kind] = append(ids[kind], entity)
			}
			rows.Close()
		}
		s.json(w, map[string]any{"tracks": tracks, "albums": albums, "artists": artists, "ids": ids})
	case http.MethodPost, http.MethodPut:
		if b.Kind == "" || b.ID == "" {
			s.fail(w, 400, "invalid_request", "kind and id are required")
			return
		}
		if !s.entityExists(b.Kind, b.ID) {
			s.fail(w, 404, "not_found", b.Kind+" not found")
			return
		}
		if _, e := s.db.Exec(`INSERT OR IGNORE INTO favorites(kind,entity_id) VALUES(?,?)`, b.Kind, b.ID); e != nil {
			s.fail(w, 500, "favorites", e.Error())
			return
		}
		s.json(w, map[string]any{"kind": b.Kind, "id": b.ID, "favorite": true})
	case http.MethodDelete:
		if b.Kind == "" || b.ID == "" {
			s.fail(w, 400, "invalid_request", "kind and id are required")
			return
		}
		s.db.Exec(`DELETE FROM favorites WHERE kind=? AND entity_id=?`, b.Kind, b.ID)
		s.json(w, map[string]any{"kind": b.Kind, "id": b.ID, "favorite": false})
	default:
		s.fail(w, 405, "method_not_allowed", "GET, POST, PUT or DELETE")
	}
}

func (s *Server) entityExists(kind, id string) bool {
	var n int
	switch kind {
	case "track":
		s.db.QueryRow(`SELECT count(*) FROM tracks WHERE id=?`, id).Scan(&n)
	case "album":
		s.db.QueryRow(`SELECT count(*) FROM albums WHERE id=?`, id).Scan(&n)
	case "artist":
		s.db.QueryRow(`SELECT count(*) FROM artists WHERE id=?`, id).Scan(&n)
	}
	return n > 0
}

func (s *Server) stats(w http.ResponseWriter) {
	var tracks, albums, artists, playlists int
	s.db.QueryRow(`SELECT count(*) FROM tracks`).Scan(&tracks)
	s.db.QueryRow(`SELECT count(*) FROM albums`).Scan(&albums)
	s.db.QueryRow(`SELECT count(*) FROM artists`).Scan(&artists)
	s.db.QueryRow(`SELECT count(*) FROM playlists`).Scan(&playlists)
	var playTime float64
	var plays, skips int
	var completion float64
	s.db.QueryRow(`SELECT COALESCE(SUM(t.duration*h.play_count),0), COALESCE(SUM(h.play_count),0),
		COALESCE(SUM(h.skip_count),0), COALESCE(AVG(h.completion_rate),0)
		FROM tracks t JOIN history h ON h.track_id=t.id`).Scan(&playTime, &plays, &skips, &completion)
	topArtists := []map[string]any{}
	rows, e := s.db.Query(`SELECT ar.id,ar.name,COALESCE(SUM(h.play_count),0) AS plays FROM artists ar
		JOIN tracks t ON t.artist_id=ar.id JOIN history h ON h.track_id=t.id
		GROUP BY ar.id ORDER BY plays DESC,ar.name LIMIT 10`)
	if e == nil {
		for rows.Next() {
			var id, count int
			var name string
			rows.Scan(&id, &name, &count)
			topArtists = append(topArtists, map[string]any{"id": id, "name": name, "play_count": count})
		}
		rows.Close()
	}
	topAlbums := []map[string]any{}
	rows, e = s.db.Query(`SELECT al.id,al.title,COALESCE(ar.name,''),COALESCE(SUM(h.play_count),0) AS plays
		FROM albums al LEFT JOIN artists ar ON ar.id=al.artist_id
		JOIN tracks t ON t.album_id=al.id JOIN history h ON h.track_id=t.id
		GROUP BY al.id ORDER BY plays DESC,al.title LIMIT 10`)
	if e == nil {
		for rows.Next() {
			var id, count int
			var title, artist string
			rows.Scan(&id, &title, &artist, &count)
			topAlbums = append(topAlbums, map[string]any{"id": id, "title": title, "artist": artist, "play_count": count, "cover": fmt.Sprintf("/api/cover/%d", id)})
		}
		rows.Close()
	}
	s.json(w, map[string]any{
		"library":      map[string]int{"tracks": tracks, "albums": albums, "artists": artists, "playlists": playlists},
		"play_time":    playTime,
		"play_count":   plays,
		"skip_count":   skips,
		"completion":   completion,
		"most_played":  mustList(s, "play_count", 10),
		"most_skipped": mustList(s, "skip_count", 5),
		"top_artists":  topArtists,
		"top_albums":   topAlbums,
	})
}

var allowedSettings = map[string]func(string) bool{
	"language":     oneOf("en", "zh", "ja"),
	"theme":        oneOf("system", "light", "dark"),
	"music_folder": nonEmpty,
	"cache_size":   numericRange(16, 32768),
	"auto_scan":    oneOf("true", "false"),
	"audio_output": oneOf("null", "pulse", "alsa", "pipewire", "jack"),
	"proxy":        func(string) bool { return true },
}

func oneOf(values ...string) func(string) bool {
	return func(v string) bool {
		for _, allowed := range values {
			if v == allowed {
				return true
			}
		}
		return false
	}
}

func nonEmpty(v string) bool { return strings.TrimSpace(v) != "" }

func numericRange(min, max int) func(string) bool {
	return func(v string) bool {
		n, e := strconv.Atoi(v)
		return e == nil && n >= min && n <= max
	}
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		var b struct {
			Settings map[string]string `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			s.fail(w, 400, "invalid_json", err.Error())
			return
		}
		for k, v := range b.Settings {
			validate, ok := allowedSettings[k]
			if !ok {
				s.fail(w, 400, "invalid_setting", "unsupported setting: "+k)
				return
			}
			if !validate(v) {
				s.fail(w, 400, "invalid_value", "invalid value for "+k)
				return
			}
		}
		for k, v := range b.Settings {
			if _, err := s.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, k, v); err != nil {
				s.fail(w, 500, "settings", err.Error())
				return
			}
			switch k {
			case "music_folder":
				s.root = v
			case "audio_output":
				s.mpv.setOutput(v)
			case "auto_scan":
				if strings.EqualFold(v, "true") {
					go s.startScan(nil)
				}
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
	s.json(w, map[string]any{"settings": m, "root": s.root})
}
