# Pocket Music

Pocket Music is a desktop Linux music player built as a React + TypeScript web
UI, a Go `musicd` service, SQLite storage and mpv audio playback. The UI talks to
the service over HTTP JSON; it never scans a directory, reads an audio file or
controls the audio engine itself.

The same repository builds two things:

- **Development mode** — Vite on `:5173` with `/api` proxied to `musicd` on
  `:8765`, giving hot reload while the Go service keeps its state.
- **Single-binary desktop mode** — `make build` compiles the UI into
  `frontend/dist` and the service serves it from `MUSICD_WEB_DIR`, so a desktop
  session is one process with no Node runtime.

## Quick start

Prerequisites: Go 1.24+, Node.js 20+, npm, and mpv (audio is optional — the
default output is `null`).

```sh
npm --prefix apps/pocket-music/frontend install
MUSICD_MOCK=true make dev      # demo library, http://127.0.0.1:5173
```

For a real library:

```sh
MUSICD_MUSIC_DIR="$HOME/Music" make dev
```

then open **Settings → Scan now**. `make serve` runs the compiled single-binary
build instead.

## Configuration

| Variable           | Default                 | Purpose                                       |
| ------------------ | ----------------------- | --------------------------------------------- |
| `MUSICD_ADDR`      | `:8765`                 | HTTP listen address                           |
| `MUSICD_DB`        | `musicd.db`             | SQLite file                                   |
| `MUSICD_MUSIC_DIR` | `$HOME/Music`           | Library root scanned on demand                |
| `MUSICD_MOCK`      | `false`                 | Seed a deterministic 240-track demo library   |
| `MUSICD_AO`        | `null`                  | mpv audio output (`null`, `pulse`, `alsa`, …) |
| `MUSICD_MPV`       | `mpv`                   | mpv executable                                |
| `MUSICD_CACHE`     | `$TMPDIR/musicd-cache`  | Extracted cover art cache                     |
| `MUSICD_WEB_DIR`   | *(unset)*               | Serve the compiled UI instead of only `/api`  |

Persisted settings (language, theme, music folder, cache size, auto scan, audio
output) live in the `settings` table and take precedence over the environment on
the next start.

## Demo library

`MUSICD_MOCK=true` seeds, once, into an empty database:

- 8 artists × 3 albums × 10 tracks = **240 tracks** with realistic titles,
  genres, years, durations and artist/album folder paths;
- listening history for roughly a quarter of the tracks (plays, skips,
  completion rate) so Home and Statistics are populated;
- 8 favourite tracks, 4 favourite albums, 3 favourite artists;
- 3 playlists of 5–7 tracks.

Mock playback runs on the service clock instead of mpv, so the Now Playing
screen, lyrics panel, queue advancement and completion accounting all work on a
headless host. Demo tracks are database rows only — nothing is read from disk.

## Product surface

- **Home** — recently added, recently played, most played, favourite tracks,
  favourite albums.
- **Library** — songs (paginated, 100 per page), albums, artists, genres,
  folders (expandable, backed by a folder filter), favourites.
- **Detail pages** — album (cover, year, duration, play all, add to queue,
  favourite, numbered tracks) and artist (albums plus all tracks).
- **Now Playing** — art, title, artist, album links, timeline, transport,
  favourite, playback mode, volume, synced lyrics, up-next preview.
- **Queue** — play next, add, remove, clear, drag-reorder.
- **Playlists** — create, rename, delete, add/remove/reorder, M3U/M3U8 import
  and export.
- **Search** — instant, grouped by songs/albums/artists/playlists.
- **Statistics** — listening time, plays, skips, completion, top artists and
  albums, most played, most skipped.
- **Settings** — layout, theme, language, music folder, scan with progress and
  cancel, scan on startup, cover cache size, audio output, service status,
  keyboard help.

## Layouts

The layout is a root data attribute (`data-layout`) so CSS and screenshots can
target it deterministically.

| Layout    | Structure                                                   |
| --------- | ----------------------------------------------------------- |
| `music`   | fixed sidebar, album-forward content, persistent player bar |
| `library` | tab rail floated left of a dense list (Foobar-style)        |
| `minimal` | no sidebar, drawer navigation, compact top bar, Now Playing |

At widths below 760 px the sidebar becomes a drawer for every layout and a
bottom navigation bar appears; below 340 px of height the page heading collapses
into the breadcrumb so content rows stay usable at 480×320 and 320×240.

## Input

- **Keyboard** — `Space` play/pause, `←/→` previous/next, `↑/↓` volume,
  `/` focus search, `Esc` home, `Enter` activate.
- **Mouse / touch** — every control is a real button; sliders support pointer
  drag and `role="slider"` keyboard stepping.
- **Gamepad** — A toggles playback, shoulders skip, D-pad changes volume, left
  stick jumps between Home and Now Playing. Buttons are edge-triggered so a held
  button fires once.
- **Custom controls** — segmented pickers, listbox selects, switches and sliders
  are all drawn by the app; no native `select` or `input[type=range]` is used, so
  focus and appearance stay consistent on handhelds and mirrored displays.

## Verification

```sh
make test          # 19 Go tests, tsc, 16 unit tests, i18n parity (141 keys × 3)
make bench         # API benchmarks against a 10,000 track library
make screenshots   # re-capture the matrix + layout audit against a live instance
```

Tests cover schema migration, scanning and incremental rescans, stale-file
pruning, metadata import, search grouping and FTS escaping, folder grouping and
filtering, favourites, queue ordering, play-next, skip counting, completion,
playback modes, covers (including the generated fallback), LRC and embedded
lyrics precedence, playlist CRUD, M3U/M3U8 round-trips, settings validation and
SPA static serving.

Screenshots and the per-page overflow audit are documented in
[SCREENSHOTS.md](SCREENSHOTS.md).

## Documentation map

- [ARCHITECTURE.md](ARCHITECTURE.md) — layers, playback state machine, providers,
  PJOS migration path.
- [DATABASE.md](DATABASE.md) — schema, indexes, FTS5 and scan bookkeeping.
- [API.md](API.md) — every route, request and response.
- [ROADMAP.md](ROADMAP.md) — implemented surface and remaining work.
- [SCREENSHOTS.md](SCREENSHOTS.md) — capture matrix, layout audit, reproduce.
