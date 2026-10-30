# Screenshot index

The PNGs below were captured from the running Vite application backed by `musicd` in mock mode using Chromium headless mode. They are not synthetic. PNG dimensions were checked from their file headers after capture. Stateful views were selected in the running React application before saving their screenshot.

## Captures

| Filename | Resolution | Theme | View / proof | Capture command |
| --- | --- | --- | --- | --- |
| `home-1280x720-dark.png` | 1280×720 | Dark | API backed Home | `chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars --force-device-scale-factor=1 --window-size=1280,720 --screenshot=apps/pocket-music/docs/screenshots/home-1280x720-dark.png http://127.0.0.1:5175/` |
| `library-1280x720-dark.png` | 1280×720 | Dark | Library | Same command with Library selected |
| `albums-1280x720-dark.png` | 1280×720 | Dark | Album records | Same command with Library → Albums selected |
| `artists-1280x720-dark.png` | 1280×720 | Dark | Artist records | Same command with Library → Artists selected |
| `now-playing-1280x720-dark.png` | 1280×720 | Dark | Playing Demo Track 001 | Same command after selecting a song |
| `queue-1280x720-dark.png` | 1280×720 | Dark | Queue view | Same command with Queue selected |
| `playlists-1280x720-dark.png` | 1280×720 | Dark | Playlist manager | Same command with Playlists selected |
| `search-1280x720-dark.png` | 1280×720 | Dark | Search view | Same command with Search selected |
| `settings-1280x720-dark.png` | 1280×720 | Dark | Service status and preferences | Same command with Settings selected |
| `home-720x720-dark.png` | 720×720 | Dark | Home, square viewport | `chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars --window-size=720,720 --screenshot=apps/pocket-music/docs/screenshots/home-720x720-dark.png http://127.0.0.1:5175/` |
| `home-720x720-light.png` | 720×720 | Light | Home, light theme | Same command after selecting Light |
| `settings-720x720-light.png` | 720×720 | Light | Settings, light theme | Same command after selecting Settings and Light |
| `home-480x320-dark.png` | 480×320 | Dark | Home, compact viewport | `chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars --window-size=480,320 --screenshot=apps/pocket-music/docs/screenshots/home-480x320-dark.png http://127.0.0.1:5175/` |
| `home-480x320-light.png` | 480×320 | Light | Home, compact light theme | Same command after selecting Light |
| `library-480x320-dark.png` | 480×320 | Dark | Library, compact viewport | Same command with Library selected |
| `home-320x240-dark.png` | 320×240 | Dark | Home at minimum viewport | `chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars --window-size=320,240 --screenshot=apps/pocket-music/docs/screenshots/home-320x240-dark.png http://127.0.0.1:5175/` |
| `home-320x240-light.png` | 320×240 | Light | Home, minimum viewport light theme | Same command after selecting Light |

The 320×240 view was interacted with in Chromium: Library navigation, song playback, Queue navigation, and Settings layout/language/theme selection all worked.

## Reproduce

From the repository root, start the application with `MUSICD_MOCK=true make dev`, then substitute the filename and viewport into the Chromium command. For screenshots with a named application state, navigate to that view first in the running application. `--force-device-scale-factor=1` is used for the 1280×720 files so their pixel dimensions match the viewport.