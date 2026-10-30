# Verification — pocket-music @ 2026-10-31

- Verdict: PASS
- Commit: `494d1987a1eb5d2d84b7460fc27deeeab8982d21` (fixes + screenshots; this document is the following commit)
- Remote: `494d1987a1eb5d2d84b7460fc27deeeab8982d21`
- Runtime: `make build` then `MUSICD_ADDR=127.0.0.1:8765 MUSICD_DB=/tmp/pm/music.db MUSICD_CACHE=/tmp/pm/cache MUSICD_WEB_DIR=$PWD/apps/pocket-music/frontend/dist MUSICD_MOCK=true ./build/musicd` ; port 8765 ; backend Go `musicd`
- Toolchain: go 1.26.0, node v22.23.3, Chrome 155.0.8059.84, mpv present (`/usr/bin/mpv`)

## Test gate
| command | result | notes |
|---|---|---|
| `go test ./...` | PASS | 19 Go tests, 0 failures (includes API/contract/EOF tests) |
| `npm run typecheck` | PASS | `tsc --noEmit` |
| `npm run test:unit` | PASS | 16 node:test unit tests |
| i18n contract | PASS | 141 keys across en/zh-CN/ja |
| `node scripts/capture-screenshots.mjs` | PASS | 46 pages captured, 0 layout failures, DPI 1.5×/2×/3× overflow 0 |

`make test` runs the whole sequence; it finished green.

## Runtime / playback
The service ran in its documented single-binary production form (`make serve` path) with
`MUSICD_MOCK=true`, real mpv engine (`"audio":{"engine":"mpv","status":"ready","ao":"null"}`,
library 240 tracks / 24 albums / 8 artists). Drive through the real API:

- `POST /api/play` → `playing=true`, position 0.0002 s, duration 228 s, track "Closing Light 10"
- after 2 s `GET /api/player/state` → `position=2.001 s` (clock advances in real time)
- `POST /api/player/pause` → `playing=false`; `resume` → `true`
- `POST /api/player/seek {position:5}` → `position=5`
- `POST /api/player/volume {volume:42}` → `volume=42`
- `POST /api/player/next` → queue advanced to "Closing Light 100"

Web probe (`tools/probe.mjs`) on the production build at 1280×720 for
`?view=now&theme=light&layout=minimal`, `?view=library&theme=dark&layout=music`,
`?view=settings&theme=dark&layout=library`: HTTP 200, no console errors, no page errors,
no failed/4xx/5xx requests, no horizontal overflow.

## Screenshot matrix
All 46 files were regenerated this run by the repo's own capture script against the
production `musicd` at 127.0.0.1:8765; each entry carries a rendered-text proof and an
overflow audit in `capture-report.json`.

