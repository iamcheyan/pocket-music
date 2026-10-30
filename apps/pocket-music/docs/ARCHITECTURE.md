# Architecture

## Runtime layers

```text
React + TypeScript UI (Vite during development)
                │ HTTP JSON /api
                ▼
         Go musicd service
          ├── SQLite library and settings
          ├── local file scanner / metadata
          └── mpv JSON IPC audio engine
```

`musicd` is the sole business entry point. It owns filesystem scanning, metadata, persistence, playback, queues, playlists, and settings. The frontend should use HTTP APIs only; it must not scan directories, read library files, or issue mpv commands. Vite's development proxy routes `/api` to `127.0.0.1:8765`; a production deployment arrangement should be documented when settled.

## Service responsibilities

The Go service provides the HTTP API, coordinates provider operations, and persists library and user state in SQLite. The local provider scans configured music directories and streams local tracks through service endpoints. A mock provider may supply deterministic sample data for demonstrations; verify its exact availability and behavior against the implementation before describing it as an automatic fallback.

Scanning should be incremental using file modification time and size, report progress through the implemented status mechanism, and support cancellation if implemented. The API contract and current endpoint coverage are tracked in [API.md](API.md).

## Audio engine

The service controls mpv through its JSON IPC socket, including playback state, seek, volume, and queue-related commands. The web UI talks only to `musicd`. Headless systems must support an mpv null output (`--ao=null`) so playback progress and completion can operate without a physical output device. The selected output is configurable through settings. Confirm actual startup flags, IPC lifecycle, and state event behavior during integration; do not assume an audio device exists.

## Data flow

1. The UI sends a user action to an HTTP endpoint.
2. `musicd` validates the operation and uses persistence, a provider, or the mpv engine as appropriate.
3. The service returns JSON or a media/cover stream and updates durable state as required.
4. The UI renders returned data and refreshes player or scan state through the implemented API mechanism.

## Provider extension points

The local provider is the initial source for on-device music. Future `BilibiliProvider`, `NavidromeProvider`, or `JellyfinProvider` implementations can satisfy the same provider-facing operations for naming, scanning with progress, listing, and streaming. Provider-specific credentials and remote synchronization policies belong in `musicd`, never in UI code.

## PJOS migration path

Keep UI state and workflows dependent on the HTTP API contract rather than native filesystem or process APIs. A PJOS adapter can replace Vite's development hosting and provide a compatible API transport while `musicd` remains the application service. Where PJOS uses another service/runtime arrangement, preserve the API semantics and provider boundary rather than coupling UI components to Go implementation details.
