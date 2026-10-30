# Roadmap

Status legend: **done** = implemented and covered by tests/screenshots in this
repository; **partial** = functional but limited; **planned** = designed, not
implemented.

## Delivered

| Area | State | Evidence |
| --- | --- | --- |
| Service boundary (UI → `musicd` → SQLite/mpv) | done | `docs/ARCHITECTURE.md`, API tests |
| SQLite schema, FTS5 index, migrations | done | `TestMigrationAndCoreAPI`, `docs/DATABASE.md` |
| Scanner: MP3/FLAC/AAC/M4A/OGG/WAV/APE, incremental, prune, cancel | done | `TestScanIncrementalAndCancellation`, `TestScanRemovesDeletedFilesAndReportsProgress` |
| Metadata extraction (ffprobe) with sensible fallbacks | done | scan tests, demo library |
| Library: songs (paged), albums, artists, genres, folders, favourites | done | `TestFoldersGroupingAndFiltering`, screenshots |
| Search: instant, grouped, FTS-escaped, bounded preview | done | `TestSearchGroupsAndFilters`, benchmarks |
| Playback: play/pause/seek/volume, modes, queue engine | done | `TestPlaybackModesQueueOperationsAndSkipAccounting` |
| Skip count and completion rate accounting | done | same test |
| Queue: play next, add, remove, reorder, clear | done | queue test, queue screenshot |
| Favourites: tracks, albums, artists | done | `TestFavoritesLifecycleAndHomeSections` |
| Playlists: CRUD, ordering, M3U/M3U8 import and export | done | `TestPlaylistM3URoundTripAndMalformedRollback`, `TestImportM3U8WithBOMAndRelativePaths` |
| Covers: stored, `folder.jpg`/`cover.jpg`, embedded, generated | done | `TestCoverLyricsAndArtistAlbumDetail` |
| Lyrics: LRC files, embedded tags, synced highlight | done | `TestLyricsFileTakesPrecedence`, now-playing screenshots |
| Statistics: play time, plays, skips, completion, top artists/albums | done | `TestStatsAggregatesHistory`, stats screenshot |
| Settings: layout, theme, language, folder, scan, cache, audio output | done | `TestSettingsValidationAndStaticServing`, settings screenshots |
| i18n: English, 简体中文, 日本語 | done | `i18n-contract.ts` (141 keys × 3) |
| Responsive 320×240 → 1920×1080, DPI 1×/1.5×/2×/3× | done | 46 screenshots, 0 layout failures |
| Input: mouse, touch, keyboard, gamepad | done | `App.tsx` handlers, live interaction at 320×240 |
| Single-binary desktop build | done | `TestSettingsValidationAndStaticServing` (SPA fallback), `make serve` |
| 10,000 track performance | done | `make bench` (see ARCHITECTURE) |

## Partial

| Area | Gap |
| --- | --- |
| Audio engine | mpv is launched and controlled over IPC; no gapless playback or crossfade, and no ReplayGain/dynamic range handling |
| Gamepad | Transport and volume are mapped; full D-pad focus traversal (launcher-style) is not |
| Cover cache | size setting is stored and surfaced but not yet enforced as an eviction policy |
| Playlist import | matches by absolute path then path tail; no re-link UI for missing entries |
| Folder browsing | flat two-level grouping; no nested tree or breadcrumb navigation |

## Planned

1. **Remote providers** — implement `BilibiliProvider`, `NavidromeProvider` and
   `JellyfinProvider` behind `MusicProvider` (credentials stay service-side,
   items land in the same tables, UI unchanged).
2. **PJOS runtime adapter** — a PocketJS client over the same API, with device
   profiles for 320×240, 480×320 and 720×720 panels, `onFrame` input mapping and
   KMS rendering.
3. **DAP hardware profile** — physical button map, rotary volume, hold switch,
   safe-shutdown handling, battery in the status bar.
4. **Library intelligence** — smart playlists (most played, never played, recent
   additions), duplicate detection, multi-disc handling, embedded composer/work
   tags for classical libraries.
5. **Playback depth** — gapless/crossfade, ReplayGain, output device hot-swap,
   sleep timer.
6. **Sync** — provider-backed download/upload queue with conflict resolution and
   offline pinning for handhelds.
7. **Accessibility** — full screen-reader pass, high-contrast theme, reduced
   motion is already honoured by CSS.
8. **Packaging** — `.deb`/AppImage with a bundled service, plus a Flatpak
   manifest; currently the single binary is built from source.

## Known limitations

- Audio output is whatever mpv and the host provide; the default `null` sink
  keeps CI and headless hosts working but produces no sound.
- Demo-mode tracks are metadata only, so `/api/media/{id}` returns 404 for them.
- The PocketJS adapter and hardware verification are out of scope for this
  repository; desktop Linux is the shipped target.
