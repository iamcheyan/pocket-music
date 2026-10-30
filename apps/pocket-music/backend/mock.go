package main

import (
	"fmt"
	"math/rand"
)

// Deterministic demo library dimensions. Keeping them as constants lets tests
// assert on the seeded size without re-deriving it.
const (
	mockArtistCount     = 8
	mockAlbumsPerArtist = 3
	mockTracksPerAlbum  = 10
	mockTrackCount      = mockArtistCount * mockAlbumsPerArtist * mockTracksPerAlbum
)

var mockArtistNames = []string{
	"Nova Lantern", "Mira Kade", "Field of Static", "Aiko Sera",
	"Low Tide Radio", "Vesper Lane", "Kite & Compass", "Hana Mori",
}

var mockAlbumNames = []string{
	"Night Cartography", "Signal Garden", "Slow Machines",
	"Paper Satellites", "Analogue Summer", "Tidal Rooms",
	"Harbour Lights", "Quiet Voltage",
}

var mockGenres = []string{"Electronic", "Ambient", "Jazz", "Folk", "Shoegaze", "City Pop", "Modern Classical", "Lo-Fi"}

// seedMock fills an empty library with a demo collection: tracks, history,
// favourites and playlists. Every page of the UI is then reviewable without a
// real music folder.
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

	rng := rand.New(rand.NewSource(20261030))
	processed := 0
	for a := 0; a < mockArtistCount; a++ {
		artist := mockArtistNames[a%len(mockArtistNames)]
		tx.Exec(`INSERT OR IGNORE INTO artists(name) VALUES(?)`, artist)
		var artistID int
		tx.QueryRow(`SELECT id FROM artists WHERE name=?`, artist).Scan(&artistID)

		for b := 0; b < mockAlbumsPerArtist; b++ {
			album := fmt.Sprintf("%s %s", mockAlbumNames[(a+b)%len(mockAlbumNames)],
				[]string{"I", "II", "III"}[b])
			year := 2011 + (a*3+b)%14
			tx.Exec(`INSERT INTO albums(title,artist_id,year) SELECT ?,?,?
				WHERE NOT EXISTS(SELECT 1 FROM albums WHERE title=? AND artist_id=?)`, album, artistID, year, album, artistID)
			var albumID int
			tx.QueryRow(`SELECT id FROM albums WHERE title=? AND artist_id=?`, album, artistID).Scan(&albumID)
			genre := mockGenres[(a*3+b)%len(mockGenres)]

			for t := 0; t < mockTracksPerAlbum; t++ {
				processed++
				id := fmt.Sprintf("mock-%03d", processed)
				title := fmt.Sprintf("%s %02d", []string{"Opening Light", "Drift", "Half Awake", "Paper Moon",
					"Static Bloom", "Night Bus", "Small Hours", "Soft Focus", "Tidal", "Closing Light"}[t%10], processed)
				duration := 118 + float64(rng.Intn(190))
				days := rng.Intn(120)
				// Demo paths mirror a real Artist/Album layout so the Folders view
				// has something meaningful to group; the mock: prefix keeps the
				// scanner, media and cover handlers from touching the filesystem.
				mockPath := fmt.Sprintf("mock:%s/%s/%02d %s.mp3", artist, album, t+1, title)
				tx.Exec(`INSERT OR IGNORE INTO tracks(id,title,artist_id,album_id,genre,year,duration,path,track_no,added_at)
					VALUES(?,?,?,?,?,?,?,?,?,datetime('now',?))`,
					id, title, artistID, albumID, genre, year, duration, mockPath, t+1, fmt.Sprintf("-%d days", days))
				tx.Exec(`INSERT OR IGNORE INTO tracks_fts(track_id,title,artist,album,genre) VALUES(?,?,?,?,?)`,
					id, title, artist, album, genre)

				// Roughly one in four demo tracks has listening history.
				if rng.Intn(4) == 0 {
					plays := 1 + rng.Intn(14)
					skips := rng.Intn(3)
					completion := 0.45 + rng.Float64()*0.55
					tx.Exec(`INSERT INTO history(track_id,last_played,play_count,skip_count,completion_rate)
						VALUES(?,datetime('now',?),?,?,?)`,
						id, fmt.Sprintf("-%d hours", rng.Intn(600)), plays, skips, completion)
				}
			}
		}
	}

	for i := 1; i <= 8; i++ {
		tx.Exec(`INSERT OR IGNORE INTO favorites(kind,entity_id,created_at) VALUES('track',?,datetime('now',?))`,
			fmt.Sprintf("mock-%03d", i*7), fmt.Sprintf("-%d hours", i))
	}
	for i := 1; i <= 4; i++ {
		tx.Exec(`INSERT OR IGNORE INTO favorites(kind,entity_id,created_at) VALUES('album',?,datetime('now',?))`,
			fmt.Sprint(i*2), fmt.Sprintf("-%d hours", i*3))
	}
	for i := 1; i <= 3; i++ {
		tx.Exec(`INSERT OR IGNORE INTO favorites(kind,entity_id,created_at) VALUES('artist',?,datetime('now',?))`,
			fmt.Sprint(i), fmt.Sprintf("-%d hours", i*5))
	}

	playlists := []struct {
		name  string
		items []string
	}{
		{"Late Night Drive", []string{"mock-001", "mock-013", "mock-027", "mock-044", "mock-061", "mock-092"}},
		{"Sunday Morning", []string{"mock-003", "mock-008", "mock-030", "mock-075", "mock-110"}},
		{"Focus Session", []string{"mock-020", "mock-021", "mock-046", "mock-047", "mock-088", "mock-089", "mock-140"}},
	}
	for _, p := range playlists {
		res, e := tx.Exec(`INSERT INTO playlists(name) VALUES(?)`, p.name)
		if e != nil {
			continue
		}
		pid, _ := res.LastInsertId()
		for i, trackID := range p.items {
			tx.Exec(`INSERT OR IGNORE INTO playlist_items(playlist_id,position,track_id) VALUES(?,?,?)`, pid, i, trackID)
		}
	}

	tx.Commit()
}
