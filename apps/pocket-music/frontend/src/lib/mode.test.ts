import { test } from 'node:test';
import assert from 'node:assert/strict';
import type { PlaybackMode } from '../app/types';
import { MODES, isSupportedMode, modeLabelKey, nextMode } from './mode';

test('modes cycle normal → repeat all → repeat one → shuffle → normal', () => {
  assert.deepEqual(MODES, ['normal', 'repeat_all', 'repeat_one', 'shuffle']);
  assert.equal(nextMode('normal'), 'repeat_all');
  assert.equal(nextMode('repeat_all'), 'repeat_one');
  assert.equal(nextMode('repeat_one'), 'shuffle');
  assert.equal(nextMode('shuffle'), 'normal');
});

test('cycling four times returns to the starting mode', () => {
  let mode: PlaybackMode = 'normal';
  for (let i = 0; i < 4; i += 1) {
    mode = nextMode(mode);
  }
  assert.equal(mode, 'normal');
});

test('mode labels map to translation keys', () => {
  assert.equal(modeLabelKey('normal'), 'modeNormal');
  assert.equal(modeLabelKey('repeat_one'), 'modeRepeatOne');
  assert.equal(modeLabelKey('shuffle'), 'modeShuffle');
  assert.equal(isSupportedMode('repeat_all'), true);
  assert.equal(isSupportedMode('bogus'), false);
});
