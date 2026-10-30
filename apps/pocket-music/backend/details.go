package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func decodeJSON(r *http.Request, value any) error { return json.NewDecoder(r.Body).Decode(value) }
func (s *Server) detail(w http.ResponseWriter, r *http.Request, kind, id string) {
	if kind == "albums" {
		rows, e := s.db.Query(`SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM albums al LEFT JOIN tracks t ON t.album_id=al.id LEFT JOIN artists ar ON ar.id=t.artist_id WHERE al.id=?`, id)
		if e != nil {
			s.fail(w, 404, "not_found", "album not found")
			return
		}
		defer rows.Close()
		tracks := []Track{}
		for rows.Next() {
			var t Track
			rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt)
			if t.ID != "" {
				tracks = append(tracks, t)
			}
		}
		var title, artist string
		s.db.QueryRow(`SELECT al.title,COALESCE(ar.name,'') FROM albums al LEFT JOIN artists ar ON ar.id=al.artist_id WHERE al.id=?`, id).Scan(&title, &artist)
		s.json(w, map[string]any{"id": id, "title": title, "artist": artist, "tracks": tracks})
		return
	}
	rows, e := s.db.Query(`SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM tracks t LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id WHERE t.artist_id=?`, id)
	if e != nil {
		s.fail(w, 404, "not_found", "artist not found")
		return
	}
	defer rows.Close()
	tracks := []Track{}
	for rows.Next() {
		var t Track
		rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt)
		tracks = append(tracks, t)
	}
	var name string
	s.db.QueryRow(`SELECT name FROM artists WHERE id=?`, id).Scan(&name)
	s.json(w, map[string]any{"id": id, "name": name, "tracks": tracks})
}
func (s *Server) playlistByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodDelete {
		s.db.Exec(`DELETE FROM playlist_items WHERE playlist_id=?`, id)
		s.db.Exec(`DELETE FROM playlists WHERE id=?`, id)
		w.WriteHeader(204)
		return
	}
	if r.Method == http.MethodPut {
		var b struct {
			Name string `json:"name"`
		}
		if jsonErr := decodeJSON(r, &b); jsonErr != nil {
			s.fail(w, 400, "invalid_json", jsonErr.Error())
			return
		}
		s.db.Exec(`UPDATE playlists SET name=? WHERE id=?`, b.Name, id)
	}
	rows, e := s.db.Query(`SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM playlist_items pi JOIN tracks t ON t.id=pi.track_id LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id WHERE pi.playlist_id=? ORDER BY pi.position`, id)
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	defer rows.Close()
	ts := []Track{}
	for rows.Next() {
		var t Track
		rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt)
		ts = append(ts, t)
	}
	var name string
	s.db.QueryRow(`SELECT name FROM playlists WHERE id=?`, id).Scan(&name)
	s.json(w, map[string]any{"id": id, "name": name, "tracks": ts})
}
func (s *Server) playlistItems(w http.ResponseWriter, r *http.Request, id string) {
	var b struct {
		Action  string `json:"action"`
		TrackID string `json:"track_id"`
		Index   int    `json:"index"`
		To      int    `json:"to"`
	}
	if e := decodeJSON(r, &b); e != nil {
		s.fail(w, 400, "invalid_json", e.Error())
		return
	}
	switch b.Action {
	case "add":
		var pos int
		s.db.QueryRow(`SELECT COALESCE(MAX(position)+1,0) FROM playlist_items WHERE playlist_id=?`, id).Scan(&pos)
		s.db.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, id, pos, b.TrackID)
	case "remove":
		tx, _ := s.db.Begin()
		tx.Exec(`DELETE FROM playlist_items WHERE playlist_id=? AND position=?`, id, b.Index)
		tx.Exec(`UPDATE playlist_items SET position=position-1 WHERE playlist_id=? AND position>?`, id, b.Index)
		tx.Commit()
	case "reorder":
		tx, _ := s.db.Begin()
		var tid string
		tx.QueryRow(`SELECT track_id FROM playlist_items WHERE playlist_id=? AND position=?`, id, b.Index).Scan(&tid)
		tx.Exec(`DELETE FROM playlist_items WHERE playlist_id=? AND position=?`, id, b.Index)
		tx.Exec(`UPDATE playlist_items SET position=position+100000 WHERE playlist_id=? AND position>=?`, id, b.To)
		tx.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, id, b.To, tid)
		tx.Exec(`UPDATE playlist_items SET position=position-99999 WHERE playlist_id=? AND position>=100000`, id)
		tx.Commit()
	default:
		s.fail(w, 400, "invalid_action", "action must be add, remove, or reorder")
		return
	}
	s.playlistByID(w, r, id)
}
func (s *Server) exportPlaylist(w http.ResponseWriter, id, format string) {
	rows, e := s.db.Query(`SELECT t.path,COALESCE(t.title,''),COALESCE(t.duration,0) FROM playlist_items pi JOIN tracks t ON t.id=pi.track_id WHERE pi.playlist_id=? ORDER BY pi.position`, id)
	if e != nil {
		s.fail(w, 404, "playlist_not_found", "playlist not found")
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "audio/x-mpegurl")
	w.Header().Set("Content-Disposition", `attachment; filename="playlist.`+format+`"`)
	fmt.Fprintln(w, "#EXTM3U")
	for rows.Next() {
		var path, title string
		var duration float64
		if e := rows.Scan(&path, &title, &duration); e != nil {
			s.fail(w, 500, "playlist", e.Error())
			return
		}
		fmt.Fprintf(w, "#EXTINF:%d,%s\n%s\n", int(duration), title, path)
	}
	if e := rows.Err(); e != nil {
		s.fail(w, 500, "playlist", e.Error())
	}
}
func (s *Server) importPlaylist(w http.ResponseWriter, r *http.Request) {
	f, header, e := r.FormFile("file")
	if e != nil {
		s.fail(w, 400, "missing_file", "multipart field file required")
		return
	}
	defer f.Close()
	name := r.FormValue("name")
	if name == "" {
		name = "Imported playlist"
	}
	scan := bufio.NewScanner(f)
	var paths []string
	for scan.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scan.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(filepath.Dir(header.Filename), filepath.FromSlash(line))
		}
		paths = append(paths, filepath.Clean(line))
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
	pid, e := res.LastInsertId()
	if e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	position := 0
	for _, path := range paths {
		var id string
		if e = tx.QueryRow(`SELECT id FROM tracks WHERE path=?`, path).Scan(&id); e != nil {
			continue
		}
		if _, e = tx.Exec(`INSERT INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, pid, position, id); e != nil {
			s.fail(w, 500, "playlist", e.Error())
			return
		}
		position++
	}
	if e = tx.Commit(); e != nil {
		s.fail(w, 500, "playlist", e.Error())
		return
	}
	s.json(w, map[string]any{"id": pid, "name": name, "count": position})
}
func (s *Server) cover(w http.ResponseWriter, r *http.Request, id string) {
	var path string
	e := s.db.QueryRow(`SELECT COALESCE(al.cover,'') FROM albums al WHERE al.id=?`, id).Scan(&path)
	if e != nil {
		s.fail(w, 404, "cover_not_found", "album not found")
		return
	}
	if path != "" {
		if f, err := os.Open(path); err == nil {
			defer f.Close()
			http.ServeContent(w, r, filepath.Base(path), time.Time{}, f)
			return
		}
	}
	var trackPath string
	s.db.QueryRow(`SELECT path FROM tracks WHERE album_id=? ORDER BY id LIMIT 1`, id).Scan(&trackPath)
	for _, candidate := range []string{filepath.Join(filepath.Dir(trackPath), "folder.jpg"), filepath.Join(filepath.Dir(trackPath), "cover.jpg")} {
		if f, err := os.Open(candidate); err == nil {
			defer f.Close()
			http.ServeContent(w, r, filepath.Base(candidate), time.Time{}, f)
			return
		}
	}
	if trackPath != "" {
		tmp := filepath.Join(os.TempDir(), "musicd-cover-"+id+".jpg")
		if err := exec.Command("ffmpeg", "-y", "-v", "error", "-i", trackPath, "-an", "-frames:v", "1", tmp).Run(); err == nil {
			defer os.Remove(tmp)
			http.ServeFile(w, r, tmp)
			return
		}
	}
	http.NotFound(w, r)
}
