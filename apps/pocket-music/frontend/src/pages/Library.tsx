import { useMemo, useState } from 'react';
import { ChevronRight, FolderOpen, Heart, Music2, Search, Tag, Users } from 'lucide-react';
import type { Pocket } from '../app/usePocket';
import type { Track } from '../app/types';
import { TrackList } from '../components/TrackList';
import { AlbumCard, ArtistCard } from '../components/cards';
import { Cover, EmptyState, PageHeader } from '../components/ui';

const TABS = ['songs', 'albums', 'artists', 'genres', 'folders', 'favorites'] as const;

export function LibraryPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions, nav } = pocket;
  const [expandedFolder, setExpandedFolder] = useState<string | null>(null);
  const [folderTracks, setFolderTracks] = useState<Track[]>([]);

  const openFolder = (name: string) => {
    if (expandedFolder === name) {
      setExpandedFolder(null);
      setFolderTracks([]);
      return;
    }
    setExpandedFolder(name);
    setFolderTracks([]);
    void fetch(`/api/library/songs?folder=${encodeURIComponent(name)}&limit=500`)
      .then((response) => response.json())
      .then((body: { items: Track[] }) => setFolderTracks(body.items ?? []))
      .catch(() => setFolderTracks([]));
  };

  const tabs = useMemo(
    () =>
      TABS.map((key) => ({
        key,
        label: tr(key),
        count:
          key === 'songs'
            ? data.songTotal
            : key === 'albums'
              ? data.albums.length
              : key === 'artists'
                ? data.artists.length
                : key === 'genres'
                  ? data.genres.length
                  : key === 'folders'
                    ? data.folders.length
                    : data.favorites.tracks.length,
      })),
    [tr, data],
  );

  const showFilter = nav.tab === 'songs' || nav.tab === 'albums' || nav.tab === 'artists';

  return (
    <>
      <PageHeader
        eyebrow={tr('library')}
        title={tr(nav.tab)}
        actions={
          <div className="page-actions-row">
            {showFilter && (
              <label className="filter-field">
                <Search size={14} />
                <input
                  value={nav.q}
                  placeholder={tr('searchHint')}
                  onChange={(event) => actions.setQuery(event.target.value)}
                  aria-label={tr('search')}
                />
              </label>
            )}
            <button type="button" className="secondary-button" onClick={() => actions.navigate('search', { q: nav.q })}>
              <Search size={15} /> {tr('search')}
            </button>
          </div>
        }
      />

      <div className="tabs" role="tablist">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={nav.tab === tab.key}
            className={nav.tab === tab.key ? 'active' : ''}
            onClick={() => actions.navigate('library', { tab: tab.key, q: nav.q })}
          >
            {tab.label}
            <span className="tab-count">{tab.count}</span>
          </button>
        ))}
      </div>

      {nav.tab === 'songs' && (
        <>
          <TrackList tracks={data.songs} pocket={pocket} />
          {data.songs.length < data.songTotal && (
            <div className="list-more">
              <button type="button" className="secondary-button" onClick={() => void actions.loadMoreSongs()}>
                {tr('loadMore')} ({data.songs.length}/{data.songTotal})
              </button>
            </div>
          )}
        </>
      )}

      {nav.tab === 'albums' && (
        <div className="card-grid">
          {data.albums.map((album) => (
            <AlbumCard key={album.id} album={album} pocket={pocket} onOpen={(value) => actions.navigate('album', { id: String(value.id) })} />
          ))}
          {data.albums.length === 0 && <EmptyState icon={<Music2 />} title={tr('noResults')} />}
        </div>
      )}

      {nav.tab === 'artists' && (
        <div className="card-grid">
          {data.artists.map((artist) => (
            <ArtistCard key={artist.id} artist={artist} pocket={pocket} onOpen={(value) => actions.navigate('artist', { id: String(value.id) })} />
          ))}
          {data.artists.length === 0 && <EmptyState icon={<Users />} title={tr('noResults')} />}
        </div>
      )}

      {nav.tab === 'genres' && (
        <div className="chip-grid">
          {data.genres.map((genre) => (
            <button
              key={genre.name}
              type="button"
              className="chip"
              onClick={() => actions.navigate('library', { tab: 'songs', q: genre.name })}
            >
              <Tag size={14} />
              <span>{genre.name}</span>
              <small>{genre.track_count}</small>
            </button>
          ))}
          {data.genres.length === 0 && <EmptyState icon={<Tag />} title={tr('noResults')} />}
        </div>
      )}

      {nav.tab === 'folders' && (
        <div className="folder-list">
          {data.folders.map((folder) => (
            <div key={folder.name || 'root'} className="folder-group">
              <button type="button" className="folder-row" aria-expanded={expandedFolder === folder.name} onClick={() => openFolder(folder.name)}>
                <FolderOpen size={16} />
                <span className="folder-name">{folder.name || '.'}</span>
                <small>
                  {folder.track_count} {tr('trackCount')}
                </small>
                <ChevronRight size={15} className={expandedFolder === folder.name ? 'rotated' : ''} />
              </button>
              {expandedFolder === folder.name && (
                <div className="folder-tracks">
                  <TrackList tracks={folderTracks} pocket={pocket} numbering="track" />
                </div>
              )}
            </div>
          ))}
          {data.folders.length === 0 && <EmptyState icon={<FolderOpen />} title={tr('noResults')} />}
        </div>
      )}

      {nav.tab === 'favorites' && (
        <>
          <section className="section">
            <div className="section-heading">
              <h2>{tr('favoriteTracks')}</h2>
            </div>
            {data.favorites.tracks.length === 0 ? (
              <EmptyState icon={<Heart />} title={tr('noResults')} hint={tr('emptyHint')} />
            ) : (
              <TrackList tracks={data.favorites.tracks} pocket={pocket} />
            )}
          </section>
          <section className="section">
            <div className="section-heading">
              <h2>{tr('favoriteAlbums')}</h2>
            </div>
            {data.favorites.albums.length === 0 ? (
              <p className="muted-note">{tr('noResults')}</p>
            ) : (
              <div className="card-grid">
                {data.favorites.albums.map((album) => (
                  <AlbumCard key={`fav-${album.id}`} album={album} pocket={pocket} onOpen={(value) => actions.navigate('album', { id: String(value.id) })} />
                ))}
              </div>
            )}
          </section>
          <section className="section">
            <div className="section-heading">
              <h2>{tr('favoriteArtists')}</h2>
            </div>
            {data.favorites.artists.length === 0 ? (
              <p className="muted-note">{tr('noResults')}</p>
            ) : (
              <div className="card-grid">
                {data.favorites.artists.map((artist) => (
                  <ArtistCard key={`fav-artist-${artist.id}`} artist={artist} pocket={pocket} onOpen={(value) => actions.navigate('artist', { id: String(value.id) })} />
                ))}
              </div>
            )}
          </section>
        </>
      )}
    </>
  );
}

