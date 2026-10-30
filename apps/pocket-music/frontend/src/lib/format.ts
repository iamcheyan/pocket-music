// Pure formatting helpers. Kept free of React so they can be unit tested with
// node:test against the same source the UI imports.

export function formatDuration(seconds?: number | null): string {
  if (seconds === undefined || seconds === null || !Number.isFinite(seconds) || seconds < 0) {
    return '—:—';
  }
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
  }
  return `${minutes}:${String(secs).padStart(2, '0')}`;
}

export function formatPlayTime(seconds: number, labels: { hour: string; minute: string }): string {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return `0 ${labels.minute}`;
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.round((seconds % 3600) / 60);
  if (hours > 0) {
    return `${hours} ${labels.hour} ${minutes} ${labels.minute}`;
  }
  return `${Math.max(1, minutes)} ${labels.minute}`;
}

export function formatPercent(fraction: number): string {
  if (!Number.isFinite(fraction)) {
    return '0%';
  }
  return `${Math.round(Math.max(0, Math.min(1, fraction)) * 100)}%`;
}

export function initials(title: string): string {
  const words = title.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return '♪';
  }
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }
  return (words[0][0] + words[1][0]).toUpperCase();
}
