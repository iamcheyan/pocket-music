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

The provider interface is the extension seam for future Bilibili, Navidrome, or Jellyfin implementations. Remote credentials and synchronization belong in the service, not the UI. Those remote providers are not implemented.

## Scanning and storage

The local scanner walks the configured music root, recognizes MP3, FLAC, AAC, M4A, OGG, WAV, and APE, and uses ffprobe for format duration and title/artist/album/genre tags. Missing title/artist/album values receive fallback labels. Scan routes expose start, polling status, and cancellation. Current implementation's incremental comparison should be verified against the stated mtime-and-size requirement; see [ROADMAP.md](ROADMAP.md).

SQLite is opened with `modernc.org/sqlite`; schema initialization creates tables/indexes and FTS5 at startup. Current schema/version limitations are detailed in [DATABASE.md](DATABASE.md).

## Audio engine

The service controls mpv through JSON IPC, including commands and properties, and configures `--ao=null` by default for headless environments. `MUSICD_AO` provides audio output configuration, and `MUSICD_MPV` selects the mpv executable. The web UI talks only to `musicd`. Verify mpv process lifecycle, IPC behavior, playback progress/completion, and physical output selection at runtime; the presence of IPC code alone does not prove end-to-end playback.

## Data flow and PJOS migration

1. The UI sends user actions to HTTP endpoints.
2. `musicd` uses its provider, database, or mpv engine as required.
3. The service returns JSON or media data and persists relevant state.
4. The UI renders service responses and refreshes state through API calls.

Keep UI workflows dependent on the API contract rather than native filesystem or process APIs. A PJOS adapter can replace Vite hosting and provide compatible API transport while preserving service boundaries. Validate transport, input/focus, storage configuration, and media rendering on target devices.
