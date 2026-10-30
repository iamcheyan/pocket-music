import type { PlaybackMode } from '../app/types';

// Cycling order matters: it is what the mode button does on every press.
export const MODES: PlaybackMode[] = ['normal', 'repeat_all', 'repeat_one', 'shuffle'];

export function nextMode(current: PlaybackMode): PlaybackMode {
  const index = MODES.indexOf(current);
  return MODES[(index + 1) % MODES.length];
}

export function modeLabelKey(mode: PlaybackMode): string {
  switch (mode) {
    case 'repeat_all':
      return 'modeRepeatAll';
    case 'repeat_one':
      return 'modeRepeatOne';
    case 'shuffle':
      return 'modeShuffle';
    default:
      return 'modeNormal';
  }
}

export function isSupportedMode(value: string): value is PlaybackMode {
  return (MODES as string[]).includes(value);
}
