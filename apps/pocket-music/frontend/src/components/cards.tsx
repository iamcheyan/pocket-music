import { Heart, Play } from 'lucide-react';
import type { AlbumSummary, ArtistSummary, Track } from '../app/types';
import type { Pocket } from '../app/usePocket';
import { formatDuration } from '../lib/format';
import { Cover } from './ui';

export function TrackCard({ track, pocket, context }: { track: Track; pocket: Pocket; context?: Track[] }) {
  const { tr, actions } = pocket;
  return (
    <button type="button" className="card" onClick={() => actions.playTrack(track, context ?? [track])} aria-label={`${tr('play')} ${track.title}`}>
      <div className="card-art">
        <Cover albumId={track.album_id} title={track.album || track.title} artist={track.artist} />
        <span className="card-play">
          <Play size={14} fill="currentColor" />
        </span>
      </div>
      <b>{track.title}</b>
      <span>{track.artist || tr('unknownArtist')}</span>
    </button>
  );
}

export function AlbumCard({
  album,
  pocket,
  onOpen,
}: {
  album: AlbumSummary;
  pocket: Pocket;
  onOpen?: (album: AlbumSummary) => void;
}) {
  const { tr, actions } = pocket;
  const favourite = actions.isFavorite('album', album.id);
  return (
    <div className="card card-album">
      <button type="button" className="card-open" onClick={() => onOpen?.(album)} aria-label={`${tr('openAlbum')}: ${album.title}`}>
        <div className="card-art">
          <Cover albumId={album.id} title={album.title} artist={album.artist} />
          <span className="card-play">
            <Play size={14} fill="currentColor" />
          </span>
        </div>
        <b>{album.title}</b>
        <span>{album.artist || tr('unknownArtist')}</span>
      </button>
      <button
        type="button"
        className={`icon-button card-heart ${favourite ? 'is-favourite' : ''}`}
        aria-label={favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
        aria-pressed={favourite}
        onClick={() => void actions.toggleFavorite('album', album.id)}
      >
        <Heart size={14} fill={favourite ? 'currentColor' : 'none'} />
      </button>
    </div>
  );
}

export function ArtistCard({ artist, pocket, onOpen }: { artist: ArtistSummary; pocket: Pocket; onOpen: (artist: ArtistSummary) => void }) {
  const { tr, actions } = pocket;
  const favourite = actions.isFavorite('artist', artist.id);
  return (
    <div className="card card-artist">
      <button type="button" className="card-open" onClick={() => onOpen(artist)} aria-label={`${tr('openArtist')}: ${artist.name}`}>
        <div className="card-art artist-art">
          <span className="artist-initials">{artist.name.slice(0, 2).toUpperCase()}</span>
          <span className="cover-rings" aria-hidden="true" />
        </div>
        <b>{artist.name}</b>
        <span>
          {artist.track_count} {tr('trackCount')}
        </span>
      </button>
      <button
        type="button"
        className={`icon-button card-heart ${favourite ? 'is-favourite' : ''}`}
        aria-label={favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
        aria-pressed={favourite}
        onClick={() => void actions.toggleFavorite('artist', artist.id)}
      >
        <Heart size={14} fill={favourite ? 'currentColor' : 'none'} />
      </button>
    </div>
  );
}

export function PlaylistCard({ playlist, pocket, onOpen }: { playlist: { id: number | string; name: string; track_count?: number }; pocket: Pocket; onOpen: () => void }) {
  const { tr } = pocket;
  return (
    <button type="button" className="card" onClick={onOpen} aria-label={playlist.name}>
      <div className="card-art playlist-art">
        <span className="cover-rings" aria-hidden="true" />
        <span className="playlist-glyph">{String(playlist.name).slice(0, 1).toUpperCase()}</span>
      </div>
      <b>{playlist.name}</b>
      <span>
        {playlist.track_count ?? 0} {tr('trackCount')}
      </span>
    </button>
  );
}

export function TrackRowCompact({ track, pocket, index }: { track: Track; pocket: Pocket; index?: number }) {
  return (
    <button type="button" className="compact-row" onClick={() => pocket.actions.playTrack(track, [track])}>
      {index !== undefined && <span className="compact-index">{String(index + 1).padStart(2, '0')}</span>}
      <span className="compact-title">{track.title}</span>
      <span className="compact-time">{formatDuration(track.duration)}</span>
    </button>
  );
}
