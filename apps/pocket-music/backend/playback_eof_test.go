package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func generateTone(t *testing.T, path string, seconds int) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi",
		"-i", "sine=frequency=440:duration="+itoa(seconds), "-metadata", "title=EOF Tone", "-y", path)
	if err := cmd.Run(); err != nil {
		t.Skipf("ffmpeg unavailable: %v", err)
	}
}

func itoa(n int) string {
	return string(rune('0' + n))
}

func TestPlayerProgressesAndCompletesWithoutAudioDevice(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "tone.wav")
	generateTone(t, audio, 1)

	t.Setenv("MUSICD_DB", filepath.Join(t.TempDir(), "eof.db"))
	t.Setenv("MUSICD_MUSIC_DIR", dir)
	t.Setenv("MUSICD_MOCK", "false")
	t.Setenv("MUSICD_AO", "null")
	s, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.db.Close() })
	if _, err := s.provider.Scan(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	tracks, err := s.listQuery(Query{Page: 1, Limit: 1})
	if err != nil || len(tracks) != 1 {
		t.Fatalf("scan result: %v %d", err, len(tracks))
	}

	s.playerMu.Lock()
	s.player.Track = &tracks[0]
	s.player.Playing = true
	s.player.Duration = tracks[0].Duration
	s.playerMu.Unlock()
	if err := s.mpv.start(tracks[0].Path); err != nil {
		t.Skipf("mpv unavailable: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	advanced := false
	for time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		s.syncPlayerState()
		s.playerMu.Lock()
		position, duration, playing := s.player.Position, s.player.Duration, s.player.Playing
		s.playerMu.Unlock()
		if position > 0 {
			advanced = true
		}
		if advanced && position == duration && !playing {
			break
		}
	}
	if !advanced {
		t.Fatal("position never advanced from zero")
	}
	s.playerMu.Lock()
	position, duration, playing := s.player.Position, s.player.Duration, s.player.Playing
	s.playerMu.Unlock()
	if position != duration || playing {
		t.Fatalf("completion state position=%v duration=%v playing=%v", position, duration, playing)
	}
	var completion float64
	if err := s.db.QueryRow(`SELECT completion_rate FROM history WHERE track_id=?`, tracks[0].ID).Scan(&completion); err != nil {
		t.Fatal(err)
	}
	if completion < 0.99 {
		t.Fatalf("completion rate %v", completion)
	}
}
