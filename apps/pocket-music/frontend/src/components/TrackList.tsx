import { useEffect, useRef, useState } from 'react';
import {
  CornerDownRight, Heart, ListPlus, MoreHorizontal, Music2, Play, Plus, Trash2, GripVertical,
} from 'lucide-react';
import type { Track } from '../app/types';
import type { Pocket } from '../app/usePocket';
import { formatDuration } from '../lib/format';
import { Cover } from './ui';

type Variant = 'default' | 'queue' | 'playlist';

export function TrackList({
  tracks,
  pocket,
  context,
  variant = 'default',
  onRemove,
  onReorder,
  emptyTitle,
  emptyHint,
  numbering = 'index',
}: {
  tracks: Track[];
  pocket: Pocket;
  context?: Track[];
  variant?: Variant;
  onRemove?: (index: number) => void;
  onReorder?: (from: number, to: number) => void;
  emptyTitle?: string;
  emptyHint?: string;
  numbering?: 'index' | 'none' | 'track';
}) {
  const { tr } = pocket;
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [overIndex, setOverIndex] = useState<number | null>(null);

  if (tracks.length === 0) {
    return (
      <div className="empty-state">
        <Music2 />
        <strong>{emptyTitle ?? tr('empty')}</strong>
        {emptyHint && <span>{emptyHint}</span>}
      </div>
    );
  }

  const reorderable = Boolean(onReorder);

  return (
    <div className={`track-list ${reorderable ? 'track-list-sortable' : ''}`} role="list">
      {tracks.map((track, index) => (
        <TrackRow
          key={`${track.id}-${index}`}
          track={track}
          index={index}
          pocket={pocket}
          context={context ?? tracks}
          variant={variant}
          onRemove={onRemove ? () => onRemove(index) : undefined}
          numbering={numbering}
          drag={{
            enabled: reorderable,
            active: dragIndex === index,
            over: overIndex === index,
            onStart: () => setDragIndex(index),
            onOver: () => setOverIndex(index),
            onDrop: () => {
              if (dragIndex !== null && dragIndex !== index) onReorder?.(dragIndex, index);
              setDragIndex(null);
              setOverIndex(null);
            },
            onEnd: () => {
              setDragIndex(null);
              setOverIndex(null);
            },
          }}
        />
      ))}
    </div>
  );
}

