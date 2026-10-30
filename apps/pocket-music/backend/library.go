package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// The shared projection keeps every endpoint returning the same track shape.
const trackColumns = `t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,''),COALESCE(t.album_id,0),COALESCE(t.artist_id,0),CASE WHEN fv.entity_id IS NULL THEN 0 ELSE 1 END,COALESCE(h.play_count,0),COALESCE(h.skip_count,0),COALESCE(t.track_no,0)`

const trackFrom = ` FROM tracks t
	LEFT JOIN artists ar ON ar.id=t.artist_id
	LEFT JOIN albums al ON al.id=t.album_id
	LEFT JOIN favorites fv ON fv.kind='track' AND fv.entity_id=t.id
	LEFT JOIN history h ON h.track_id=t.id`

type rowScanner interface{ Scan(...any) error }

func scanTrack(r rowScanner) (Track, error) {
	var t Track
	var fav int
	e := r.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration,
		&t.Path, &t.AddedAt, &t.AlbumID, &t.ArtistID, &fav, &t.PlayCount, &t.SkipCount, &t.TrackNo)
	t.Favorite = fav == 1
	return t, e
}

func (s *Server) list(order string, n int) ([]Track, error) {
	q := `SELECT ` + trackColumns + trackFrom
	switch order {
	case "last_played":
		q += ` WHERE h.last_played IS NOT NULL ORDER BY h.last_played DESC`
	case "play_count":
		q += ` WHERE COALESCE(h.play_count,0)>0 ORDER BY h.play_count DESC,t.title`
	case "skip_count":
		q += ` WHERE COALESCE(h.skip_count,0)>0 ORDER BY h.skip_count DESC,t.title`
	default:
		q += ` ORDER BY t.added_at DESC,t.title`
	}
	rows, e := s.db.Query(q+` LIMIT ?`, n)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Track{}
	for rows.Next() {
		t, e := scanTrack(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func mustList(s *Server, o string, n int) []Track {
	x, e := s.list(o, n)
	if e != nil || x == nil {
		return []Track{}
	}
	return x
}

func (s *Server) home(w http.ResponseWriter) {
	favTracks, _ := s.favoriteTracks(8)
	favAlbums, _ := s.favoriteAlbums(8)
	s.json(w, map[string]any{
		"recently_added":  mustList(s, "added_at", 8),
		"recently_played": mustList(s, "last_played", 8),
		"most_played":     mustList(s, "play_count", 8),
		"favorite_tracks": favTracks,
		"favorite_albums": favAlbums,
	})
}

func pagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 50
	}
	return page, limit
}

func (s *Server) songs(w http.ResponseWriter, r *http.Request) {
	page, limit := pagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	conditions := []string{}
	args := []any{}
	if q != "" {
		conditions = append(conditions, `t.id IN (SELECT track_id FROM tracks_fts WHERE tracks_fts MATCH ?)`)
		args = append(args, ftsQuery(q))
	}
	if folder != "" && folder != "." && folder != "/" {
		// Real libraries carry absolute paths; demo rows are prefixed with
		// "mock:" so the artist/album tail needs its own pattern.
		conditions = append(conditions, `(t.path LIKE ? OR t.path LIKE ?)`)
		args = append(args, `%/`+folder+`/%`, `mock:`+folder+`/%`)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	s.db.QueryRow(`SELECT count(*) FROM tracks t`+where, args...).Scan(&total)
	listArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	rows, e := s.db.Query(`SELECT `+trackColumns+trackFrom+where+` ORDER BY t.title LIMIT ? OFFSET ?`, listArgs...)
	if e != nil {
		s.fail(w, 500, "query", e.Error())
		return
	}
	defer rows.Close()
	items := []Track{}
	for rows.Next() {
		t, e := scanTrack(rows)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		items = append(items, t)
	}
	s.json(w, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

// ftsQuery turns free text into a safe FTS5 MATCH expression: every word is
// quoted and prefix-matched, so punctuation in a user query cannot break the
// statement.
func ftsQuery(text string) string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\''
	})
	quoted := make([]string, 0, len(fields))
	for _, f := range fields {
		quoted = append(quoted, `"`+strings.ReplaceAll(f, `"`, `""`)+`"*`)
	}
	if len(quoted) == 0 {
		return `""`
	}
	return strings.Join(quoted, " ")
}

func (s *Server) collections(w http.ResponseWriter, r *http.Request, p string) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	like := "%" + q + "%"
	out := []map[string]any{}
	switch p {
	case "library/albums":
		rows, e := s.db.Query(`SELECT al.id,al.title,COALESCE(ar.name,''),COALESCE(al.year,0),
			(SELECT count(*) FROM tracks t WHERE t.album_id=al.id),
			CASE WHEN f.entity_id IS NULL THEN 0 ELSE 1 END
			FROM albums al
			LEFT JOIN artists ar ON ar.id=al.artist_id
			LEFT JOIN favorites f ON f.kind='album' AND f.entity_id=CAST(al.id AS TEXT)
			WHERE al.title LIKE ?
			ORDER BY al.title`, like)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, year, count, fav int
			var title, artist string
			rows.Scan(&id, &title, &artist, &year, &count, &fav)
			out = append(out, map[string]any{"id": id, "title": title, "artist": artist, "year": year, "track_count": count, "favorite": fav == 1, "cover": fmt.Sprintf("/api/cover/%d", id)})
		}
	case "library/artists":
		rows, e := s.db.Query(`SELECT ar.id,ar.name,
			(SELECT count(*) FROM tracks t WHERE t.artist_id=ar.id),
			CASE WHEN f.entity_id IS NULL THEN 0 ELSE 1 END
			FROM artists ar
			LEFT JOIN favorites f ON f.kind='artist' AND f.entity_id=CAST(ar.id AS TEXT)
			WHERE ar.name LIKE ?
			ORDER BY ar.name`, like)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, count, fav int
			var name string
			rows.Scan(&id, &name, &count, &fav)
			out = append(out, map[string]any{"id": id, "name": name, "track_count": count, "favorite": fav == 1})
		}
	case "library/genres":
		rows, e := s.db.Query(`SELECT genre,count(*) FROM tracks WHERE COALESCE(genre,'')<>'' GROUP BY genre ORDER BY genre`)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			var count int
			rows.Scan(&name, &count)
			out = append(out, map[string]any{"name": name, "track_count": count})
		}
	case "library/folders":
		rows, e := s.db.Query(`SELECT path FROM tracks ORDER BY path`)
		if e != nil {
			s.fail(w, 500, "query", e.Error())
			return
		}
		defer rows.Close()
		counts := map[string]int{}
		for rows.Next() {
			var path string
			rows.Scan(&path)
			dir := strings.TrimPrefix(filepath.Dir(path), "mock:")
			if s.root != "" && strings.HasPrefix(dir, s.root) {
				if rel, err := filepath.Rel(s.root, dir); err == nil {
					dir = rel
				}
			}
			counts[dir]++
		}
		names := make([]string, 0, len(counts))
		for name := range counts {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			out = append(out, map[string]any{"name": name, "track_count": counts[name]})
		}
	}
	s.json(w, map[string]any{"items": out, "total": len(out)})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kind := r.URL.Query().Get("type")
	want := func(k string) bool { return kind == "" || kind == "all" || kind == k }
	songs, albums, artists, playlists := []Track{}, []map[string]any{}, []map[string]any{}, []map[string]any{}
	if q != "" && want("song") {
		songs = s.searchSongs(q, 50)
	}
	if q != "" && want("album") {
		rows, e := s.db.Query(`SELECT id,title,COALESCE(year,0) FROM albums WHERE title LIKE ? ORDER BY title LIMIT 50`, "%"+q+"%")
		if e == nil {
			for rows.Next() {
				var id, year int
				var title string
				rows.Scan(&id, &title, &year)
				albums = append(albums, map[string]any{"id": id, "title": title, "year": year, "cover": fmt.Sprintf("/api/cover/%d", id)})
			}
			rows.Close()
		}
	}
	if q != "" && want("artist") {
		rows, e := s.db.Query(`SELECT id,name FROM artists WHERE name LIKE ? ORDER BY name LIMIT 50`, "%"+q+"%")
		if e == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				artists = append(artists, map[string]any{"id": id, "name": name})
			}
			rows.Close()
		}
	}
	if q != "" && want("playlist") {
		rows, e := s.db.Query(`SELECT id,name FROM playlists WHERE name LIKE ? ORDER BY name LIMIT 50`, "%"+q+"%")
		if e == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				playlists = append(playlists, map[string]any{"id": id, "name": name})
			}
			rows.Close()
		}
	}
	s.json(w, map[string]any{
		"query": q, "songs": songs, "albums": albums, "artists": artists, "playlists": playlists,
		"total": len(songs) + len(albums) + len(artists) + len(playlists),
	})
}

