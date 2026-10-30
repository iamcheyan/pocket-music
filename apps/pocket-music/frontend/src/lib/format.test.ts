import { test } from 'node:test';
import assert from 'node:assert/strict';
import { formatDuration, formatPercent, initials, formatPlayTime } from './format';

test('formatDuration renders minutes and seconds', () => {
  assert.equal(formatDuration(0), '0:00');
  assert.equal(formatDuration(9.8), '0:09');
  assert.equal(formatDuration(61), '1:01');
  assert.equal(formatDuration(3599), '59:59');
});

test('formatDuration renders hours for long tracks', () => {
  assert.equal(formatDuration(3600), '1:00:00');
  assert.equal(formatDuration(3725), '1:02:05');
});

test('formatDuration handles missing and invalid input', () => {
  assert.equal(formatDuration(undefined), '—:—');
  assert.equal(formatDuration(null), '—:—');
  assert.equal(formatDuration(-5), '—:—');
  assert.equal(formatDuration(Number.NaN), '—:—');
});

test('formatPercent clamps to a percentage', () => {
  assert.equal(formatPercent(0.5), '50%');
  assert.equal(formatPercent(1.4), '100%');
  assert.equal(formatPercent(-1), '0%');
});

test('initials picks two characters', () => {
  assert.equal(initials('Night Cartography'), 'NC');
  assert.equal(initials('Drift'), 'DR');
  assert.equal(initials('   '), '♪');
});

test('formatPlayTime switches to hours', () => {
  assert.equal(formatPlayTime(0, { hour: 'h', minute: 'min' }), '0 min');
  assert.equal(formatPlayTime(120, { hour: 'h', minute: 'min' }), '2 min');
  assert.equal(formatPlayTime(3900, { hour: 'h', minute: 'min' }), '1 h 5 min');
});