export function AlbumPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions, nav } = pocket;
  const album = data.album;
  if (!album || String(album.id) !== nav.id) {
    return <EmptyState icon={<Music2 />} title={tr('loading')} />;
  }
  const favourite = actions.isFavorite('album', album.id);
  return (
    <>
      <div className="detail-hero">
        <Cover albumId={album.id} title={album.title} artist={album.artist} className="detail-cover" />
        <div className="detail-copy">
          <span className="eyebrow">
            {tr('album')} · {album.year || '—'}
          </span>
          <h1>{album.title}</h1>
          <button type="button" className="detail-artist" onClick={() => actions.navigate('artist', { id: String(album.artist_id) })}>
            {album.artist || tr('unknownArtist')}
          </button>
          <p>
            {album.tracks.length} {tr('trackCount')} · {Math.round((album.duration || 0) / 60)} {tr('minute')}
          </p>
          <div className="hero-actions">
            <button type="button" className="primary-button" onClick={() => actions.playAll(album.tracks)} disabled={album.tracks.length === 0}>
              <Music2 size={16} /> {tr('playAll')}
            </button>
            <button type="button" className="secondary-button" onClick={() => actions.queueAddMany(album.tracks)}>
              {tr('addQueue')}
            </button>
            <button
              type="button"
              className={`icon-button heart-large ${favourite ? 'is-favourite' : ''}`}
              aria-pressed={favourite}
              aria-label={favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
              onClick={() => void actions.toggleFavorite('album', album.id)}
            >
              <Heart size={18} fill={favourite ? 'currentColor' : 'none'} />
            </button>
          </div>
        </div>
      </div>
      <TrackList tracks={album.tracks} pocket={pocket} numbering="track" />
    </>
  );
}

export function ArtistPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions, nav } = pocket;
  const artist = data.artist;
  if (!artist || String(artist.id) !== nav.id) {
    return <EmptyState icon={<Users />} title={tr('loading')} />;
  }
  const favourite = actions.isFavorite('artist', artist.id);
  return (
    <>
      <div className="artist-hero">
        <div className="artist-badge" aria-hidden="true">
          {artist.name.slice(0, 2).toUpperCase()}
        </div>
        <div className="detail-copy">
          <span className="eyebrow">{tr('artist')}</span>
          <h1>{artist.name}</h1>
          <p>
            {artist.albums.length} {tr('albumCount')} · {artist.tracks.length} {tr('trackCount')}
          </p>
          <div className="hero-actions">
            <button type="button" className="primary-button" onClick={() => actions.playAll(artist.tracks)}>
              <Music2 size={16} /> {tr('playAll')}
            </button>
            <button
              type="button"
              className={`icon-button heart-large ${favourite ? 'is-favourite' : ''}`}
              aria-pressed={favourite}
              aria-label={favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
              onClick={() => void actions.toggleFavorite('artist', artist.id)}
            >
              <Heart size={18} fill={favourite ? 'currentColor' : 'none'} />
            </button>
          </div>
        </div>
      </div>
      {artist.albums.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>{tr('albums')}</h2>
          </div>
          <div className="card-grid">
            {artist.albums.map((album) => (
              <AlbumCard key={album.id} album={album} pocket={pocket} onOpen={(value) => actions.navigate('album', { id: String(value.id) })} />
            ))}
          </div>
        </section>
      )}
      <section className="section">
        <div className="section-heading">
          <h2>{tr('songs')}</h2>
        </div>
        <TrackList tracks={artist.tracks} pocket={pocket} numbering="track" />
      </section>
    </>
  );
}
