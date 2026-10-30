# HTTP API

All paths use the `/api` prefix. JSON responses use `application/json`; errors have the shape `{"error":{"code":"...","message":"..."}}`. Routes and response shapes below reflect the current `musicd` handlers. Several handlers do not enforce HTTP methods beyond the routes stated.

## Health and library

```http
GET /api/health
GET /api/home
GET /api/library/songs?q=blue&page=1&limit=50
GET /api/library/albums?q=blue
GET /api/library/artists?q=blue
GET /api/library/genres
GET /api/library/folders
GET /api/albums/{id}
GET /api/artists/{id}
GET /api/tracks/{id}
```

`/api/health` returns provider name, mock flag, and audio status/output, e.g. `{ "provider":"local", "mock":false, "audio":{"status":"ready","ao":"null"} }`.

Track objects contain `id`, `title`, `artist`, `album`, `genre`, `year`, `duration`, `path` (omitted if empty), and `added_at`. Songs returns `{ "items": [...], "total": n, "page": n, "limit": n }`; page defaults to 1, limit defaults to 50, and values above 500 reset to 50. Song `q` uses FTS. Album items contain `id`, `title`, `artist`, `year`, `track_count`; artist items contain `id`, `name`, `track_count`; genre/folder items contain `name`. Collections return `{ "items": [...], "total": n }`. `q` filters album/artist names and is ignored for genres/folders.

Home returns `{ "recently_added": [...], "recently_played": [...], "most_played": [...] }`, up to four track objects each. Album detail returns album metadata and tracks; artist detail returns `{ "id":"...", "name":"...", "tracks":[...] }`. Cover binary data is not currently provided by `/api/cover`.

## Search, playlists, and favorites

```http
GET    /api/search?q=blue&type=all
GET    /api/playlists
POST   /api/playlists
GET    /api/playlists/{id}
PUT    /api/playlists/{id}
DELETE /api/playlists/{id}
POST   /api/playlists/{id}/items
GET    /api/playlists/{id}/export?format=m3u8
POST   /api/playlists/import
POST   /api/favorites
DELETE /api/favorites
```

Search returns `{ "songs": [...], "albums": [...], "artists": [...], "playlists": [...] }`. `type` filters to `all`, `song`, `album`, `artist`, or `playlist`; song matches use FTS5 and album/artist/playlist matches use SQLite name/title queries, each limited to 50 results.

Playlist listing returns `{ "items": [{"id":1,"name":"...","created_at":"..."}] }`. Creation accepts `{ "name":"Favorites mix" }` and returns `{ "id":1,"name":"Favorites mix" }`. Detail returns `{ "id":1,"name":"...","tracks":[...] }`. Update accepts a name; delete returns success status. Item actions (`add`, `remove`, `reorder`) accept track ID and/or position fields and return updated playlist detail. Export supports M3U/M3U8. Import reads playlist content and returns the imported playlist result.

Favorites POST accepts `{ "kind":"track|album|artist", "id":"..." }`; `track_id` is an alternate identifier. DELETE reads `kind` and `id` from a JSON body and returns 204. No favorites listing route is present.

## Playback and queue

```http
POST /api/play
POST /api/player/pause
POST /api/player/resume
POST /api/player/next
POST /api/player/prev
POST /api/player/seek
POST /api/player/volume
GET  /api/player/state
POST /api/queue
```

Play accepts `{ "track_id":"track-id", "mode":"normal" }` and returns player state. Supported modes are `normal`, `repeat_all`, `repeat_one`, and `shuffle`; repeat and queue-advance behavior is handled by musicd. Player state has `track`, `playing`, `position`, `duration`, `volume`, `mode`, and `queue`. `/api/player/state` synchronizes the current position and duration from mpv IPC for local tracks. Seek accepts `{ "position":42.5 }`; volume accepts `{ "volume":75 }`; both return player state.

Queue accepts `{ "action":"add|play_next|remove|reorder", "track_id":"...", "index":0, "to":1 }`. Remove uses `index`; reorder moves the item at `index` to `to`. Response is `{ "queue": [...] }`.

## Media, scan, settings, and stats

```http
GET  /api/cover/{album_id}
GET  /api/lyrics/{track_id}
GET  /api/media/{track_id}
POST /api/scan
GET  /api/scan
GET  /api/scan/status
POST /api/scan/cancel
GET  /api/settings
PUT  /api/settings
GET  /api/stats
```

`/api/cover/{album_id}` returns cover bytes in priority order: stored album cover path, `folder.jpg`, `cover.jpg`, then embedded artwork extracted with ffmpeg. Missing album/art returns 404. Lyrics return timestamped lines parsed from a sibling `.lrc`, with embedded lyrics from ffprobe as fallback. Media supports HTTP Range through `http.ServeContent`.

Scan POST returns `{ "started":true }` or a structured conflict if scanning is already active. Both GET status paths return `scanning`, `found`, `total`, and optional `current`/`error`; cancel POST returns `{ "cancelled":true }`. Settings GET returns `{ "settings": {"key":"value"} }`; PUT accepts that wrapper and applies `music_folder`, `audio_output`, and other stored settings. `MUSICD_AO=null` is the headless audio default. Stats returns accumulated play time and most played tracks/artists/albums.
