import test from 'node:test';
import assert from 'node:assert/strict';
import { getGitaAudioUrl, GitaAudioPlayer, GitaAudioQueue } from '../lib/gita-audio.ts';
class FakeAudio extends EventTarget {
  src = ''; preload = ''; pauses = 0; loads = 0;
  play = async () => {};
  pause() { this.pauses++; this.emit('pause'); }
  removeAttribute() { this.src = ''; }
  load() { this.loads++; }
  emit(event) { this.dispatchEvent(new Event(event)); }
}
function setup() {
  const audio = new FakeAudio(); let created = 0;
  const player = new GitaAudioPlayer(() => { created++; return audio; });
  const states = []; const listener = state => states.push(state);
  return { audio, player, states, listener, created: () => created };
}
test('URL padding, configured base and validation', () => {
  const previous = process.env.NEXT_PUBLIC_GITA_AUDIO_BASE_URL;
  try {
    delete process.env.NEXT_PUBLIC_GITA_AUDIO_BASE_URL;
    assert.equal(getGitaAudioUrl(14, 1), 'https://d32bst75r6nr99.cloudfront.net/bhagavad_gita/dhruv_jaradi/14/001.mp3');
    assert.match(getGitaAudioUrl(1, 47), /\/01\/047.mp3$/);
    process.env.NEXT_PUBLIC_GITA_AUDIO_BASE_URL = 'https://audio.example/gita/';
    assert.equal(getGitaAudioUrl(18, 78), 'https://audio.example/gita/18/078.mp3');
    for (const args of [[0, 1], [19, 1], [1, 0], [1.5, 1], [1, NaN]]) assert.throws(() => getGitaAudioUrl(...args), RangeError);
  } finally {
    if (previous === undefined) delete process.env.NEXT_PUBLIC_GITA_AUDIO_BASE_URL;
    else process.env.NEXT_PUBLIC_GITA_AUDIO_BASE_URL = previous;
  }
});
test('lazy playback, pause, resume, buffering and completion', async () => {
  const { audio, player, states, listener, created } = setup();
  assert.equal(created(), 0);
  await player.toggle('one.mp3', listener);
  assert.deepEqual(states, ['loading', 'playing']);
  assert.equal(audio.preload, 'none');
  await player.toggle('one.mp3', listener);
  assert.equal(states.at(-1), 'paused');
  await player.toggle('one.mp3', listener);
  audio.emit('waiting'); assert.equal(states.at(-1), 'loading');
  audio.emit('playing'); assert.equal(states.at(-1), 'playing');
  audio.emit('ended'); assert.equal(states.at(-1), 'idle');
  assert.equal(created(), 1);
});
test('switching stops previous verse and reuses element', async () => {
  const { audio, player, states, listener, created } = setup();
  await player.toggle('one.mp3', listener);
  const nextStates = []; const next = state => nextStates.push(state);
  await player.toggle('two.mp3', next);
  assert.equal(states.at(-1), 'idle'); assert.equal(audio.src, 'two.mp3');
  assert.ok(audio.pauses > 0); assert.equal(created(), 1);
  player.stop(listener); assert.equal(nextStates.at(-1), 'playing');
});
test('play rejection, media error and retry', async () => {
  const { audio, player, states, listener } = setup();
  audio.play = async () => { throw new Error('Network unavailable'); };
  await player.toggle('one.mp3', listener); assert.equal(states.at(-1), 'error');
  audio.play = async () => {};
  await player.toggle('one.mp3', listener); assert.equal(states.at(-1), 'playing');
  audio.emit('error'); assert.equal(states.at(-1), 'error');
});
test('navigation cleanup clears source and removes listeners', async () => {
  const { audio, player, states, listener } = setup();
  await player.toggle('one.mp3', listener); player.stop(listener);
  assert.equal(audio.src, ''); assert.equal(audio.loads, 1); assert.equal(states.at(-1), 'idle');
  const count = states.length;
  for (const event of ['playing', 'waiting', 'pause', 'ended', 'error']) audio.emit(event);
  assert.equal(states.length, count);
});
test('pending playback cannot revive state after navigation or pause', async () => {
  for (const stop of [true, false]) {
    const { audio, player, states, listener } = setup(); let resolve;
    audio.play = () => new Promise(r => { resolve = r; });
    const pending = player.toggle('one.mp3', listener);
    assert.equal(states.at(-1), 'loading');
    if (stop) player.stop(listener); else await player.toggle('one.mp3', listener);
    resolve(); await pending; assert.equal(states.at(-1), stop ? 'idle' : 'paused');
  }
});

function queueSetup() {
  const { audio, player } = setup();
  const snapshots = [];
  const queue = new GitaAudioQueue(player, snapshot => snapshots.push(snapshot));
  queue.setTracks([{ chapter: 1, verse: 46 }, { chapter: 1, verse: 47 }, { chapter: 2, verse: 1 }]);
  return { audio, queue, snapshots };
}
const settle = () => new Promise(resolve => setImmediate(resolve));
test('queue starts at selected verse and crosses chapter boundary', async () => {
  const { audio, queue, snapshots } = queueSetup();
  queue.select(1); await queue.toggle();
  assert.match(audio.src, /01\/047.mp3$/);
  audio.emit('ended'); await settle();
  assert.match(audio.src, /02\/001.mp3$/);
  assert.deepEqual(snapshots.at(-1), { index: 2, state: 'playing' });
  audio.emit('ended'); assert.equal(snapshots.at(-1).state, 'complete');
});
test('chapter-only listening stops at chapter boundary', async () => {
  const { audio, queue, snapshots } = queueSetup();
  queue.setChapterOnly(true); await queue.toggle();
  audio.emit('ended'); await settle();
  assert.match(audio.src, /01\/047.mp3$/);
  audio.emit('ended'); await settle();
  assert.deepEqual(snapshots.at(-1), { index: 1, state: 'complete' });
});
test('queue selection and navigation stop playback and auto advance', async () => {
  const { audio, queue, snapshots } = queueSetup();
  await queue.toggle(); await queue.toggle();
  assert.equal(snapshots.at(-1).state, 'paused');
  await queue.toggle(); queue.select(2);
  assert.equal(audio.src, '');
  audio.emit('ended'); assert.deepEqual(snapshots.at(-1), { index: 2, state: 'idle' });
  await queue.toggle(); queue.stop(); audio.emit('ended');
  assert.deepEqual(snapshots.at(-1), { index: 2, state: 'idle' });
});
test('auto advance errors stop the queue and can be retried', async () => {
  const { audio, queue, snapshots } = queueSetup();
  await queue.toggle();
  audio.play = async () => { throw new Error('Unavailable'); };
  audio.emit('ended'); await settle();
  assert.deepEqual(snapshots.at(-1), { index: 1, state: 'error' });
  audio.play = async () => {}; await queue.toggle();
  assert.equal(snapshots.at(-1).state, 'playing');
});