function TrackRow({
  track,
  index,
  pocket,
  context,
  variant,
  onRemove,
  numbering,
  drag,
}: {
  track: Track;
  index: number;
  pocket: Pocket;
  context: Track[];
  variant: Variant;
  onRemove?: () => void;
  numbering: 'index' | 'none' | 'track';
  drag: {
    enabled: boolean;
    active: boolean;
    over: boolean;
    onStart: () => void;
    onOver: () => void;
    onDrop: () => void;
    onEnd: () => void;
  };
}) {
  const { tr, data, actions } = pocket;
  const playing = data.player.track?.id === track.id;
  const favourite = actions.isFavorite('track', track.id);
  const label = numbering === 'track' && track.track_number ? String(track.track_number) : String(index + 1).padStart(2, '0');

  return (
    <div
      className={`track-row ${playing ? 'track-row-playing' : ''} ${drag.active ? 'dragging' : ''} ${drag.over ? 'drop-target' : ''}`}
      role="listitem"
      draggable={drag.enabled}
      onDragStart={(event) => {
        event.dataTransfer.effectAllowed = 'move';
        event.dataTransfer.setData('text/plain', String(index));
        drag.onStart();
      }}
      onDragOver={(event) => {
        if (!drag.enabled) return;
        event.preventDefault();
        drag.onOver();
      }}
      onDrop={(event) => {
        if (!drag.enabled) return;
        event.preventDefault();
        drag.onDrop();
      }}
      onDragEnd={drag.onEnd}
      onDoubleClick={() => actions.playTrack(track, context)}
    >
      {numbering !== 'none' && (
        <span className="row-index" aria-hidden="true">
          {drag.enabled ? <GripVertical size={13} /> : label}
        </span>
      )}
      <button className="row-cover" aria-label={`${tr('play')} ${track.title}`} onClick={() => actions.playTrack(track, context)} type="button">
        <Cover albumId={track.album_id} title={track.album || track.title} className="cover-thumb" rounded />
        <span className="row-play">
          <Play size={12} fill="currentColor" />
        </span>
      </button>
      <div className="row-meta">
        <b>{track.title || tr('noTrack')}</b>
        <span>
          {track.artist || tr('unknownArtist')}
          <i className="row-dot">·</i>
          {track.album || tr('unknownAlbum')}
        </span>
      </div>
      <span className="row-album">{(track.genre || '').slice(0, 18)}</span>
      <span className="row-time">{formatDuration(track.duration)}</span>
      <div className="row-actions">
        <button
          type="button"
          className={`icon-button row-heart ${favourite ? 'is-favourite' : ''}`}
          aria-label={favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
          aria-pressed={favourite}
          onClick={() => void actions.toggleFavorite('track', track.id)}
        >
          <Heart size={15} fill={favourite ? 'currentColor' : 'none'} />
        </button>
        {variant === 'default' && (
          <button type="button" className="icon-button row-add" aria-label={tr('addQueue')} onClick={() => actions.queueAdd(track)}>
            <Plus size={15} />
          </button>
        )}
        {(variant === 'queue' || variant === 'playlist') && onRemove && (
          <button type="button" className="icon-button row-add" aria-label={tr('remove')} onClick={onRemove}>
            <Trash2 size={15} />
          </button>
        )}
        <RowMenu track={track} pocket={pocket} onRemove={onRemove} variant={variant} />
      </div>
    </div>
  );
}

function RowMenu({ track, pocket, onRemove, variant }: { track: Track; pocket: Pocket; onRemove?: () => void; variant: Variant }) {
  const { tr, data, actions } = pocket;
  const [open, setOpen] = useState(false);
  const [showPlaylists, setShowPlaylists] = useState(false);
  const wrapper = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => {
      if (wrapper.current && !wrapper.current.contains(event.target as Node)) {
        setOpen(false);
        setShowPlaylists(false);
      }
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
        setShowPlaylists(false);
      }
    };
    document.addEventListener('pointerdown', onPointer);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('pointerdown', onPointer);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  const favourite = actions.isFavorite('track', track.id);

  return (
    <div className="row-menu" ref={wrapper}>
      <button
        type="button"
        className="icon-button"
        aria-label={tr('more')}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <MoreHorizontal size={16} />
      </button>
      {open && (
        <div className="popover" role="menu">
          <button type="button" role="menuitem" onClick={() => { actions.playTrack(track, [track]); setOpen(false); }}>
            <Play size={14} /> {tr('play')}
          </button>
          <button type="button" role="menuitem" onClick={() => { actions.queuePlayNext(track); setOpen(false); }}>
            <CornerDownRight size={14} /> {tr('playNext')}
          </button>
          <button type="button" role="menuitem" onClick={() => { actions.queueAdd(track); setOpen(false); }}>
            <ListPlus size={14} /> {tr('addQueue')}
          </button>
          <button type="button" role="menuitem" onClick={() => { void actions.toggleFavorite('track', track.id); setOpen(false); }}>
            <Heart size={14} fill={favourite ? 'currentColor' : 'none'} /> {favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
          </button>
          {data.playlists.length > 0 && (
            <button type="button" role="menuitem" onClick={() => setShowPlaylists((current) => !current)} aria-expanded={showPlaylists}>
              <Plus size={14} /> {tr('addToPlaylist')}
            </button>
          )}
          {showPlaylists &&
            data.playlists.map((playlist) => (
              <button
                key={playlist.id}
                type="button"
                role="menuitem"
                className="popover-sub"
                onClick={() => {
                  actions.addToPlaylist(playlist.id, [track.id]);
                  setOpen(false);
                  setShowPlaylists(false);
                }}
              >
                {playlist.name}
              </button>
            ))}
          {variant !== 'default' && onRemove && (
            <button type="button" role="menuitem" onClick={() => { onRemove(); setOpen(false); }}>
              <Trash2 size={14} /> {tr('remove')}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
