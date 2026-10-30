# Database

`musicd` stores library and user state in SQLite through the pure Go `modernc.org/sqlite` driver. The service should initialize its schema on startup and track a schema version so later changes can be applied as migrations. Exact column definitions and migration version numbers must be updated from the implemented schema; this document records the required entities, not a verified schema dump.

## Required entities

| Table | Purpose |
| --- | --- |
| `tracks` | Audio metadata, file identity/path, duration, and relationships to album and artist. |
| `albums` | Album metadata and cover association. |
| `artists` | Artist identities and metadata. |
| `playlists` | User-created playlist metadata. |
| `playlist_items` | Ordered track membership in playlists. |
| `favorites` | Favorite tracks, albums, or artists. |
| `history` | Playback events and derived play statistics. |
| `settings` | Persistent user and service preferences. |

History must retain last played time, play count, skip count, and completion rate. Settings must represent language, theme, music folder, cache size, manual and automatic scan preferences, audio output, and reserved proxy/network configuration.

## Indexing and search

Library queries need indexes for common relationship, ordering, and filtering paths, including track-to-album/artist and playlist item ordering. Search across 10,000 or more tracks must remain responsive; use SQLite FTS5 or an equivalent indexed search strategy and paginate listing queries. The actual index and FTS definitions belong here once confirmed from migrations.

Required performance evidence: record the generated dataset size, search/list query shape, and measured timings in [README.md](README.md). No benchmark has been run or reported in this document.

## Migration policy

- Keep a schema version in the database.
- Create the initial schema automatically when the service starts on a new database.
- Apply ordered, forward migrations when opening an older database.
- Make each migration safe to identify as applied so a restart does not apply it twice.
- Preserve user data during schema upgrades; destructive transformations require an explicit migration strategy.

Update this document alongside schema changes and include migration behavior in backend coverage. The database location is configurable through `MUSICD_DB`; verify the exact default and path interpretation in the service implementation.
