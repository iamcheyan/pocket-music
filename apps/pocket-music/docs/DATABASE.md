# Database

The service opens SQLite through the pure Go `modernc.org/sqlite` driver. `MUSICD_DB` selects the database path (default `musicd.db`). Startup initializes tables and indexes and records schema version 1. The version marker is present, but the current implementation does not yet provide an ordered migration runner.

## Current tables

| Table | Current columns / purpose |
| --- | --- |
| `schema_version` | `version`; initialized to 1. |
| `artists` | `id`, unique required `name`. |
| `albums` | `id`, `title`, `artist_id`, `year`, `cover`. |
| `tracks` | `id`, `title`, artist/album IDs, `genre`, `year`, `duration`, unique required `path`, `mtime`, `size`, `added_at`. |
| `tracks_fts` | FTS5 virtual table with unindexed `track_id` and indexed title, artist, album, genre. |
| `playlists` | `id`, `name`, `created_at`. |
| `playlist_items` | `playlist_id`, ordered `position`, `track_id`; composite primary key on playlist and position. |
| `favorites` | `kind`, `entity_id`, `created_at`; composite primary key on kind and entity. |
| `history` | `track_id`, `last_played`, `play_count`, `skip_count`, `completion_rate`. |
| `settings` | key/value pairs. |

Startup inserts settings defaults when absent: language `en`, theme `system`, music folder set to the configured root, cache size `512`, auto scan `false`, audio output `null`, and empty proxy value.

## Indexes and search

Explicit indexes cover track title, artist ID, and album ID. FTS5 indexes track title, artist, album, and genre. Song listing uses title ordering with limit/offset pagination; song search uses FTS5. General search also queries albums, artists, and playlists by name. The current 10,000-track benchmark measures a paired FTS5 search and paginated list query; see the measured local result in [README.md](README.md).

## Migration and integrity notes

Schema version 1 is written at startup, but version upgrades are not yet managed by an ordered migration runner. Track/history relationships are not enforced with SQLite foreign keys. History records last played time, play count, skip count, and completion rate; playback handlers update these fields as events occur.