// searchSongs prefers the FTS index and falls back to a LIKE scan when the
// MATCH expression cannot be evaluated.
func (s *Server) searchSongs(q string, limit int) []Track {
	out := []Track{}
	// Search is a preview, so bound the ranked candidate set instead of joining
	// every match: a two-character query can hit the whole library.
	rows, e := s.db.Query(`SELECT `+trackColumns+trackFrom+` WHERE t.id IN (SELECT track_id FROM tracks_fts WHERE tracks_fts MATCH ? LIMIT 400) ORDER BY t.title LIMIT ?`, ftsQuery(q), limit)
	if e != nil {
		rows, e = s.db.Query(`SELECT `+trackColumns+trackFrom+` WHERE t.title LIKE ? OR ar.name LIKE ? OR al.title LIKE ? ORDER BY t.title LIMIT ?`, "%"+q+"%", "%"+q+"%", "%"+q+"%", limit)
		if e != nil {
			return out
		}
	}
	defer rows.Close()
	for rows.Next() {
		if t, err := scanTrack(rows); err == nil {
			out = append(out, t)
		}
	}
	return out
}

func (s *Server) listQuery(q Query) ([]Track, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if q.Text == "" {
		return s.queryTracks(`ORDER BY t.title LIMIT ? OFFSET ?`, limit, (page-1)*limit)
	}
	return s.queryTracks(`WHERE t.id IN (SELECT track_id FROM tracks_fts WHERE tracks_fts MATCH ?) ORDER BY t.title LIMIT ? OFFSET ?`,
		ftsQuery(q.Text), limit, (page-1)*limit)
}

