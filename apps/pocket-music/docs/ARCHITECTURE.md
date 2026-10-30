# Architecture

## Runtime layers

```text
React + TypeScript UI (Vite during development)
                │ HTTP JSON /api
                ▼
         Go musicd service
          ├── SQLite library and settings
          ├── local scanner / metadata extraction
          └── mpv JSON IPC audio engine
```

`musicd` is the business entry point. It owns filesystem scanning, metadata extraction, persistence, playback, queues, playlists, and settings. The frontend uses HTTP APIs; it must not scan directories, read library files, or issue mpv commands. Vite's development proxy sends `/api` to `127.0.0.1:8765`.

## Service and providers

The Go service coordinates persistence, provider operations, and HTTP handlers. `LocalProvider` provides local library operations. With `MUSICD_MOCK=true`, the service selects `MockProvider` and seeds deterministic demo records if the database is empty (currently 64 tracks across eight artists and eight albums). Mock records are database metadata and do not correspond to playable audio files.

The `MusicProvider` interface is `Name()`, `Scan(ctx, progress)`, `List(ctx, Query)`, and `Stream(ctx, trackID)`; see `backend/provider.go`. `LocalProvider` and `MockProvider` implement it. Bilibili, Navidrome, and Jellyfin providers remain extension points: add a type implementing the interface, register it in `NewServer`, and keep credentials and synchronization in `musicd`. They are not implemented.

## Frontend state model

The React UI is one Vite application with three layouts (`music`, `library`, `minimal`). Page, library tab, theme, language, layout, and search query can be supplied as URL parameters (`view`, `tab`, `theme`, `lang`, `layout`, `q`), which makes screenshots and deep links deterministic. All data and mutations go through `/api`.

Input is handled without a pointer requirement: Space toggles playback, ArrowLeft/Right change tracks, ArrowUp/Down change volume, and Escape returns home. Gamepad polling uses edge-triggered state held in a ref so a held button fires once; D-pad axes switch between Home and Now Playing. Layouts use CSS grid, `clamp()`, and container-relative units, with breakpoints at 900px, 650px, 380px, and 270px height.

## Scanning and storage

The local scanner walks the configured music root, recognizes MP3, FLAC, AAC, M4A, OGG, WAV, and APE, and uses ffprobe for duration and title/artist/album/genre/year tags. Incremental rescans compare both file mtime and size; unchanged files are skipped. Scan routes expose start, polling status, and cancellation, and progress reports found/total/current.

SQLite is opened with `modernc.org/sqlite`; schema initialization creates tables, indexes, and FTS5 at startup.

## Audio engine

The service starts mpv with `--idle=yes`, `--ao=<output>`, and `--input-ipc-server=<unix socket>`, then controls it through JSON IPC. `/api/player/state` polls `time-pos`, `duration`, and `pause`. When mpv reports `time-pos` unavailable and `idle-active=true`, the track is treated as finished: position is set to duration, completion is recorded, and the queue advances according to repeat/shuffle mode. `MUSICD_AO` selects the audio output (default `null` for headless hosts) and `MUSICD_MPV` selects the executable. The web UI talks only to `musicd`.

## Data flow and PJOS migration

1. The UI sends user actions to HTTP endpoints.
2. `musicd` uses its provider, database, or mpv engine as required.
3. The service returns JSON or media data and persists relevant state.
4. The UI renders service responses and refreshes state through API calls.

Keep UI workflows dependent on the API contract rather than native filesystem or process APIs. A PJOS adapter can replace Vite hosting and provide compatible API transport while preserving service boundaries. Validate transport, input/focus, storage configuration, and media rendering on target devices.