| screen | viewport | theme | file | note |
|---|---|---|---|---|
| home | 1280x720 | dark | `apps/pocket-music/docs/screenshots/home-1280x720-dark.png` | rendered proof: Browse your collecti |
| library | 1280x720 | dark | `apps/pocket-music/docs/screenshots/library-1280x720-dark.png` | rendered proof: 01Closing Light 10No |
| albums | 1280x720 | dark | `apps/pocket-music/docs/screenshots/albums-1280x720-dark.png` | rendered proof: Analogue Summer ILow |
| artists | 1280x720 | dark | `apps/pocket-music/docs/screenshots/artists-1280x720-dark.png` | rendered proof: AIAiko Sera30 tracks |
| favorites | 1280x720 | dark | `apps/pocket-music/docs/screenshots/favorites-1280x720-dark.png` | rendered proof: 01Small Hours 07Nova |
| genres | 1280x720 | dark | `apps/pocket-music/docs/screenshots/genres-1280x720-dark.png` | rendered proof: Ambient30 |
| folders | 1280x720 | dark | `apps/pocket-music/docs/screenshots/folders-1280x720-dark.png` | rendered proof: present |
| album-detail | 1280x720 | dark | `apps/pocket-music/docs/screenshots/album-detail-1280x720-dark.png` | rendered proof: Night Cartography I |
| artist-detail | 1280x720 | dark | `apps/pocket-music/docs/screenshots/artist-detail-1280x720-dark.png` | rendered proof: Nova Lantern |
| now-playing | 1280x720 | dark | `apps/pocket-music/docs/screenshots/now-playing-1280x720-dark.png` | rendered proof: Opening Light 11 |
| queue | 1280x720 | dark | `apps/pocket-music/docs/screenshots/queue-1280x720-dark.png` | rendered proof: Opening Light 21Nova |
| playlists | 1280x720 | dark | `apps/pocket-music/docs/screenshots/playlists-1280x720-dark.png` | rendered proof: FFocus Session7 trac |
| playlist | 1280x720 | dark | `apps/pocket-music/docs/screenshots/playlist-1280x720-dark.png` | rendered proof: present |
| search | 1280x720 | dark | `apps/pocket-music/docs/screenshots/search-1280x720-dark.png` | rendered proof: 01Closing Light 160V |
| settings | 1280x720 | dark | `apps/pocket-music/docs/screenshots/settings-1280x720-dark.png` | rendered proof: AppearanceLayoutMusi |
| stats | 1280x720 | dark | `apps/pocket-music/docs/screenshots/stats-1280x720-dark.png` | rendered proof: Listening time23 h 2 |
| home | 1280x720 | light | `apps/pocket-music/docs/screenshots/home-1280x720-light.png` | rendered proof: Browse your collecti |
| library | 1280x720 | light | `apps/pocket-music/docs/screenshots/library-1280x720-light.png` | rendered proof: 01Closing Light 10No |
| albums | 1280x720 | light | `apps/pocket-music/docs/screenshots/albums-1280x720-light.png` | rendered proof: Analogue Summer ILow |
| now-playing | 1280x720 | light | `apps/pocket-music/docs/screenshots/now-playing-1280x720-light.png` | rendered proof: Opening Light 11 |
| playlist | 1280x720 | light | `apps/pocket-music/docs/screenshots/playlist-1280x720-light.png` | rendered proof: present |
| search | 1280x720 | light | `apps/pocket-music/docs/screenshots/search-1280x720-light.png` | rendered proof: 01Closing Light 160V |
| settings | 1280x720 | light | `apps/pocket-music/docs/screenshots/settings-1280x720-light.png` | rendered proof: AppearanceLayoutMusi |
| home | 1920x1080 | dark | `apps/pocket-music/docs/screenshots/home-1920x1080-dark.png` | rendered proof: Browse your collecti |
| now-playing | 1920x1080 | dark | `apps/pocket-music/docs/screenshots/now-playing-1920x1080-dark.png` | rendered proof: Opening Light 11 |
| home | 720x720 | dark | `apps/pocket-music/docs/screenshots/home-720x720-dark.png` | rendered proof: Browse your collecti |
| now-playing | 720x720 | dark | `apps/pocket-music/docs/screenshots/now-playing-720x720-dark.png` | rendered proof: Opening Light 11 |
| settings | 720x720 | dark | `apps/pocket-music/docs/screenshots/settings-720x720-dark.png` | rendered proof: AppearanceLayoutMusi |
| albums | 720x720 | dark | `apps/pocket-music/docs/screenshots/albums-720x720-dark.png` | rendered proof: Analogue Summer ILow |
| home | 720x720 | light | `apps/pocket-music/docs/screenshots/home-720x720-light.png` | rendered proof: Browse your collecti |
| home | 640x480 | dark | `apps/pocket-music/docs/screenshots/home-640x480-dark.png` | rendered proof: Browse your collecti |
| library | 640x480 | dark | `apps/pocket-music/docs/screenshots/library-640x480-dark.png` | rendered proof: 01Closing Light 10No |
| now-playing | 640x480 | dark | `apps/pocket-music/docs/screenshots/now-playing-640x480-dark.png` | rendered proof: Opening Light 11 |
| home | 480x320 | dark | `apps/pocket-music/docs/screenshots/home-480x320-dark.png` | rendered proof: Browse your collecti |
| library | 480x320 | dark | `apps/pocket-music/docs/screenshots/library-480x320-dark.png` | rendered proof: 01Closing Light 10No |
| now-playing | 480x320 | dark | `apps/pocket-music/docs/screenshots/now-playing-480x320-dark.png` | rendered proof: Opening Light 11 |
| settings | 480x320 | dark | `apps/pocket-music/docs/screenshots/settings-480x320-dark.png` | rendered proof: AppearanceLayoutMusi |
| home | 480x320 | light | `apps/pocket-music/docs/screenshots/home-480x320-light.png` | rendered proof: Browse your collecti |
| home | 320x240 | dark | `apps/pocket-music/docs/screenshots/home-320x240-dark.png` | rendered proof: Browse your collecti |
| library | 320x240 | dark | `apps/pocket-music/docs/screenshots/library-320x240-dark.png` | rendered proof: 01Closing Light 10No |
| now-playing | 320x240 | dark | `apps/pocket-music/docs/screenshots/now-playing-320x240-dark.png` | rendered proof: Opening Light 11 |
| queue | 320x240 | dark | `apps/pocket-music/docs/screenshots/queue-320x240-dark.png` | rendered proof: Opening Light 21Nova |
| home | 320x240 | light | `apps/pocket-music/docs/screenshots/home-320x240-light.png` | rendered proof: Browse your collecti |
| home | 1280x720-1.5x | dark | `apps/pocket-music/docs/screenshots/home-1280x720-dark-1.5x.png` | rendered proof: dpi |
| home | 1280x720-2x | dark | `apps/pocket-music/docs/screenshots/home-1280x720-dark-2x.png` | rendered proof: dpi |
| home | 1280x720-3x | dark | `apps/pocket-music/docs/screenshots/home-1280x720-dark-3x.png` | rendered proof: dpi |

