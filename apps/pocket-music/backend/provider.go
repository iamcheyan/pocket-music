package main

import (
	"context"
	"io"
	"os"
)

type Query struct {
	Text        string
	Page, Limit int
}
type ScanProgress struct {
	Found, Total int
	Current      string
}
type ScanResult struct{ Found, Updated, Skipped, Removed int }
type MusicProvider interface {
	Name() string
	Scan(context.Context, func(ScanProgress)) (ScanResult, error)
	List(context.Context, Query) ([]Track, error)
	Stream(context.Context, string) (io.ReadCloser, error)
}

type LocalProvider struct{ server *Server }

func (p *LocalProvider) Name() string { return "local" }
func (p *LocalProvider) Scan(ctx context.Context, progress func(ScanProgress)) (ScanResult, error) {
	return p.server.scanProvider(ctx, progress)
}
func (p *LocalProvider) List(ctx context.Context, q Query) ([]Track, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.server.listQuery(q)
}
func (p *LocalProvider) Stream(ctx context.Context, id string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t, e := p.server.track(id)
	if e != nil {
		return nil, e
	}
	return osOpen(t.Path)
}

type MockProvider struct{ local *LocalProvider }

func (p *MockProvider) Name() string { return "mock" }
func (p *MockProvider) Scan(ctx context.Context, progress func(ScanProgress)) (ScanResult, error) {
	if progress != nil {
		progress(ScanProgress{})
	}
	return ScanResult{}, ctx.Err()
}
func (p *MockProvider) List(ctx context.Context, q Query) ([]Track, error) {
	return p.local.List(ctx, q)
}
func (p *MockProvider) Stream(ctx context.Context, id string) (io.ReadCloser, error) {
	return p.local.Stream(ctx, id)
}

func osOpen(path string) (io.ReadCloser, error) { return os.Open(path) }
