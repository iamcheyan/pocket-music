import { useEffect, useRef } from 'react';
import { Disc3, Music2, Search as SearchIcon, Users } from 'lucide-react';
import type { Pocket } from '../app/usePocket';
import { AlbumCard, ArtistCard, PlaylistCard } from '../components/cards';
import { TrackList } from '../components/TrackList';
import { EmptyState, PageHeader } from '../components/ui';

export function SearchPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions, nav } = pocket;
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    input.current?.focus();
  }, []);

  const results = data.searchResults;
  const hasQuery = nav.q.trim().length > 0;

  return (
    <>
      <PageHeader eyebrow={tr('searchAll')} title={hasQuery ? `${tr('resultsFor')} “${nav.q}”` : tr('search')} />
      <div className="search-box">
        <SearchIcon size={16} />
        <input
          ref={input}
          id="global-search"
          value={nav.q}
          onChange={(event) => actions.setQuery(event.target.value)}
          placeholder={tr('searchHint')}
          aria-label={tr('search')}
          autoComplete="off"
        />
        {hasQuery && (
          <button type="button" className="text-button" onClick={() => actions.setQuery('')}>
            {tr('cancel')}
          </button>
        )}
      </div>

      {!hasQuery && <EmptyState icon={<SearchIcon />} title={tr('search')} hint={tr('searchHint')} />}

      {hasQuery && results.total === 0 && <EmptyState icon={<Music2 />} title={tr('noResults')} hint={tr('searchHint')} />}

      {hasQuery && results.songs.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>
              {tr('songsFound')} <span className="count">{results.songs.length}</span>
            </h2>
          </div>
          <TrackList tracks={results.songs} pocket={pocket} />
        </section>
      )}

      {hasQuery && results.albums.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>
              {tr('albumsFound')} <span className="count">{results.albums.length}</span>
            </h2>
          </div>
          <div className="card-grid">
            {results.albums.map((album) => (
              <AlbumCard key={album.id} album={album} pocket={pocket} onOpen={(value) => actions.navigate('album', { id: String(value.id) })} />
            ))}
          </div>
        </section>
      )}

      {hasQuery && results.artists.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>
              {tr('artistsFound')} <span className="count">{results.artists.length}</span>
            </h2>
          </div>
          <div className="card-grid">
            {results.artists.map((artist) => (
              <ArtistCard key={artist.id} artist={{ ...artist, track_count: artist.track_count ?? 0 }} pocket={pocket} onOpen={(value) => actions.navigate('artist', { id: String(value.id) })} />
            ))}
          </div>
        </section>
      )}

      {hasQuery && results.playlists.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>
              {tr('playlistsFound')} <span className="count">{results.playlists.length}</span>
            </h2>
          </div>
          <div className="card-grid">
            {results.playlists.map((playlist) => (
              <PlaylistCard
                key={playlist.id}
                playlist={playlist}
                pocket={pocket}
                onOpen={() => actions.navigate('playlist', { id: String(playlist.id) })}
              />
            ))}
          </div>
        </section>
      )}

      {!hasQuery && data.songs.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>{tr('popular')}</h2>
          </div>
          <div className="quick-grid">
            {data.home.most_played.slice(0, 6).map((track) => (
              <button key={track.id} type="button" className="quick-chip" onClick={() => actions.playTrack(track, data.home.most_played)}>
                <Disc3 size={14} /> {track.title}
              </button>
            ))}
          </div>
        </section>
      )}
    </>
  );
}

export function StatsPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions } = pocket;
  const stats = data.stats;
  if (!stats) {
    return <EmptyState icon={<Users />} title={tr('loading')} />;
  }
  const maxPlays = Math.max(1, ...stats.top_artists.map((artist) => artist.play_count));
  return (
    <>
      <PageHeader eyebrow={tr('libraryStats')} title={tr('stats')} />
      <div className="stat-grid">
        <div className="stat">
          <span>{tr('playTime')}</span>
          <b>
            {Math.floor(stats.play_time / 3600)} {tr('hour')} {Math.round((stats.play_time % 3600) / 60)} {tr('minute')}
          </b>
        </div>
        <div className="stat">
          <span>{tr('playCount')}</span>
          <b>{stats.play_count}</b>
        </div>
        <div className="stat">
          <span>{tr('skipCount')}</span>
          <b>{stats.skip_count}</b>
        </div>
        <div className="stat">
          <span>{tr('completion')}</span>
          <b>{Math.round(stats.completion * 100)}%</b>
        </div>
        <div className="stat">
          <span>{tr('songs')}</span>
          <b>{stats.library.tracks}</b>
        </div>
        <div className="stat">
          <span>{tr('albums')}</span>
          <b>{stats.library.albums}</b>
        </div>
        <div className="stat">
          <span>{tr('artists')}</span>
          <b>{stats.library.artists}</b>
        </div>
        <div className="stat">
          <span>{tr('playlists')}</span>
          <b>{stats.library.playlists}</b>
        </div>
      </div>

      {stats.top_artists.length === 0 && stats.most_played.length === 0 ? (
        <EmptyState icon={<Music2 />} title={tr('nothingYet')} />
      ) : (
        <>
          <section className="section">
            <div className="section-heading">
              <h2>{tr('topArtists')}</h2>
            </div>
            <div className="rank-list">
              {stats.top_artists.map((artist, index) => (
                <button key={artist.id} type="button" className="rank-row" onClick={() => actions.navigate('artist', { id: String(artist.id) })}>
                  <span className="rank-index">{String(index + 1).padStart(2, '0')}</span>
                  <span className="rank-name">{artist.name}</span>
                  <span className="rank-bar">
                    <i style={{ width: `${(artist.play_count / maxPlays) * 100}%` }} />
                  </span>
                  <span className="rank-value">{artist.play_count}</span>
                </button>
              ))}
            </div>
          </section>

          <section className="section">
            <div className="section-heading">
              <h2>{tr('mostPlayed')}</h2>
            </div>
            <TrackList tracks={stats.most_played} pocket={pocket} />
          </section>

          <section className="section">
            <div className="section-heading">
              <h2>{tr('topAlbums')}</h2>
            </div>
            <div className="card-grid">
              {stats.top_albums.map((album) => (
                <AlbumCard key={album.id} album={album} pocket={pocket} onOpen={(value) => actions.navigate('album', { id: String(value.id) })} />
              ))}
            </div>
          </section>

          {stats.most_skipped.length > 0 && (
            <section className="section">
              <div className="section-heading">
                <h2>{tr('mostSkipped')}</h2>
              </div>
              <TrackList tracks={stats.most_skipped} pocket={pocket} />
            </section>
          )}
        </>
      )}
    </>
  );
}