Count: 46 screenshots, 46 unique hashes (0 duplicates). `tools/check_shots.py`: all PNGs valid,
dimensions match, minimum per-image RGB stddev 19.09, no blank frames. `capture-report.json`
records `overflow: []` for every entry.

## Fixed
- `make build` wrote the binary to `apps/build/musicd` (`go build -o ../../$(BUILD_DIR)/musicd`
  resolved relative to the backend dir), while `make serve` runs `./build/musicd` — so the
  documented single-binary path failed with "No such file or directory". → build to
  `$(abspath $(BUILD_DIR))/musicd` → `Makefile` → `make build && make serve` now starts
  `./build/musicd` serving the compiled UI (port 8765, screenshots above).
- The capture script wrote DPI screenshots as `home-1280x720-dark` + `@1.5x.png` while
  `docs/SCREENSHOTS.md` (and the previously committed files) use the `-1.5x` suffix. Result:
  every DPI capture produced an untracked `@`-prefixed file and left the tracked `-1.5x` images stale.
  → emit `home-1280x720-dark-<scale>x.png` and `1280x720-<scale>x` labels
  → `apps/pocket-music/scripts/capture-screenshots.mjs` → all three DPI files are regenerated
  and the report filenames match the files on disk (`home-1280x720-dark-1.5x.png`,
  `...-2x.png`, `...-3x.png`).

## Blocked / open questions
- None blocking. Audio output used `ao=null` (the documented headless default), so sound was
  not heard; playback state, position advance and transport commands were verified against
  the real mpv IPC process.

## Not verified here
- 掌机真机（RG34XX / PJOS-LINUX）未验证；本轮只在本机 headless Chrome 里跑真实进程。
- Scanning a real music folder and remote providers (Bilibili/Navidrome/Jellyfin) — not implemented/out of scope.
