# Screenshot index

The PNGs below were captured from the running Vite application backed by `musicd` in mock mode using Chromium headless mode. They are not synthetic. Each file has a distinct SHA-256 and its PNG header dimensions match the filename; the 17 files were verified with a Python `glob` + `hashlib` check. View, library tab, theme, language, and search query are URL parameters (`?view=…&tab=…&theme=…&lang=…&q=…`), so every capture is reproducible without clicking.

## Captures

| Filename | Resolution | Theme | View / proof | Capture command |
| --- | --- | --- | --- | --- |
| `home-1280x720-dark.png` | 1280×720 | Dark | API backed Home | `chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars --virtual-time-budget=5000 --force-device-scale-factor=1 --window-size=1280,720 --screenshot=apps/pocket-music/docs/screenshots/home-1280x720-dark.png 'http://127.0.0.1:5175/?view=home&theme=dark&lang=en'` |
| `library-1280x720-dark.png` | 1280×720 | Dark | Library, Songs tab | Replace the URL with `?view=library&tab=songs&theme=dark&lang=en` |
| `albums-1280x720-dark.png` | 1280×720 | Dark | Album records | Replace the URL with `?view=library&tab=albums&theme=dark&lang=en` |
| `artists-1280x720-dark.png` | 1280×720 | Dark | Artist records | Replace the URL with `?view=library&tab=artists&theme=dark&lang=en` |
| `now-playing-1280x720-dark.png` | 1280×720 | Dark | Playing Demo Track 001 | Replace the URL with `?view=now&theme=dark&lang=en` |
| `queue-1280x720-dark.png` | 1280×720 | Dark | Queue with one queued track | Replace the URL with `?view=queue&theme=dark&lang=en` |
| `playlists-1280x720-dark.png` | 1280×720 | Dark | Playlist manager | Replace the URL with `?view=playlists&theme=dark&lang=en` |
| `search-1280x720-dark.png` | 1280×720 | Dark | Search results for "Demo" | Replace the URL with `?view=search&q=Demo&theme=dark&lang=en` |
| `settings-1280x720-dark.png` | 1280×720 | Dark | Service status and preferences | Replace the URL with `?view=settings&theme=dark&lang=en` |
| `home-720x720-dark.png` | 720×720 | Dark | Home, square viewport | Same command with `--window-size=720,720` and `?view=home&theme=dark&lang=en` |
| `home-720x720-light.png` | 720×720 | Light | Home, light theme | Same command with `--window-size=720,720` and `?view=home&theme=light&lang=en` |
| `settings-720x720-light.png` | 720×720 | Light | Settings, light theme | Same command with `--window-size=720,720` and `?view=settings&theme=light&lang=en` |
| `home-480x320-dark.png` | 480×320 | Dark | Home, compact viewport | Same command with `--window-size=480,320` and `?view=home&theme=dark&lang=en` |
| `home-480x320-light.png` | 480×320 | Light | Home, compact light theme | Same command with `--window-size=480,320` and `?view=home&theme=light&lang=en` |
| `library-480x320-dark.png` | 480×320 | Dark | Library, compact viewport | Same command with `--window-size=480,320` and `?view=library&tab=songs&theme=dark&lang=en` |
| `home-320x240-dark.png` | 320×240 | Dark | Home at minimum viewport | Same command with `--window-size=320,240` and `?view=home&theme=dark&lang=en` |
| `home-320x240-light.png` | 320×240 | Light | Home, minimum viewport light theme | Same command with `--window-size=320,240` and `?view=home&theme=light&lang=en` |

The 320×240 view was interacted with in Chromium: Library navigation, song playback, Queue navigation, and Settings layout/language/theme selection all worked.

## Reproduce

From the repository root, start the application with `MUSICD_MOCK=true make dev`, then substitute the filename and viewport into the Chromium command. For screenshots with a named application state, navigate to that view first in the running application. `--force-device-scale-factor=1` is used for the 1280×720 files so their pixel dimensions match the viewport.