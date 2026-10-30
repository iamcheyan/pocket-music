package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) playlists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var b struct {
			Name  string   `json:"name"`
			Track []string `json:"track_ids"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		name := strings.TrimSpace(b.Name)
		if name == "" {
			s.fail(w, 400, "invalid_name", "playlist name is required")
			return
		}
		res, e := s.db.Exec(`INSERT INTO playlists(name) VALUES(?)`, name)
		if e != nil {
			s.fail(w, 400, "playlist", e.Error())
			return
		}
		id, _ := res.LastInsertId()
		for i, trackID := range b.Track {
			s.db.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, id, i, trackID)
		}
		s.json(w, map[string]any{"id": id, "name": name, "track_count": len(b.Track)})
		return
	}
	rows, e := s.db.Query(`SELECT p.id,p.name,p.created_at,
		(SELECT count(*) FROM playlist_items pi WHERE pi.playlist_id=p.id)
		FROM playlists p ORDER BY p.created_at DESC,p.name`)
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, count int
		var name, created string
		rows.Scan(&id, &name, &created, &count)
		out = append(out, map[string]any{"id": id, "name": name, "created_at": created, "track_count": count})
	}
	s.json(w, map[string]any{"items": out, "total": len(out)})
}

func (s *Server) playlistByID(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodDelete:
		s.db.Exec(`DELETE FROM playlist_items WHERE playlist_id=?`, id)
		s.db.Exec(`DELETE FROM playlists WHERE id=?`, id)
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodPut, http.MethodPost:
		var b struct {
			Name string `json:"name"`
		}
		if e := decodeBody(r, &b); e != nil {
			s.fail(w, 400, "invalid_json", e.Error())
			return
		}
		if strings.TrimSpace(b.Name) == "" {
			s.fail(w, 400, "invalid_name", "playlist name is required")
			return
		}
		s.db.Exec(`UPDATE playlists SET name=? WHERE id=?`, strings.TrimSpace(b.Name), id)
	}
	s.writePlaylist(w, id)
}

// writePlaylist renders a playlist with its ordered tracks. It is separated
// from playlistByID so mutations can render a response after consuming the
// request body.
func (s *Server) writePlaylist(w http.ResponseWriter, id string) {
	var name string
	if e := s.db.QueryRow(`SELECT name FROM playlists WHERE id=?`, id).Scan(&name); e != nil {
		s.fail(w, 404, "not_found", "playlist not found")
		return
	}
	tracks, e := s.queryTracks(`JOIN playlist_items pi ON pi.track_id=t.id WHERE pi.playlist_id=? ORDER BY pi.position`, id)
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	var duration float64
	s.db.QueryRow(`SELECT COALESCE(SUM(t.duration),0) FROM tracks t JOIN playlist_items pi ON pi.track_id=t.id WHERE pi.playlist_id=?`, id).Scan(&duration)
	s.json(w, map[string]any{"id": id, "name": name, "tracks": tracks, "track_count": len(tracks), "duration": duration})
}

func (s *Server) playlistItems(w http.ResponseWriter, r *http.Request, id string) {
	var b struct {
		Action   string   `json:"action"`
		TrackID  string   `json:"track_id"`
		TrackIDs []string `json:"track_ids"`
		Index    int      `json:"index"`
		To       int      `json:"to"`
	}
	if e := decodeBody(r, &b); e != nil {
		s.fail(w, 400, "invalid_json", e.Error())
		return
	}
	switch b.Action {
	case "add":
		ids := b.TrackIDs
		if b.TrackID != "" {
			ids = append(ids, b.TrackID)
		}
		for _, trackID := range ids {
			var pos int
			s.db.QueryRow(`SELECT COALESCE(MAX(position)+1,0) FROM playlist_items WHERE playlist_id=?`, id).Scan(&pos)
			s.db.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, id, pos, trackID)
		}
	case "remove":
		tx, e := s.db.Begin()
		if e != nil {
			s.fail(w, 500, "playlist", e.Error())
			return
		}
		tx.Exec(`DELETE FROM playlist_items WHERE playlist_id=? AND position=?`, id, b.Index)
		tx.Exec(`UPDATE playlist_items SET position=position-1 WHERE playlist_id=? AND position>?`, id, b.Index)
		tx.Commit()
	case "reorder":
		tx, e := s.db.Begin()
		if e != nil {
			s.fail(w, 500, "playlist", e.Error())
			return
		}
		var trackID string
		tx.QueryRow(`SELECT track_id FROM playlist_items WHERE playlist_id=? AND position=?`, id, b.Index).Scan(&trackID)
		tx.Exec(`DELETE FROM playlist_items WHERE playlist_id=? AND position=?`, id, b.Index)
		tx.Exec(`UPDATE playlist_items SET position=position+100000 WHERE playlist_id=? AND position>=?`, id, b.To)
		tx.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, id, b.To, trackID)
		tx.Exec(`UPDATE playlist_items SET position=position-99999 WHERE playlist_id=? AND position>=100000`, id)
		tx.Commit()
	case "clear":
		s.db.Exec(`DELETE FROM playlist_items WHERE playlist_id=?`, id)
	default:
		s.fail(w, 400, "invalid_action", "action must be add, remove, reorder or clear")
		return
	}
	s.writePlaylist(w, id)
}

func (s *Server) exportPlaylist(w http.ResponseWriter, id, format string) {
	var name string
	if e := s.db.QueryRow(`SELECT name FROM playlists WHERE id=?`, id).Scan(&name); e != nil {
		s.fail(w, 404, "playlist_not_found", "playlist not found")
		return
	}
	if format != "m3u8" {
		format = "m3u"
	}
	rows, e := s.db.Query(`SELECT COALESCE(t.path,''),COALESCE(t.title,''),COALESCE(t.duration,0),COALESCE(ar.name,'')
		FROM playlist_items pi JOIN tracks t ON t.id=pi.track_id
		LEFT JOIN artists ar ON ar.id=t.artist_id
		WHERE pi.playlist_id=? ORDER BY pi.position`, id)
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	defer rows.Close()
	filename := strings.NewReplacer("\"", "", "/", "-").Replace(name)
	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.`+format+`"`)
	if format == "m3u8" {
		w.Write([]byte("\ufeff"))
	}
	fmt.Fprintln(w, "#EXTM3U")
	for rows.Next() {
		var path, title, artist string
		var duration float64
		if e := rows.Scan(&path, &title, &duration, &artist); e != nil {
			return
		}
		fmt.Fprintf(w, "#EXTINF:%d,%s - %s\n%s\n", int(duration), artist, title, path)
	}
}

