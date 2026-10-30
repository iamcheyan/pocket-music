# Architecture

## Runtime layers

```text
React 19 + TypeScript UI (Vite in development, served by musicd in release)
                                │  HTTP JSON  /api/…
                                ▼
                       Go service `musicd`
        ├── SQLite library, history, favourites, playlists, settings
        ├── scanner + metadata service (filesystem walk, ffprobe)
        ├── cover + lyric resolution
        └── queue engine ──► mpv JSON IPC (unix socket)
```

`musicd` is the only component that touches the filesystem, the database or the
audio engine. The UI holds no library state of its own beyond what the last API
response returned. This is the boundary a PocketJS runtime adapter will preserve
on PJOS: swap the transport and the view layer, keep the service.

## Service modules

| File             | Responsibility                                                            |
| ---------------- | ------------------------------------------------------------------------- |
| `main.go`        | Server type, configuration, migrations, route table, static SPA serving    |
| `library.go`     | Track projection, paged listing, collections, search, scanner, metadata   |
| `playback.go`    | Player state, queue engine, skip/completion accounting, mode switching    |
| `media.go`       | Audio streaming with Range support, cover resolution, LRC parsing         |
| `playlists.go`   | Playlist CRUD, item ordering, M3U/M3U8 import/export, scan routes         |
| `collections.go` | Album/artist detail, favourites, statistics, settings validation          |
| `mock.go`        | Deterministic demo library (240 tracks, history, favourites, playlists)   |
| `provider.go`    | `MusicProvider` interface, `LocalProvider`, `MockProvider`                |
| `mpv.go`         | mpv process lifecycle and JSON IPC client                                 |

Every handler is a `*Server` method, so the whole service runs inside
`httptest` without opening ports — that is how the API tests and the 10k-track
benchmarks execute.

### Concurrency

- `db.SetMaxOpenConns(1)` serialises SQLite access; there is no writer contention
  because handlers never hold the database open across a blocking wait.
- `playerMu` guards the player struct, the queue and the playback clock. Handlers
  capture state under the lock and release it before writing history rows.
- `mu` guards scan progress and the cancel function, so `GET /api/scan/status`
  is always a snapshot.
- The UI polls `GET /api/player/state` every 1.2 s. Each poll advances the state
  machine: with mpv it reads `time-pos`/`duration`/`pause`; in mock mode it
  advances the position by elapsed wall time.

## Playback state machine

```text
play(track) ──► depart(previous) ──► load file (or mock clock) ──► history++play
   │                │
   │                └─ skipped under 95 % → skip_count++, completion_rate = fraction
   │
   ├─ state poll: position ≥ duration − 0.25 s, or mpv idle after playback
   │        └─ complete: completion_rate = 1, advance queue
   │
   └─ advance: normal → queue head, repeat_all → queue or restart when empty,
              repeat_one → restart current, shuffle → random queue index
```

Rules worth knowing:

- `play` accepts an optional `queue` of track ids, so "play album", "play artist"
  and "play playlist" are one request that also replaces the queue.
- Departure is only recorded when a *different* track replaces the current one,
  and never once completion has been recorded, so a skip cannot double count.
- `completion_rate` means "fraction of the last play that was heard"; the
  statistics endpoint averages it across history rows.
- The queue never contains the playing track. `play_next` inserts at index 0,
  `advance` pops the head (or a random index in shuffle), `clear` empties it.
- Mock mode drives the same code path with a wall clock, so completion, queue
  advancement and skip accounting are testable without an audio device.

## Media resolution

**Cover art** (`GET /api/cover/{albumID}`) tries, in order:

1. `albums.cover` — an explicit stored path;
2. `folder.jpg`, `cover.jpg`, `Folder.jpg`, `Cover.jpg`, `front.jpg` next to the
   album's first track;
3. embedded art extracted once with `ffmpeg -frames:v 1` into `MUSICD_CACHE`;
4. a generated SVG cover — deterministic palette, initials and title — so the UI
   never renders a hole and demo libraries look intentional.

**Lyrics** (`GET /api/lyrics/{trackID}`) reads `<track>.lrc` first, then the
`lyrics` / `unsyncedlyrics` / `syncedlyrics` tags in format or stream metadata
via ffprobe. Both LRC and plain text are accepted; LRC timestamps are sorted and
returned as `{time, text}` lines that the UI highlights against the playhead.

**Audio** (`GET /api/media/{trackID}`) serves the file path recorded in the
database with `http.ServeContent`, giving Range requests for progressive
playback. Paths only ever come from the scanner, never from the request.

## Scanner

1. Walk the configured root, skipping dot-directories, accepting
   `.mp3 .flac .aac .m4a .ogg .wav .ape`.
