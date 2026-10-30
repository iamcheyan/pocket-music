# Pocket Music

Pocket Music is a desktop Linux music player built as a React + TypeScript web UI, a Go `musicd` service, SQLite storage, and mpv audio playback. The UI communicates with the service over HTTP JSON; it does not access library files or control mpv directly. The architecture keeps application behavior behind an API so the UI can later move to PocketJS Runtime.

## Quick start

Prerequisites: Go, Node.js/npm, and mpv. Install frontend dependencies with `npm --prefix apps/pocket-music/frontend install`, then run from the repository root:

```sh
make dev
```

Open <http://127.0.0.1:5173>. The backend listens on port 8765 by default, and Vite forwards `/api` requests to it. Stop both services with Ctrl-C. `MUSICD_ADDR` can override the backend listen address; `VITE_HOST` controls the Vite bind host.

Run the repository check orchestration with `make test` (Go tests followed by the frontend typecheck). This command is documented for use after implementation; no test or build result is asserted here.

## Library modes

The backend supports configuration through `MUSICD_ADDR`, `MUSICD_DB`, `MUSICD_MUSIC_DIR`, `MUSICD_MOCK`, `MUSICD_MPV`, and `MUSICD_AO`. The exact accepted values and default paths should be confirmed against the running backend before relying on a particular mode. For a real library, configure the music directory and ensure mpv is available. In headless environments, `MUSICD_AO` can select mpv's null audio output if supported by the backend configuration. Mock-mode behavior and default settings are to be recorded after integration verification.

## Screenshots

See [SCREENSHOTS.md](SCREENSHOTS.md) for the required screenshot matrix, output names, and reproducible capture workflow. Images must come from the running application; the expected images are not included as placeholders.

## Performance

Required acceptance measurement: search and paginated list latency with 10,000+ tracks. **Not measured yet.** Record the dataset setup, query, environment, and observed timings here after running the real benchmark. Do not infer performance from the presence of database indexes.

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) — service boundaries, mpv IPC, providers, and migration path.
- [API.md](API.md) — target HTTP contract and request/response examples.
- [DATABASE.md](DATABASE.md) — required entities and schema expectations.
- [ROADMAP.md](ROADMAP.md) — implementation status and known gaps.
