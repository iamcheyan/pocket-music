# Screenshots

Every PNG in this directory was captured from a running Pocket Music instance —
no mock-ups, no synthetic renders. The UI was served by the compiled `musicd`
single-binary build (`MUSICD_WEB_DIR=frontend/dist MUSICD_MOCK=true`) with the
deterministic 240-track demo library, so the same command reproduces the same
screens.

## Method

`scripts/capture-screenshots.mjs` drives headless Chrome over the DevTools
protocol:

1. Seeds playback (`/api/play` with a 4-track queue, seek to 44 s) so Now
   Playing, lyrics, the queue and the player bar show real state.
2. For each viewport, applies `Emulation.setDeviceMetricsOverride` with the exact
   CSS width/height and device scale factor.
3. Navigates to a URL that fully determines the screen
   (`?view=…&tab=…&id=…&q=…&theme=…&lang=en&layout=…`).
4. Waits until the page's proof selector has rendered, then evaluates a layout
   audit, then captures `Page.captureScreenshot`.

The audit fails the run when:

- the proof selector for that screen never renders (the view did not load), or
- `documentElement.scrollWidth > clientWidth` (the page scrolls sideways), or
- any element extends past the viewport, ignoring content inside an intentional
  horizontal scroller (the tab rail) and the closed off-canvas drawer.

Latest run: **46 pages captured, 0 layout failures**, including DPI 1.5×, 2× and
3× where the CSS layout is byte-for-byte identical to 1× while the raster doubles
or triples.

## Matrix

| File | View | Viewport | Theme | Rendered content | Overflow |
| --- | --- | --- | --- | --- | --- |
| `home-1280x720-dark.png` | home | 1280x720 | dark | yes | 0 |
| `home-1280x720-light.png` | home | 1280x720 | light | yes | 0 |
| `library-1280x720-dark.png` | library | 1280x720 | dark | yes | 0 |
| `library-1280x720-light.png` | library | 1280x720 | light | yes | 0 |
| `albums-1280x720-dark.png` | albums | 1280x720 | dark | yes | 0 |
| `albums-1280x720-light.png` | albums | 1280x720 | light | yes | 0 |
| `artists-1280x720-dark.png` | artists | 1280x720 | dark | yes | 0 |
| `genres-1280x720-dark.png` | genres | 1280x720 | dark | yes | 0 |
| `folders-1280x720-dark.png` | folders | 1280x720 | dark | yes | 0 |
| `favorites-1280x720-dark.png` | favorites | 1280x720 | dark | yes | 0 |
| `album-detail-1280x720-dark.png` | album-detail | 1280x720 | dark | yes | 0 |
| `artist-detail-1280x720-dark.png` | artist-detail | 1280x720 | dark | yes | 0 |
| `now-playing-1280x720-dark.png` | now-playing | 1280x720 | dark | yes | 0 |
| `now-playing-1280x720-light.png` | now-playing | 1280x720 | light | yes | 0 |
| `queue-1280x720-dark.png` | queue | 1280x720 | dark | yes | 0 |
| `playlists-1280x720-dark.png` | playlists | 1280x720 | dark | yes | 0 |
| `playlist-1280x720-dark.png` | playlist | 1280x720 | dark | yes | 0 |
| `playlist-1280x720-light.png` | playlist | 1280x720 | light | yes | 0 |
| `search-1280x720-dark.png` | search | 1280x720 | dark | yes | 0 |
| `search-1280x720-light.png` | search | 1280x720 | light | yes | 0 |
| `settings-1280x720-dark.png` | settings | 1280x720 | dark | yes | 0 |
| `settings-1280x720-light.png` | settings | 1280x720 | light | yes | 0 |
| `stats-1280x720-dark.png` | stats | 1280x720 | dark | yes | 0 |
| `home-1920x1080-dark.png` | home | 1920x1080 | dark | yes | 0 |
| `now-playing-1920x1080-dark.png` | now-playing | 1920x1080 | dark | yes | 0 |
| `home-720x720-dark.png` | home | 720x720 | dark | yes | 0 |
| `home-720x720-light.png` | home | 720x720 | light | yes | 0 |
| `albums-720x720-dark.png` | albums | 720x720 | dark | yes | 0 |
| `now-playing-720x720-dark.png` | now-playing | 720x720 | dark | yes | 0 |
| `settings-720x720-dark.png` | settings | 720x720 | dark | yes | 0 |
| `home-640x480-dark.png` | home | 640x480 | dark | yes | 0 |
| `library-640x480-dark.png` | library | 640x480 | dark | yes | 0 |
| `now-playing-640x480-dark.png` | now-playing | 640x480 | dark | yes | 0 |
| `home-480x320-dark.png` | home | 480x320 | dark | yes | 0 |
| `home-480x320-light.png` | home | 480x320 | light | yes | 0 |
| `library-480x320-dark.png` | library | 480x320 | dark | yes | 0 |
| `now-playing-480x320-dark.png` | now-playing | 480x320 | dark | yes | 0 |
| `settings-480x320-dark.png` | settings | 480x320 | dark | yes | 0 |
| `home-320x240-dark.png` | home | 320x240 | dark | yes | 0 |
| `home-320x240-light.png` | home | 320x240 | light | yes | 0 |
| `library-320x240-dark.png` | library | 320x240 | dark | yes | 0 |
| `now-playing-320x240-dark.png` | now-playing | 320x240 | dark | yes | 0 |
| `queue-320x240-dark.png` | queue | 320x240 | dark | yes | 0 |
| `home-1280x720-dark-1.5x.png` | home | 1280x720-1.5x | dark | yes | 0 |
| `home-1280x720-dark-2x.png` | home | 1280x720-2x | dark | yes | 0 |
| `home-1280x720-dark-3x.png` | home | 1280x720-3x | dark | yes | 0 |