func (s *Server) queryTracks(suffix string, args ...any) ([]Track, error) {
	rows, e := s.db.Query(`SELECT `+trackColumns+trackFrom+` `+suffix, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Track{}
	for rows.Next() {
		t, e := scanTrack(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Server) track(id string) (Track, error) {
	row := s.db.QueryRow(`SELECT `+trackColumns+trackFrom+` WHERE t.id=?`, id)
	return scanTrack(row)
}

func (s *Server) trackRoute(w http.ResponseWriter, id string) {
	t, e := s.track(id)
	if e == sql.ErrNoRows {
		s.fail(w, 404, "not_found", "track not found")
		return
	}
	if e != nil {
		s.fail(w, 500, "query", e.Error())
		return
	}
	t.LyricsHint = len(loadLyrics(t)) > 0
	s.json(w, t)
}

// scanProvider walks the configured root and imports changed files. Progress is
// reported through the callback so the UI can poll while a large library scans.
func (s *Server) scanProvider(ctx context.Context, progress func(ScanProgress)) (ScanResult, error) {
	paths := []string{}
	e := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != s.root {
				return filepath.SkipDir
			}
			return nil
		}
		if supportedAudio(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if e != nil {
		return ScanResult{}, e
	}
	sort.Strings(paths)
	result := ScanResult{}
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if e := ctx.Err(); e != nil {
			return result, e
		}
		seen[path] = true
		info, e := os.Stat(path)
		if e != nil {
			continue
		}
		var oldMtime, oldSize int64
		s.db.QueryRow(`SELECT COALESCE(mtime,0),COALESCE(size,0) FROM tracks WHERE path=?`, path).Scan(&oldMtime, &oldSize)
		if oldMtime == info.ModTime().UnixNano() && oldSize == info.Size() {
			result.Skipped++
		} else {
			s.importTrack(path, info)
			result.Updated++
		}
		result.Found++
		if progress != nil {
			progress(ScanProgress{Found: result.Found, Total: len(paths), Current: path})
		}
	}
	// Remove rows for files that disappeared from the library.
	rows, e := s.db.Query(`SELECT id,path FROM tracks`)
	if e == nil {
		missing := []string{}
		for rows.Next() {
			var id, path string
			rows.Scan(&id, &path)
			if !strings.HasPrefix(path, "mock:") && !seen[path] {
				missing = append(missing, id)
			}
		}
		rows.Close()
		for _, id := range missing {
			s.db.Exec(`DELETE FROM tracks_fts WHERE track_id=?`, id)
			s.db.Exec(`DELETE FROM playlist_items WHERE track_id=?`, id)
			s.db.Exec(`DELETE FROM tracks WHERE id=?`, id)
			result.Removed++
		}
	}
	return result, nil
}

func supportedAudio(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".flac", ".aac", ".m4a", ".ogg", ".wav", ".ape":
		return true
	}
	return false
}

type ffprobeOutput struct {
	Format struct {
		Duration string            `json:"duration"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		CodecType string            `json:"codec_type"`
		Tags      map[string]string `json:"tags"`
	} `json:"streams"`
}

func probe(path string) ffprobeOutput {
	var data ffprobeOutput
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json",
		"-show_entries", "format=duration:format_tags:stream=codec_type:stream_tags", path)
	out, e := cmd.Output()
	if e != nil {
		return data
	}
	json.Unmarshal(out, &data)
	return data
}

func tagValue(tags map[string]string, keys ...string) string {
	for _, want := range keys {
		for k, v := range tags {
			if strings.EqualFold(k, want) && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func (s *Server) importTrack(path string, info os.FileInfo) {
	data := probe(path)
	tags := data.Format.Tags
	title := tagValue(tags, "title")
	artist := tagValue(tags, "artist", "album_artist")
	album := tagValue(tags, "album")
	genre := tagValue(tags, "genre")
	year := 0
	if raw := tagValue(tags, "date", "year"); len(raw) >= 4 {
		year, _ = strconv.Atoi(raw[:4])
	}
	trackNo, _ := strconv.Atoi(tagValue(tags, "track", "tracknumber"))
	duration, _ := strconv.ParseFloat(data.Format.Duration, 64)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = filepath.Base(filepath.Dir(path))
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
	tx.Exec(`INSERT INTO tracks(id,title,artist_id,album_id,genre,year,duration,path,mtime,size,track_no)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET id=excluded.id,title=excluded.title,artist_id=excluded.artist_id,
			album_id=excluded.album_id,genre=excluded.genre,year=excluded.year,duration=excluded.duration,
			mtime=excluded.mtime,size=excluded.size,track_no=excluded.track_no`,
		id, title, aid, alid, genre, year, duration, path, info.ModTime().UnixNano(), info.Size(), trackNo)
	tx.Exec(`INSERT INTO tracks_fts(track_id,title,artist,album,genre) VALUES(?,?,?,?,?)`, id, title, artist, album, genre)
	tx.Commit()
}