2. Compare `mtime` + `size` per path; unchanged files are skipped, changed files
   are re-imported, new files are inserted.
3. Metadata from `ffprobe` (title, artist, album, genre, date, track number,
   duration) with file-name and folder-name fallbacks.
4. Write the row, refresh the FTS5 row, and delete rows whose files disappeared.
5. Report `found/total/current` through a callback; the status endpoint exposes
   the snapshot and a cancel endpoint stops the walk.
6. The whole scan runs in one goroutine; a second `POST /api/scan` returns 409.

## Frontend architecture

```text
src/
  main.tsx            bootstrap
  App.tsx             shell: sidebar, top bar, player bar, mobile nav, input
  app/usePocket.ts    one hook: data, navigation, polling, all mutations
  app/api.ts          typed API client (one function per route)
  app/types.ts        shared response types
  components/         Cover, Slider, Select, Segmented, Toggle, TrackList, cards
  pages/              Home, Library (+album/artist), Playback (now/queue),
                      Playlists, Search (+statistics), Settings
  lib/                pure helpers (format, mode cycle, queue math) + unit tests
  locale.ts           en / zh / ja catalogs
  i18n-contract.ts    build-time key-parity check
```

- **Navigation** is a small state machine (`page`, `tab`, `id`, `q`) mirrored to
  the URL on every change. A URL fully determines a screen, which is what makes
  `make screenshots` deterministic.
- **Data flow** is one-way: `usePocket` fetches, pages render, actions call the
  API and then update state. No page touches `fetch` directly except the folder
  expansion, which uses the same `/api/library/songs` endpoint.
- **Polling** keeps playback state live; the hook compares track ids to refresh
  "recently played" and reloads lyrics when the track changes.
- **i18n** fails the build if a key exists in one catalog and not another.
- **Styling** is a single stylesheet built on CSS custom properties. Theme is a
  `data-theme` attribute (`light` / `dark` / `system`), layout is `data-layout`;
  `system` uses `prefers-color-scheme` so the OS decides.
- **Controls** are app-drawn: segmented pickers, listbox selects, switches and
  pointer/keyboard sliders (`role="slider"`, arrow, Home and End keys). Native
  `select` and `input[type=range]` are deliberately not used.
- **Responsive strategy** uses `clamp()` for type and spacing, `auto-fill` grids
  for card shelves, and breakpoints at 1180 / 980 / 760 / 430 px plus 340 / 300 px
  of height. DPI is not special-cased: everything is in rem/CSS pixels, verified
  at 1.5×/2×/3× device scale factors by the capture script.

## Providers and PJOS migration

`MusicProvider` is `Name()`, `Scan(ctx, progress)`, `List(ctx, Query)` and
`Stream(ctx, trackID)`. `LocalProvider` implements it against the filesystem;
`MockProvider` wraps it for the demo library. A Bilibili, Navidrome or Jellyfin
provider implements the same interface, keeps credentials service-side and
persists its items into the same tables — the UI does not change.

PJOS migration keeps the contract and replaces the edges:

1. **Transport** — a PocketJS client speaks the same `/api` JSON over loopback or
   a runtime RPC bridge instead of `fetch`.
2. **View layer** — the page model (home/library/now/queue/playlists/search/
   settings) maps 1:1 onto PocketJS screens; the layout breakpoints become device
   profiles (320×240, 480×320, 720×720).
3. **Input** — the keyboard/gamepad handlers become `onFrame` edge-detected
   button maps; the focus model already only uses real buttons.
4. **Storage and audio** — unchanged: `musicd` still owns SQLite and mpv, or a
   native audio sink behind the same provider interface.

## Performance evidence

`make bench` builds a 10,000 track (200 album) library in a temporary database
and measures the request path, not just the SQL:

| Benchmark                        | Measured (4-core i5-1155G7) |
| -------------------------------- | --------------------------- |
| `BenchmarkAPISearch10K`          | 2.7 – 4.7 ms per search     |
| `BenchmarkAPILibraryPage10K`     | 1.6 – 2.4 ms per page       |
| `BenchmarkAPIAlbums10K`          | 1.9 – 5.6 ms for 200 albums |
| `BenchmarkAPISongQueryWorstCase10K` | ~54 ms (every track matches) |

The first three cover what the UI does while typing and scrolling. The worst case
is the paginated `/api/library/songs?q=` path where the query matches all 10,000
rows and the service must sort them before applying `LIMIT`; the interactive
search preview uses a bounded, ranked candidate set instead
(`/api/search`), which is why it stays in single-digit milliseconds.

Numbers vary with host load (the development machine was carrying unrelated
workloads during measurement); treat them as an upper bound for this hardware,
not a cross-machine guarantee.
