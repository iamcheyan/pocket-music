package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// departure captures the state of the outgoing track so the history row can be
// updated after the player mutex is released.
type departure struct {
	trackID  string
	fraction float64
}

func (s *Server) playerState(w http.ResponseWriter) {
	s.syncPlayerState()
	s.playerMu.Lock()
	state := s.player
	state.Queue = append([]Track{}, s.player.Queue...)
	s.playerMu.Unlock()
	s.json(w, state)
}

func (s *Server) play(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ID      string   `json:"track_id"`
		Mode    string   `json:"mode"`
		Queue   []string `json:"queue"`
		Context string   `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		s.fail(w, 400, "invalid_json", err.Error())
		return
	}
	if b.Mode != "" && !playbackModes[b.Mode] {
		s.fail(w, 400, "invalid_mode", "unsupported playback mode")
		return
	}
	t, err := s.track(b.ID)
	if err != nil {
		s.fail(w, 404, "track_not_found", "track not found")
		return
	}
	queued := []Track{}
	for _, id := range b.Queue {
		if id == t.ID {
			continue
		}
		if other, e := s.track(id); e == nil {
			queued = append(queued, other)
		}
	}
	s.playerMu.Lock()
	dep := s.departureLocked(t.ID)
	if b.Mode != "" {
		s.player.Mode = b.Mode
	}
	s.player.Track = &t
	s.player.Playing = true
	s.player.Position = 0
	s.player.Duration = t.Duration
	s.player.Queue = queued
	s.player.Source = "local"
	s.trackCompleted = false
	s.lastTick = time.Now()
	s.playerMu.Unlock()
	if dep != nil {
		s.recordDeparture(*dep)
	}
	s.recordStart(t.ID)
	if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
		if err := s.mpv.start(t.Path); err != nil {
			s.playerMu.Lock()
			s.player.Playing = false
			s.playerMu.Unlock()
		}
	}
	s.playerState(w)
}

// departureLocked returns the finished portion of the current track when it is
// being replaced by another one (a skip). It must be called with playerMu held.
func (s *Server) departureLocked(nextID string) *departure {
	if s.player.Track == nil || s.player.Track.ID == nextID {
		if s.player.Track != nil && s.player.Track.ID == nextID {
			return nil
		}
		return nil
	}
	if s.trackCompleted || s.player.Duration <= 0 {
		return nil
	}
	frac := s.player.Position / s.player.Duration
	if frac < 0 || frac >= 0.95 {
		return nil
	}
	return &departure{trackID: s.player.Track.ID, fraction: frac}
}

func (s *Server) recordDeparture(d departure) {
	s.db.Exec(`INSERT INTO history(track_id,skip_count,completion_rate) VALUES(?,1,?)
		ON CONFLICT(track_id) DO UPDATE SET skip_count=skip_count+1,completion_rate=?`, d.trackID, d.fraction, d.fraction)
}

func (s *Server) recordStart(id string) {
	s.db.Exec(`INSERT INTO history(track_id,last_played,play_count) VALUES(?,CURRENT_TIMESTAMP,1)
		ON CONFLICT(track_id) DO UPDATE SET last_played=CURRENT_TIMESTAMP,play_count=play_count+1`, id)
}

// syncPlayerState pulls position from the audio engine. In mock mode there is no
// engine, so the service advances the clock itself; that keeps demo playback
// (and the completion path) testable on a headless host.
func (s *Server) syncPlayerState() {
	if s.mock {
		s.tickMock()
		return
	}
	s.playerMu.Lock()
	if s.player.Track == nil || s.player.Track.Path == "" || strings.HasPrefix(s.player.Track.Path, "mock:") {
		s.playerMu.Unlock()
		return
	}
	s.playerMu.Unlock()
	position, duration, paused, finished, err := s.mpv.status()
	if err != nil {
		return
	}
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	if s.player.Track == nil {
		return
	}
	if finished {
		s.player.Position = s.player.Duration
		s.completeTrack()
		return
	}
	s.player.Position = position
	if duration > 0 {
		s.player.Duration = duration
	}
	s.player.Playing = !paused
	if s.player.Duration > 0 && position >= s.player.Duration-0.25 {
		s.completeTrack()
	}
}

func (s *Server) tickMock() {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	if s.player.Track == nil || !s.player.Playing {
		s.lastTick = time.Now()
		return
	}
	now := time.Now()
	if !s.lastTick.IsZero() {
		delta := now.Sub(s.lastTick).Seconds()
		if delta > 0 && delta < 30 {
			s.player.Position += delta
		}
	}
	s.lastTick = now
	if s.player.Duration > 0 && s.player.Position >= s.player.Duration {
		s.player.Position = s.player.Duration
		s.completeTrack()
	}
}

// completeTrack records a full listen and advances the queue. Caller holds playerMu.
func (s *Server) completeTrack() {
	if s.player.Track == nil || s.trackCompleted {
		return
	}
	t := *s.player.Track
	s.trackCompleted = true
	s.db.Exec(`INSERT INTO history(track_id,last_played,completion_rate) VALUES(?,CURRENT_TIMESTAMP,1)
		ON CONFLICT(track_id) DO UPDATE SET completion_rate=1`, t.ID)
	s.advance(1)
}

// advance moves the queue forward (delta>0) or restarts the current track (delta<0).
// Caller holds playerMu.
func (s *Server) advance(delta int) {
	if s.player.Track == nil {
		s.player.Playing = false
		return
	}
	if delta < 0 || (s.player.Mode == ModeRepeatOne && delta > 0) {
		t := *s.player.Track
		s.player.Position = 0
		s.player.Playing = true
		s.trackCompleted = false
		s.lastTick = time.Now()
		if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
			_ = s.mpv.start(t.Path)
		}
		return
	}
	if len(s.player.Queue) == 0 {
		s.player.Playing = false
		s.player.Position = 0
		return
	}
	index := 0
	if s.player.Mode == ModeShuffle && len(s.player.Queue) > 1 {
		index = int(time.Now().UnixNano() % int64(len(s.player.Queue)))
	}
	t := s.player.Queue[index]
	s.player.Queue = append(s.player.Queue[:index], s.player.Queue[index+1:]...)
	s.player.Track = &t
	s.player.Position = 0
	s.player.Duration = t.Duration
	s.player.Playing = true
	s.trackCompleted = false
	s.lastTick = time.Now()
	s.recordStart(t.ID)
	if t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
		if e := s.mpv.start(t.Path); e != nil {
			s.player.Playing = false
		}
	}
}

func (s *Server) setMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		s.fail(w, 405, "method_not_allowed", "use PUT")
		return
	}
	var b struct {
		Mode string `json:"mode"`
	}
	if e := json.NewDecoder(r.Body).Decode(&b); e != nil || !playbackModes[b.Mode] {
		s.fail(w, 400, "invalid_mode", "mode must be normal, repeat_all, repeat_one or shuffle")
		return
	}
	s.playerMu.Lock()
	s.player.Mode = b.Mode
	state := s.player
	s.playerMu.Unlock()
	s.json(w, state)
}

func (s *Server) playerAction(w http.ResponseWriter, r *http.Request, p string) {
	s.playerMu.Lock()
	switch p {
	case "player/pause":
		s.player.Playing = false
		s.lastTick = time.Now()
		_ = s.mpv.property("pause", true)
	case "player/resume":
		s.player.Playing = true
		s.lastTick = time.Now()
		_ = s.mpv.property("pause", false)
	case "player/next":
		if s.player.Track != nil {
			s.recordDeparture(departure{trackID: s.player.Track.ID, fraction: s.skipFraction()})
		}
		s.advance(1)
	case "player/prev":
		s.player.Position = 0
		s.player.Playing = true
		s.trackCompleted = false
		s.lastTick = time.Now()
		if t := s.player.Track; t != nil && t.Path != "" && !strings.HasPrefix(t.Path, "mock:") {
			_ = s.mpv.start(t.Path)
		}
	case "player/seek":
		var b struct {
			Position float64 `json:"position"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Position < 0 {
			b.Position = 0
		}
		if s.player.Duration > 0 && b.Position > s.player.Duration {
			b.Position = s.player.Duration
		}
		s.player.Position = b.Position
		s.lastTick = time.Now()
		if b.Position < s.player.Duration-0.5 {
			s.trackCompleted = false
		}
		_ = s.mpv.command("seek", b.Position, "absolute")
	case "player/volume":
		var b struct {
			Volume float64 `json:"volume"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Volume < 0 {
			b.Volume = 0
		}
		if b.Volume > 100 {
			b.Volume = 100
		}
		s.player.Volume = b.Volume
		_ = s.mpv.property("volume", b.Volume)
	}
	state := s.player
	s.playerMu.Unlock()
	s.json(w, state)
}

func (s *Server) skipFraction() float64 {
	if s.player.Duration <= 0 || s.trackCompleted {
		return -1
	}
	return s.player.Position / s.player.Duration
}

func (s *Server) queue(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Action   string   `json:"action"`
		TrackID  string   `json:"track_id"`
		TrackIDs []string `json:"track_ids"`
		Index    int      `json:"index"`
		To       int      `json:"to"`
	}
	if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
		s.fail(w, 400, "invalid_json", e.Error())
		return
	}
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	add := func(id string) {
		if t, e := s.track(id); e == nil {
			s.player.Queue = append(s.player.Queue, t)
		}
	}
	switch b.Action {
	case "add":
		for _, id := range b.TrackIDs {
			add(id)
		}
		if b.TrackID != "" {
			add(b.TrackID)
		}
	case "play_next":
		if b.TrackID != "" {
			if t, e := s.track(b.TrackID); e == nil {
				s.player.Queue = append([]Track{t}, s.player.Queue...)
			}
		}
	case "remove":
		if b.Index >= 0 && b.Index < len(s.player.Queue) {
			s.player.Queue = append(s.player.Queue[:b.Index], s.player.Queue[b.Index+1:]...)
		}
	case "reorder":
		if b.Index >= 0 && b.Index < len(s.player.Queue) && b.To >= 0 && b.To < len(s.player.Queue) {
			item := s.player.Queue[b.Index]
			rest := append(s.player.Queue[:b.Index], s.player.Queue[b.Index+1:]...)
			out := make([]Track, 0, len(rest)+1)
			out = append(out, rest[:b.To]...)
			out = append(out, item)
			out = append(out, rest[b.To:]...)
			s.player.Queue = out
		}
	case "clear":
		s.player.Queue = []Track{}
	default:
		s.failLocked(w)
		return
	}
	s.json(w, map[string]any{"queue": s.player.Queue})
}

// failLocked is used where the player mutex is already held; it writes the
// response directly instead of unlocking.
func (s *Server) failLocked(w http.ResponseWriter) {
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_action", "message": "action must be add, play_next, remove, reorder or clear"}})
}
