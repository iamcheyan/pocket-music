// Queue and playlist reordering helpers. The backend applies the same
// semantics; these keep the optimistic UI in step with it.

export function move<T>(list: readonly T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) {
    return [...list];
  }
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

export function withoutIndex<T>(list: readonly T[], index: number): T[] {
  if (index < 0 || index >= list.length) {
    return [...list];
  }
  return [...list.slice(0, index), ...list.slice(index + 1)];
}

export function insertFirst<T>(list: readonly T[], item: T): T[] {
  return [item, ...list];
}

export function clampVolume(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round(value)));
}

export function clampPosition(value: number, duration: number): number {
  if (!Number.isFinite(value) || value < 0) {
    return 0;
  }
  if (!Number.isFinite(duration) || duration <= 0) {
    return value;
  }
  return Math.min(value, duration);
}

// groupedQueue splits the queue into the track that is playing and the rest, so
// the queue page can mark the current row without duplicating it.
export function splitCurrent<T extends { id: string }>(queue: readonly T[], currentId?: string) {
  if (!currentId) {
    return { current: undefined, rest: [...queue] };
  }
  const index = queue.findIndex((item) => item.id === currentId);
  if (index < 0) {
    return { current: undefined, rest: [...queue] };
  }
  return { current: queue[index], rest: [...queue.slice(0, index), ...queue.slice(index + 1)] };
}
