package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Server) media(w http.ResponseWriter, r *http.Request, id string) {
	t, e := s.track(id)
	if e != nil {
		s.fail(w, 404, "track_not_found", "track not found")
		return
	}
	if t.Path == "" || strings.HasPrefix(t.Path, "mock:") {
		s.fail(w, 404, "media_missing", "track has no audio file")
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

type lyricLine struct {
	Time float64 `json:"time"`
	Text string  `json:"text"`
}

func parseLRC(data string) []lyricLine {
	lines := []lyricLine{}
	for _, raw := range strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n") {
		text := raw
		times := []float64{}
		for strings.HasPrefix(text, "[") {
			end := strings.IndexByte(text, ']')
			if end < 0 {
				break
			}
			stamp := text[1:end]
			if !strings.Contains(stamp, ":") {
				break
			}
			colon := strings.IndexByte(stamp, ':')
			minutes, err1 := strconv.Atoi(strings.TrimSpace(stamp[:colon]))
			seconds, err2 := strconv.ParseFloat(strings.TrimSpace(stamp[colon+1:]), 64)
			if err1 != nil || err2 != nil || minutes < 0 || seconds < 0 || seconds >= 60 {
				break
			}
			times = append(times, float64(minutes)*60+seconds)
			text = strings.TrimSpace(text[end+1:])
		}
		if len(times) == 0 || text == "" {
			continue
		}
		for _, at := range times {
			lines = append(lines, lyricLine{Time: at, Text: text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })
	return lines
}

// loadLyrics reads an .lrc file next to the track first, then falls back to
// lyrics embedded in the audio tags.
func loadLyrics(t Track) []lyricLine {
	if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
		candidate := strings.TrimSuffix(t.Path, filepath.Ext(t.Path)) + ".lrc"
		if b, e := os.ReadFile(candidate); e == nil {
			if lines := parseLRC(string(b)); len(lines) > 0 {
				return lines
			}
		}
		data := probe(t.Path)
		for _, tags := range append([]map[string]string{data.Format.Tags}, streamTags(data)...) {
			for _, key := range []string{"lyrics", "unsyncedlyrics", "syncedlyrics", "unsynced_lyrics"} {
				if v := tagValue(tags, key); v != "" {
					if lines := parseLRC(v); len(lines) > 0 {
						return lines
					}
					var plain []lyricLine
					for i, raw := range strings.Split(v, "\n") {
						if strings.TrimSpace(raw) == "" {
							continue
						}
						plain = append(plain, lyricLine{Time: float64(i * 4), Text: strings.TrimSpace(raw)})
					}
					if len(plain) > 0 {
						return plain
					}
				}
			}
		}
	}
	// Demo tracks carry a deterministic lyric sheet so the lyrics view is
	// reviewable without any audio assets.
	if strings.HasPrefix(t.Path, "mock:") {
		return demoLyrics(t)
	}
	return nil
}

func streamTags(data ffprobeOutput) []map[string]string {
	out := []map[string]string{}
	for _, st := range data.Streams {
		out = append(out, st.Tags)
	}
	return out
}

func demoLyrics(t Track) []lyricLine {
	verses := []string{
		"Signals in the static", "A room without a clock", "Everything is slower here",
		"and the needle finds the groove", "Pocket full of music", "Carried through the noise",
		"Somewhere in the chorus", "the evening finds its voice",
	}
	out := make([]lyricLine, 0, len(verses))
	for i, v := range verses {
		out = append(out, lyricLine{Time: float64(i) * 12, Text: v})
	}
	_ = t
	return out
}

func (s *Server) lyrics(w http.ResponseWriter, id string) {
	t, e := s.track(id)
	if e != nil {
		s.fail(w, 404, "track_not_found", "track not found")
		return
	}
	lines := loadLyrics(t)
	s.json(w, map[string]any{"track_id": t.ID, "lines": lines, "synced": len(lines) > 0, "count": len(lines)})
}

// cover serves album art: cached embedded art, folder.jpg/cover.jpg next to the
// tracks, or a deterministic generated cover so the UI never shows a hole.
func (s *Server) cover(w http.ResponseWriter, r *http.Request, id string) {
	var title, artist, stored string
	var artistID int
	e := s.db.QueryRow(`SELECT al.title,COALESCE(ar.name,''),COALESCE(al.cover,''),COALESCE(al.artist_id,0) FROM albums al LEFT JOIN artists ar ON ar.id=al.artist_id WHERE al.id=?`, id).Scan(&title, &artist, &stored, &artistID)
	if e != nil {
		s.fail(w, 404, "cover_not_found", "album not found")
		return
	}
	if stored != "" {
		if f, err := os.Open(stored); err == nil {
			defer f.Close()
			http.ServeContent(w, r, filepath.Base(stored), time.Time{}, f)
			return
		}
	}
	var trackPath string
	s.db.QueryRow(`SELECT path FROM tracks WHERE album_id=? AND path NOT LIKE 'mock:%' ORDER BY COALESCE(track_no,0),id LIMIT 1`, id).Scan(&trackPath)
	if trackPath != "" {
		dir := filepath.Dir(trackPath)
		for _, name := range []string{"folder.jpg", "cover.jpg", "Folder.jpg", "Cover.jpg", "front.jpg"} {
			if f, err := os.Open(filepath.Join(dir, name)); err == nil {
				defer f.Close()
				http.ServeContent(w, r, name, time.Time{}, f)
				return
			}
		}
		if err := os.MkdirAll(s.cacheDir, 0o755); err == nil {
			cached := filepath.Join(s.cacheDir, "cover-"+id+".jpg")
			if info, err := os.Stat(cached); err != nil || info.Size() == 0 {
				cmd := exec.Command("ffmpeg", "-y", "-v", "error", "-i", trackPath, "-an", "-frames:v", "1", cached)
				if err := cmd.Run(); err != nil {
					os.Remove(cached)
				}
			}
			if info, err := os.Stat(cached); err == nil && info.Size() > 0 {
				http.ServeFile(w, r, cached)
				return
			}
		}
	}
	writePlaceholderCover(w, title, artist)
}

// placeholderPalettes are ordered by hue so consecutive albums in a grid look
// like a curated shelf rather than random noise.
var placeholderPalettes = [][2]string{
	{"#3c4527", "#c8f26a"}, {"#4a3527", "#ffcb92"}, {"#24404b", "#a5dfeb"},
	{"#43284a", "#efadf4"}, {"#4f3d20", "#eac66e"}, {"#27402f", "#b3f2c0"},
	{"#3a3040", "#c9b6ff"}, {"#40332b", "#ffb6a3"},
}

func writePlaceholderCover(w http.ResponseWriter, title, artist string) {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(title)))
	seed := h.Sum32()
	pal := placeholderPalettes[int(seed)%len(placeholderPalettes)]
	angle := 100 + seed%70
	initials := ""
	for _, part := range strings.Fields(title) {
		if len(initials) < 2 {
			r := []rune(strings.ToUpper(part))
			if len(r) > 0 {
				initials += string(r[0])
			}
		}
	}
	title = escapeXML(truncate(title, 26))
	artist = escapeXML(truncate(artist, 28))
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 600 600" role="img" aria-label="%s">
<defs><linearGradient id="g" gradientTransform="rotate(%d)"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="#101109"/></linearGradient>
<radialGradient id="glow" cx="50%%" cy="42%%" r="55%%"><stop offset="0" stop-color="%s" stop-opacity=".38"/><stop offset="1" stop-color="%s" stop-opacity="0"/></radialGradient></defs>
<rect width="600" height="600" fill="url(#g)"/><rect width="600" height="600" fill="url(#glow)"/>
<g fill="none" stroke="%s" stroke-opacity=".30"><circle cx="300" cy="300" r="196"/><circle cx="300" cy="300" r="152"/><circle cx="300" cy="300" r="108"/></g>
<circle cx="300" cy="300" r="30" fill="%s" fill-opacity=".85"/>
<text x="52" y="60" fill="%s" font-family="system-ui,sans-serif" font-size="26" letter-spacing="6" opacity=".85">POCKET</text>
<text x="52" y="512" fill="#f4f4ec" font-family="system-ui,sans-serif" font-size="40" font-weight="700">%s</text>
<text x="52" y="552" fill="%s" font-family="system-ui,sans-serif" font-size="24" opacity=".8">%s</text>
<text x="548" y="512" text-anchor="end" fill="%s" font-family="system-ui,sans-serif" font-size="60" font-weight="800" opacity=".9">%s</text>
</svg>`, artist, angle, pal[0], pal[1], pal[1], pal[1], pal[1], pal[1], title, pal[1], artist, pal[1], escapeXML(initials))
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(svg))
}

func escapeXML(v string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(v)
}

func truncate(v string, n int) string {
	r := []rune(v)
	if len(r) <= n {
		return v
	}
	return string(r[:n-1]) + "…"
}

// decodeBody is a small helper for handlers that accept optional JSON bodies.
func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty body")
	}
	return json.NewDecoder(r.Body).Decode(v)
}
