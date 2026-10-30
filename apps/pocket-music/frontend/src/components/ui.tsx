import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from 'react';
import { Check, ChevronDown } from 'lucide-react';
import { clampPosition } from '../lib/queue';

/* ------------------------------------------------------------------ cover */

export function Cover({
  albumId,
  title,
  artist,
  className = '',
  rounded = true,
}: {
  albumId?: number | null;
  title?: string;
  artist?: string;
  className?: string;
  rounded?: boolean;
}) {
  const [failed, setFailed] = useState(false);
  const label = title || artist || 'Pocket Music';
  useEffect(() => {
    setFailed(false);
  }, [albumId]);
  const style: React.CSSProperties = { borderRadius: rounded ? undefined : 0 };
  if (!albumId || failed) {
    return (
      <div className={`cover cover-fallback ${className}`} style={style} aria-label={label} role="img">
        <span className="cover-initials">{label.slice(0, 2).toUpperCase()}</span>
        <span className="cover-rings" aria-hidden="true" />
      </div>
    );
  }
  return (
    <img
      className={`cover ${className}`}
      style={style}
      src={`/api/cover/${albumId}`}
      alt={label}
      loading="lazy"
      draggable={false}
      onError={() => setFailed(true)}
    />
  );
}

/* ----------------------------------------------------------------- slider */

export function Slider({
  value,
  max,
  onChange,
  label,
  step,
  valueText,
  className = '',
}: {
  value: number;
  max: number;
  onChange: (value: number) => void;
  label: string;
  step?: number;
  valueText?: string;
  className?: string;
}) {
  const track = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);
  const safeMax = max > 0 ? max : 1;
  const ratio = Math.max(0, Math.min(1, value / safeMax));

  const commit = useCallback(
    (clientX: number) => {
      const element = track.current;
      if (!element) return;
      const rect = element.getBoundingClientRect();
      if (rect.width <= 0) return;
      const next = clampPosition(((clientX - rect.left) / rect.width) * safeMax, safeMax);
      onChange(next);
    },
    [onChange, safeMax],
  );

  const onKeyDown = (event: React.KeyboardEvent) => {
    const increment = step ?? safeMax / 20;
    if (event.key === 'ArrowRight' || event.key === 'ArrowUp') {
      event.preventDefault();
      onChange(clampPosition(value + increment, safeMax));
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowDown') {
      event.preventDefault();
      onChange(clampPosition(value - increment, safeMax));
    } else if (event.key === 'Home') {
      event.preventDefault();
      onChange(0);
    } else if (event.key === 'End') {
      event.preventDefault();
      onChange(safeMax);
    }
  };

  return (
    <div
      ref={track}
      className={`slider ${dragging ? 'slider-active' : ''} ${className}`}
      role="slider"
      tabIndex={0}
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={Math.round(safeMax)}
      aria-valuenow={Math.round(value)}
      aria-valuetext={valueText}
      onKeyDown={onKeyDown}
      onPointerDown={(event) => {
        event.currentTarget.setPointerCapture(event.pointerId);
        setDragging(true);
        commit(event.clientX);
      }}
      onPointerMove={(event) => {
        if (dragging) commit(event.clientX);
      }}
      onPointerUp={(event) => {
        if (dragging) commit(event.clientX);
        setDragging(false);
        if (event.currentTarget.hasPointerCapture(event.pointerId)) {
          event.currentTarget.releasePointerCapture(event.pointerId);
        }
      }}
      onPointerCancel={() => setDragging(false)}
    >
      <div className="slider-track">
        <div className="slider-fill" style={{ width: `${ratio * 100}%` }} />
        <span className="slider-knob" style={{ left: `${ratio * 100}%` }} />
      </div>
    </div>
  );
}

/* ----------------------------------------------------------------- select */

export type Option = { value: string; label: string; hint?: string };

export function Select({
  value,
  options,
  onChange,
  label,
  className = '',
}: {
  value: string;
  options: Option[];
  onChange: (value: string) => void;
  label: string;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const wrapper = useRef<HTMLDivElement>(null);
  const listId = useId();
  const current = options.find((option) => option.value === value) ?? options[0];

  useEffect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => {
      if (wrapper.current && !wrapper.current.contains(event.target as Node)) setOpen(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    document.addEventListener('pointerdown', onPointer);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('pointerdown', onPointer);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  return (
    <div className={`select ${className}`} ref={wrapper}>
      <button
        type="button"
        className="select-button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        aria-label={label}
        onClick={() => setOpen((current) => !current)}
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            setOpen(true);
          }
        }}
      >
        <span>{current?.label ?? '—'}</span>
        <ChevronDown size={14} aria-hidden="true" />
      </button>
      {open && (
        <div className="select-popover" role="listbox" id={listId} aria-label={label}>
          {options.map((option) => (
            <button
              key={option.value}
              type="button"
              role="option"
              aria-selected={option.value === value}
              className={`select-option ${option.value === value ? 'selected' : ''}`}
              onClick={() => {
                onChange(option.value);
                setOpen(false);
              }}
            >
              <span>
                {option.label}
                {option.hint && <small>{option.hint}</small>}
              </span>
              {option.value === value && <Check size={14} aria-hidden="true" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/* -------------------------------------------------------------- segmented */

export function Segmented({
  value,
  options,
  onChange,
  label,
  className = '',
}: {
  value: string;
  options: Option[];
  onChange: (value: string) => void;
  label: string;
  className?: string;
}) {
  return (
    <div className={`segmented ${className}`} role="group" aria-label={label}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          className={`segmented-option ${option.value === value ? 'active' : ''}`}
          aria-pressed={option.value === value}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

export function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (value: boolean) => void; label: string }) {
  return (
    <button type="button" role="switch" aria-checked={checked} aria-label={label} className={`toggle ${checked ? 'on' : ''}`} onClick={() => onChange(!checked)}>
      <span className="toggle-knob" />
    </button>
  );
}

/* ------------------------------------------------------------- structure */

export function PageHeader({ eyebrow, title, actions }: { eyebrow?: string; title: string; actions?: ReactNode }) {
  return (
    <div className="page-heading">
      <div>
        {eyebrow && <span className="eyebrow">{eyebrow}</span>}
        <h1>{title}</h1>
      </div>
      {actions && <div className="page-actions">{actions}</div>}
    </div>
  );
}

export function Section({ title, action, children, className = '' }: { title: string; action?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`section ${className}`}>
      <div className="section-heading">
        <h2>{title}</h2>
        {action}
      </div>
      {children}
    </section>
  );
}

export function EmptyState({ icon, title, hint }: { icon?: ReactNode; title: string; hint?: string }) {
  return (
    <div className="empty-state">
      {icon}
      <strong>{title}</strong>
      {hint && <span>{hint}</span>}
    </div>
  );
}
