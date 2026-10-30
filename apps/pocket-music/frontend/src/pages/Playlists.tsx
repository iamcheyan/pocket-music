import { useMemo, useRef, useState } from 'react';
import { Download, ListMusic, Play, Plus, Search, Trash2, Upload } from 'lucide-react';
import type { Pocket } from '../app/usePocket';
import type { Track } from '../app/types';
import { TrackList } from '../components/TrackList';
import { PlaylistCard } from '../components/cards';
import { EmptyState, PageHeader } from '../components/ui';
import { formatDuration } from '../lib/format';

export function PlaylistsPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions } = pocket;
  const [name, setName] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  return (
    <>
      <PageHeader eyebrow={tr('library')} title={tr('playlists')} />
      <div className="playlist-tools">
        <form
          className="inline-form"
          onSubmit={(event) => {
            event.preventDefault();
            if (!name.trim()) return;
            actions.createPlaylist(name.trim());
            setName('');
          }}
        >
          <input value={name} onChange={(event) => setName(event.target.value)} placeholder={tr('playlistName')} aria-label={tr('playlistName')} />
          <button type="submit" className="primary-button" disabled={!name.trim()}>
            <Plus size={16} /> {tr('create')}
          </button>
        </form>
        <form
          className="inline-form"
          onSubmit={(event) => {
            event.preventDefault();
            const input = fileRef.current;
            const file = input?.files?.[0];
            if (!file) return;
            actions.importPlaylist(file, file.name.replace(/\.(m3u8?)$/i, ''));
            input.value = '';
          }}
        >
          <input ref={fileRef} type="file" accept=".m3u,.m3u8" aria-label={tr('importM3u')} className="file-input" />
          <button type="submit" className="secondary-button">
            <Upload size={15} /> {tr('importM3u')}
          </button>
        </form>
      </div>

      {data.playlists.length === 0 ? (
        <EmptyState icon={<ListMusic />} title={tr('empty')} hint={tr('emptyHint')} />
      ) : (
        <div className="card-grid">
          {data.playlists.map((playlist) => (
            <PlaylistCard
              key={playlist.id}
              playlist={playlist}
              pocket={pocket}
              onOpen={() => actions.navigate('playlist', { id: String(playlist.id) })}
            />
          ))}
        </div>
      )}
    </>
  );
}

export function PlaylistPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions, nav } = pocket;
  const playlist = data.playlist;
  const [name, setName] = useState('');
  const [query, setQuery] = useState('');
  const [pending, setPending] = useState<Track | null>(null);

  const results = useMemo(() => {
    const text = query.trim().toLowerCase();
    if (!text) return [];
    return data.songs.filter((track) => track.title.toLowerCase().includes(text) || track.artist.toLowerCase().includes(text)).slice(0, 8);
  }, [query, data.songs]);

  if (!playlist || String(playlist.id) !== nav.id) {
    return <EmptyState icon={<ListMusic />} title={tr('loading')} />;
  }

  const tracks = playlist.tracks ?? [];
  const duration = tracks.reduce((sum, track) => sum + (track.duration || 0), 0);

  return (
    <>
      <div className="playlist-header">
        <div className="playlist-cover" aria-hidden="true">
          <span>{playlist.name.slice(0, 1).toUpperCase()}</span>
        </div>
        <div className="detail-copy">
          <span className="eyebrow">{tr('playlist')}</span>
          <input
            className="title-input"
            value={name || playlist.name}
            aria-label={tr('rename')}
            onChange={(event) => setName(event.target.value)}
            onBlur={() => {
              if (name && name !== playlist.name) actions.renamePlaylist(playlist.id, name);
            }}
          />
          <p>
            {tracks.length} {tr('trackCount')} · {formatDuration(duration)}
          </p>
          <div className="hero-actions">
            <button type="button" className="primary-button" onClick={() => actions.playAll(tracks)} disabled={tracks.length === 0}>
              <Play size={16} fill="currentColor" /> {tr('playAll')}
            </button>
            <a className="secondary-button" href={`/api/playlists/${playlist.id}/export?format=m3u`} download>
              <Download size={15} /> {tr('exportM3u')}
            </a>
            <a className="secondary-button" href={`/api/playlists/${playlist.id}/export?format=m3u8`} download>
              <Download size={15} /> {tr('exportM3u8')}
            </a>
            <button type="button" className="secondary-button danger" onClick={() => actions.deletePlaylist(playlist.id)}>
              <Trash2 size={15} /> {tr('delete')}
            </button>
          </div>
        </div>
      </div>

      <div className="playlist-add">
        <label className="filter-field wide">
          <Search size={14} />
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={tr('searchHint')} aria-label={tr('addToPlaylist')} />
        </label>
        <div className="playlist-add-results">
          {results.map((track) => (
            <button
              key={track.id}
              type="button"
              className={`playlist-add-item ${pending?.id === track.id ? 'selected' : ''}`}
              onClick={() => setPending(track)}
            >
              {track.title} <small>{track.artist}</small>
            </button>
          ))}
        </div>
        <button
          type="button"
          className="primary-button"
          disabled={!pending}
          onClick={() => {
            if (!pending) return;
            actions.playlistMutate(playlist.id, 'add', { track_id: pending.id });
            setPending(null);
            setQuery('');
          }}
        >
          <Plus size={16} /> {tr('add')}
        </button>
      </div>

      <TrackList
        tracks={tracks}
        pocket={pocket}
        variant="playlist"
        numbering="track"
        onRemove={(index) => actions.playlistMutate(playlist.id, 'remove', { index })}
        onReorder={(from, to) => actions.playlistMutate(playlist.id, 'reorder', { index: from, to })}
        emptyTitle={tr('playlistEmpty')}
        emptyHint={tr('emptyHint')}
      />
    </>
  );
}