func (s *Server) importPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.fail(w, 405, "method_not_allowed", "use POST")
		return
	}
	f, header, e := r.FormFile("file")
	if e != nil {
		s.fail(w, 400, "missing_file", "multipart field file required")
		return
	}
	defer f.Close()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}
	if name == "" {
		name = "Imported playlist"
	}
	scan := bufio.NewScanner(f)
	paths := []string{}
	for scan.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scan.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		paths = append(paths, playlistEntryPath(line, header.Filename))
	}
	if e := scan.Err(); e != nil {
		s.fail(w, 400, "invalid_playlist", e.Error())
		return
	}
	tx, e := s.db.Begin()
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	defer tx.Rollback()
	res, e := tx.Exec(`INSERT INTO playlists(name) VALUES(?)`, name)
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	pid, _ := res.LastInsertId()
	position, missing := 0, 0
	for _, path := range paths {
		var id string
		if e := tx.QueryRow(`SELECT id FROM tracks WHERE path=?`, path).Scan(&id); e != nil {
			// Entries are often relative to the playlist's own directory, which
			// an upload does not reveal. Match the tail of a library path too.
			rel := strings.TrimPrefix(filepath.ToSlash(path), "./")
			like := "%/" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(rel)
			if e := tx.QueryRow(`SELECT id FROM tracks WHERE path LIKE ? ESCAPE '\' LIMIT 1`, like).Scan(&id); e != nil {
				missing++
				continue
			}
		}
		if _, e := tx.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, pid, position, id); e != nil {
			s.fail(w, 500, "playlist", e.Error())
			return
		}
		position++
	}
	if e := tx.Commit(); e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	s.json(w, map[string]any{"id": pid, "name": name, "count": position, "missing": missing})
}

// playlistEntryPath normalises an M3U line: it decodes file:// URIs and percent
// escapes, then resolves relative entries against the playlist location.
func playlistEntryPath(line, sourceName string) string {
	if decoded, err := url.QueryUnescape(line); err == nil && strings.Contains(line, "%") {
		line = decoded
	}
	line = strings.TrimPrefix(line, "file://")
	line = filepath.FromSlash(line)
	if !filepath.IsAbs(line) {
		line = filepath.Join(filepath.Dir(sourceName), line)
	}
	return filepath.Clean(line)
}

// --- scan routes -----------------------------------------------------------

func (s *Server) scanRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.startScan(w)
		return
	}
	s.scanStatus(w)
}

func (s *Server) scanStatus(w http.ResponseWriter) {
	s.mu.Lock()
	status := s.scan
	if info, err := os.Stat(s.root); err == nil && !info.IsDir() {
		status.Error = "music folder is not a directory: " + s.root
	}
	s.mu.Unlock()
	s.json(w, status)
}

func (s *Server) scanCancelRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.fail(w, 405, "method_not_allowed", "use POST")
		return
	}
	s.mu.Lock()
	if s.scanCancel != nil {
		s.scanCancel()
	}
	s.mu.Unlock()
	s.json(w, map[string]bool{"cancelled": true})
}

func (s *Server) startScan(w http.ResponseWriter) {
	s.mu.Lock()
	if s.scanCancel != nil {
		s.mu.Unlock()
		if w != nil {
			s.fail(w, 409, "scan_running", "scan already running")
		}
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.scanCancel = cancel
	s.scan = Progress{Scanning: true}
	s.mu.Unlock()
	if w != nil {
		s.json(w, map[string]bool{"started": true})
	}
	go func() {
		result, err := s.provider.Scan(ctx, func(p ScanProgress) {
			s.mu.Lock()
			s.scan.Found = p.Found
			s.scan.Total = p.Total
			s.scan.Skipped = p.Found // replaced below by final result
			s.scan.Current = p.Current
			s.mu.Unlock()
		})
		s.mu.Lock()
		s.scan.Scanning = false
		s.scan.Done = err == nil
		s.scan.Skipped = result.Skipped
		s.scan.Found = result.Found
		s.scanCancel = nil
		if err != nil && err != context.Canceled {
			s.scan.Error = err.Error()
		}
		s.mu.Unlock()
	}()
}
