# Roadmap

Status reflects integration evidence captured on 2026-10-30 from the running application, direct API calls, tests, benchmark, and Chromium screenshots.

## Implemented in the current source

- [x] Root Makefile provides combined development startup and backend/frontend check targets.
- [x] Go service initialization creates SQLite tables, indexes, FTS5, and a version 1 marker; this is not yet a sequential migration framework.
- [x] Local scanner walks the configured root, recognizes MP3, FLAC, AAC, M4A, OGG, WAV, and APE, and invokes ffprobe for metadata.
- [x] Scan start/status/cancel routes and progress state are present.
- [x] Song listing is paginated and title ordered; FTS5 supports song search.
- [x] Mock seeding creates deterministic sample artists, albums, and tracks when the database is empty.
- [x] HTTP routes cover library, playlists, favorites, player state/actions, queue, settings, scan, media, lyrics, and stats.
- [x] mpv uses JSON IPC and `--ao=null` by default; `/api/player/state` synchronizes position, duration, pause, and completion for local tracks.

## Verification completed

- [x] `make test` passed: Go tests pass and TypeScript `tsc --noEmit` exits zero.
- [x] `npm --prefix apps/pocket-music/frontend run build` passed.
- [x] Direct curl checks covered health, home, library collections, search, playlists, settings, playback, queue, scan status, and stats.
- [x] Generated a 2-second WAV with ffmpeg; scan indexed its title, artist, album, year, duration, and path.
- [x] 10,000-row paired search/list benchmark: 23.35 ms/op (single iteration, see README).
- [x] Chromium exercised the Home, Library, playback, queue, and Settings flows at 320×240; screenshots cover all named views, Light/Dark examples, and all required viewport dimensions.

## Remaining work

- [ ] Execute `mpv` playback on a real local audio file with `--ao=null`; this was not exercised during integration.
- [ ] Validate the full screen and DPI matrix at 1x, 1.5x, 2x, and 3x. Responsive behavior and 320×240 were checked; all DPI combinations were not individually measured.
- [ ] Add and execute a true multi-version schema migration path; current database startup initializes schema version 1.

## Known limitations

Remote Bilibili, Navidrome, and Jellyfin providers are extension points documented in [ARCHITECTURE.md](ARCHITECTURE.md) and are not implemented. Real mpv playback and full DPI matrix remain unverified on this host.

## Future PJOS migration

1. Keep UI workflows behind the `musicd` API contract.
2. Replace Vite development hosting with target PJOS runtime integration while preserving API semantics.
3. Validate transport, focus/input handling, storage configuration, and media rendering on each target device.
4. Add device-specific local and remote providers without moving service responsibilities into UI code.
