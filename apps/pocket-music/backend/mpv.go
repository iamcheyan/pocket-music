package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type mpvPropertyResponse struct {
	RequestID int64  `json:"request_id"`
	Error     string `json:"error"`
	Data      any    `json:"data"`
}

func (m *mpvClient) getProperty(name string) (any, error) {
	c, e := net.DialTimeout("unix", m.socket, 300*time.Millisecond)
	if e != nil {
		return nil, e
	}
	defer c.Close()
	id := m.seq.Add(1)
	b, _ := json.Marshal(map[string]any{"command": []any{"get_property", name}, "request_id": id})
	if _, e = fmt.Fprintf(c, "%s\n", b); e != nil {
		return nil, e
	}
	_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	sc := bufio.NewScanner(c)
	for sc.Scan() {
		var r mpvPropertyResponse
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.RequestID == id {
			if r.Error != "success" {
				return nil, fmt.Errorf("mpv: %s", r.Error)
			}
			return r.Data, nil
		}
	}
	return nil, sc.Err()
}

func (m *mpvClient) status() (position, duration float64, paused bool, err error) {
	p, e := m.getProperty("time-pos")
	if e != nil {
		return 0, 0, false, e
	}
	d, e := m.getProperty("duration")
	if e != nil {
		return 0, 0, false, e
	}
	a, e := m.getProperty("pause")
	if e != nil {
		return 0, 0, false, e
	}
	if v, ok := p.(float64); ok {
		position = v
	}
	if v, ok := d.(float64); ok {
		duration = v
	}
	paused, _ = a.(bool)
	return position, duration, paused, nil
}

type mpvClient struct {
	socket string
	seq    atomic.Int64
	mu     sync.Mutex
	cmd    *exec.Cmd
	output string
}

func newMPV() *mpvClient {
	return &mpvClient{socket: filepath.Join(os.TempDir(), fmt.Sprintf("musicd-%d.sock", os.Getpid())), output: env("MUSICD_AO", "null")}
}
func (m *mpvClient) setOutput(output string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if output != "" {
		m.output = output
	}
}
func (m *mpvClient) start(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.ProcessState != nil {
		_ = os.Remove(m.socket)
		cmd := exec.Command(env("MUSICD_MPV", "mpv"), "--no-terminal", "--idle=yes", "--ao="+m.output, "--input-ipc-server="+m.socket)
		if e := cmd.Start(); e != nil {
			return e
		}
		m.cmd = cmd
		go cmd.Wait()
		for range 60 {
			if _, e := os.Stat(m.socket); e == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	return m.command("loadfile", path, "replace")
}
func (m *mpvClient) command(args ...any) error {
	c, e := net.DialTimeout("unix", m.socket, 300*time.Millisecond)
	if e != nil {
		return e
	}
	defer c.Close()
	id := m.seq.Add(1)
	b, _ := json.Marshal(map[string]any{"command": args, "request_id": id})
	fmt.Fprintf(c, "%s\n", b)
	sc := bufio.NewScanner(c)
	for sc.Scan() {
		var r struct {
			RequestID int64  `json:"request_id"`
			Error     string `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.RequestID == id {
			if r.Error != "success" {
				return fmt.Errorf("mpv: %s", r.Error)
			}
			return nil
		}
	}
	return sc.Err()
}
func (m *mpvClient) property(name string, value any) error {
	return m.command("set_property", name, value)
}
