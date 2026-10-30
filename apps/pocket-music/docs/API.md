# HTTP API

All API paths use the `/api` prefix. JSON requests and responses use `application/json`; errors follow `{"error":{"code":"...","message":"..."}}`. The examples below describe the required contract. They are not claims that each route has been implemented or exercised; confirm field names against the server before integration clients depend on them.

## Health and library

```http
GET /api/health
```

Returns service/provider identity, mock status, and audio engine status. Example shape:

```json
{"provider":"local","mock":false,"audio":{"engine":"mpv","status":"ready"}}
```

```http
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

List endpoints return JSON collections; song results should be paginated, and album detail includes tracks and cover information. Example list response:

```json
{"items":[{"id":"track-id","title":"Blue Train","artist":"Artist","album":"Album","duration":120.5}],"page":1,"limit":50,"total":1}
```

`GET /api/home` supplies Recently Added, Recently Played, and Most Played sections.

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

Search `type` accepts `all`, `song`, `album`, `artist`, or `playlist`. Playlist item operations support add, remove, and reorder; import/export support M3U and M3U8. Favorite operations identify a target type and ID in the request body. Exact body schemas should track the implementation.

Example create request:

```http
POST /api/playlists
Content-Type: application/json
```

```json
{"name":"Favorites mix"}
```

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

`POST /api/play` accepts a track ID and mode (`normal`, `repeat_all`, `repeat_one`, or `shuffle`). Player state includes current track, progress, volume, mode, and queue. Queue operations support add, play-next, remove, and reorder. Example:

```json
{"track_id":"track-id","mode":"normal"}
```

Action-specific request fields and success responses must be confirmed from the handlers.

## Media, scan, and settings

```http
GET  /api/cover/{album_id}
GET  /api/lyrics/{track_id}
GET  /api/media/{track_id}
POST /api/scan
GET  /api/scan/status
GET  /api/settings
PUT  /api/settings
GET  /api/stats
```

Media streaming supports HTTP Range requests. Lyrics return parsed LRC timing data when available. Scan status reports current progress using the implementation's polling or event mechanism. Settings include language, theme, music folder, cache size, scan controls, audio output, and reserved proxy/network fields. Stats include play time and most played artists/albums. Media, binary cover, and status responses may use non-JSON content types as appropriate.