## What the matrix proves

- **Home / Library / Albums / Artists / Genres / Folders / Favourites** — the
  library is browsable from a shelf, a dense list, a genre chip grid and an
  expandable folder tree; every screen is API-driven.
- **Album detail / Artist detail** — generated or real cover art, metadata,
  play-all, add-to-queue and favourite actions, numbered track lists.
- **Now Playing** — cover, timeline, transport, playback mode, volume, favourite,
  synced lyrics and the next four queue entries; this is the primary screen at
  320×240 and 480×320.
- **Queue / Playlists / Playlist** — queue editing with drag handles, playlist
  management with import/export, all with the persistent player bar.
- **Search / Settings / Statistics** — grouped instant search, every preference
  (layout, theme, language, folder, scan, cache, audio output) and the listening
  statistics derived from real history rows.
- **Light and dark** — both themes are captured at 1280×720 for the core screens,
  with light variants at 720×720, 480×320 and 320×240 for Home.
- **320×240 … 1920×1080** — the same application, not a cut-down build: the
  sidebar becomes a drawer below 760 px, the page heading collapses into the
  breadcrumb below 340 px of height, and the player bar and bottom navigation
  keep transport reachable on a 240 px tall display.

## Reproduce

```sh
make build                       # build/musicd + frontend/dist
MUSICD_WEB_DIR=$PWD/apps/pocket-music/frontend/dist MUSICD_MOCK=true ./build/musicd &
npm --prefix apps/pocket-music/frontend install
make screenshots                 # or: node apps/pocket-music/scripts/capture-screenshots.mjs --base http://127.0.0.1:8765
```

Use `--audit-only` to run the layout audit without rewriting the PNGs, `--out` to
write elsewhere, and `--base` to point at a Vite dev server or a device.

## Design notes

- The palette is a warm olive/lime shell with a single accent, so cover art is the
  only strong colour on screen; light mode keeps the same hierarchy with a paper
  background.
- Covers are never blank: real art when available, otherwise a generated SVG that
  derives its palette from the album title.
- Type is sized in `clamp()` steps and rows in rem, so 1.5×/2×/3× DPI scale
  cleanly without layout changes.
- No native `select`, `input[type=range]` or scrollbars are used; the segmented
  pickers, listboxes, switches and sliders are app-drawn to stay legible and
  focusable on handheld panels.
