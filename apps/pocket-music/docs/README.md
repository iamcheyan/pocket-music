# Pocket Music

Pocket Music is a desktop Linux music player built as a React + TypeScript web UI, a Go `musicd` service, SQLite storage, and mpv audio playback. The UI communicates with the service over HTTP JSON; it does not access library files or control mpv directly. The API boundary is intended to support a future PocketJS Runtime UI.

## Quick start

Prerequisites: Go, Node.js/npm, and mpv. From the repository root, install frontend dependencies and start both development processes:

```sh
npm --prefix apps/pocket-music/frontend install
make dev
```

Open <http://127.0.0.1:5173>. The frontend uses Vite on port 5173 and proxies `/api` to `127.0.0.1:8765`. The Go service listens on `:8765` by default. Stop the services with Ctrl-C. `MUSICD_ADDR` overrides the backend listen address.

`make test` runs backend `go test ./...`, the frontend `typecheck` script, and an esbuild-bundled check that the English, Simplified Chinese, and Japanese catalogs have identical keys. It passed on 2026-10-30.

## Library configuration

The service reads `MUSICD_ADDR`, `MUSICD_DB`, `MUSICD_MUSIC_DIR`, `MUSICD_MOCK`, `MUSICD_MPV`, and `MUSICD_AO`. Defaults are `:8765`, `musicd.db`, `$HOME/Music`, mock disabled, and `--ao=null` audio output. The health response reports provider, mock status, and audio status/output.

Set `MUSICD_MOCK=true` to seed a deterministic demo library when the database has no tracks: 64 demo tracks, eight artists, and eight albums. For a real library, set `MUSICD_MUSIC_DIR` and scan through `POST /api/scan`. The scanner accepts MP3, FLAC, AAC, M4A, OGG, WAV, and APE and uses ffprobe for metadata. A generated 2-second WAV was scanned into a temp directory with correct title, artist, album, year, genre, duration, and path; playback with `--ao=null` advanced the position and fired completion/queue advancement.

## Screenshots and performance

See [SCREENSHOTS.md](SCREENSHOTS.md) for Chromium screenshots captured from the running application.

The 10,000-track benchmark generated deterministic SQLite rows and measured a paired FTS5 search plus paginated list query using `go test -run '^$' -bench '^BenchmarkSearchAndLibrary10K$' -benchtime=1x -benchmem` on the HX90 (AMD Ryzen 9 5900HX): **23.35 ms/op**, 67,800 B/op, 1,060 allocs/op. This is one benchmark iteration; timings are a local baseline, not a multi-run distribution.

The screenshot index lists 17 captured files under `docs/screenshots/`, including every required view and all four required sizes. The files were captured from a running mock-mode application with Chromium headless; filenames use PNG IHDR dimensions.

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) — service boundaries, mpv IPC, providers, and PJOS migration.
- [API.md](API.md) — current routes and request/response shapes.
- [DATABASE.md](DATABASE.md) — SQLite tables, indexes, and schema behavior.
- [ROADMAP.md](ROADMAP.md) — implementation status and remaining evidence.
