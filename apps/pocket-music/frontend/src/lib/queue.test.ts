import { test } from 'node:test';
import assert from 'node:assert/strict';
import { clampPosition, clampVolume, insertFirst, move, splitCurrent, withoutIndex } from './queue';

const list = ['a', 'b', 'c', 'd'];

test('move reorders without mutating the input', () => {
  const moved = move(list, 0, 2);
  assert.deepEqual(moved, ['b', 'c', 'a', 'd']);
  assert.deepEqual(list, ['a', 'b', 'c', 'd']);
});

test('move is a no-op for out of range or identical indexes', () => {
  assert.deepEqual(move(list, 1, 1), list);
  assert.deepEqual(move(list, -1, 2), list);
  assert.deepEqual(move(list, 0, 9), list);
});

test('move backwards places the item at the target index', () => {
  assert.deepEqual(move(list, 3, 0), ['d', 'a', 'b', 'c']);
});

test('withoutIndex removes one entry and ignores bad indexes', () => {
  assert.deepEqual(withoutIndex(list, 1), ['a', 'c', 'd']);
  assert.deepEqual(withoutIndex(list, 99), list);
});

test('insertFirst puts play-next at the head', () => {
  assert.deepEqual(insertFirst(list, 'x'), ['x', 'a', 'b', 'c', 'd']);
});

test('volume and position clamp to valid ranges', () => {
  assert.equal(clampVolume(140), 100);
  assert.equal(clampVolume(-3), 0);
  assert.equal(clampVolume(Number.NaN), 0);
  assert.equal(clampVolume(42.6), 43);
  assert.equal(clampPosition(50, 120), 50);
  assert.equal(clampPosition(200, 120), 120);
  assert.equal(clampPosition(-1, 120), 0);
});

test('splitCurrent separates the playing track', () => {
  const queue = [{ id: 'a' }, { id: 'b' }, { id: 'c' }];
  const split = splitCurrent(queue, 'b');
  assert.deepEqual(split.current, { id: 'b' });
  assert.deepEqual(split.rest, [{ id: 'a' }, { id: 'c' }]);
  const missing = splitCurrent(queue, 'zzz');
  assert.equal(missing.current, undefined);
  assert.equal(missing.rest.length, 3);
});
