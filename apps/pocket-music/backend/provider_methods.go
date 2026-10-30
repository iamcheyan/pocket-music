package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) listQuery(q Query) ([]Track, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 500 {
		limit = 100
	}
	where := ""
	args := []any{}
	if q.Text != "" {
		where = `WHERE t.id IN (SELECT track_id FROM tracks_fts WHERE tracks_fts MATCH ?)`
		args = append(args, strings.ReplaceAll(q.Text, "'", ""))
	}
	args = append(args, limit, (page-1)*limit)
	rows, e := s.db.Query(`SELECT t.id,COALESCE(t.title,''),COALESCE(ar.name,''),COALESCE(al.title,''),COALESCE(t.genre,''),COALESCE(t.year,0),COALESCE(t.duration,0),COALESCE(t.path,''),COALESCE(t.added_at,'') FROM tracks t LEFT JOIN artists ar ON ar.id=t.artist_id LEFT JOIN albums al ON al.id=t.album_id `+where+` ORDER BY t.title LIMIT ? OFFSET ?`, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Track{}
	for rows.Next() {
		var t Track
		if e := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.Genre, &t.Year, &t.Duration, &t.Path, &t.AddedAt); e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *Server) scanProvider(ctx context.Context, progress func(ScanProgress)) (ScanResult, error) {
	paths := []string{}
	e := filepath.WalkDir(s.root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".mp3", ".flac", ".aac", ".m4a", ".ogg", ".wav", ".ape":
			paths = append(paths, path)
		}
		return nil
	})
	if e != nil {
		return ScanResult{}, e
	}
	result := ScanResult{}
	for _, path := range paths {
		if e := ctx.Err(); e != nil {
			return result, e
		}
		info, e := os.Stat(path)
		if e != nil {
			continue
		}
		var oldMtime, oldSize int64
		s.db.QueryRow(`SELECT mtime,size FROM tracks WHERE path=?`, path).Scan(&oldMtime, &oldSize)
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
	return result, nil
}
