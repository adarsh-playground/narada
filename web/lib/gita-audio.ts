export function getGitaAudioUrl(chapter: number, verse: number): string {
  if (!Number.isInteger(chapter) || chapter < 1 || chapter > 18 ||
      !Number.isInteger(verse) || verse < 1 || verse > 999) {
    throw new RangeError('Invalid Bhagavad Gita chapter or verse');
  }
  const base = process.env.NEXT_PUBLIC_GITA_AUDIO_BASE_URL ??
    'https://d32bst75r6nr99.cloudfront.net/bhagavad_gita/dhruv_jaradi';
  return `${base.replace(/\/+$/, '')}/${String(chapter).padStart(2, '0')}/${String(verse).padStart(3, '0')}.mp3`;
}

export type AudioState = 'idle' | 'loading' | 'playing' | 'paused' | 'error';
type Listener = (state: AudioState) => void;

// The element is created only in response to a playback request, never during SSR.
export class GitaAudioPlayer {
  private audio: HTMLAudioElement | undefined;
  private owner: Listener | undefined;
  private url: string | undefined;
  private state: AudioState = 'idle';
  private generation = 0;
  private removeListeners: (() => void) | undefined;
  private createAudio: () => HTMLAudioElement;

  constructor(createAudio: () => HTMLAudioElement = () => new Audio()) {
    this.createAudio = createAudio;
  }

  private update(state: AudioState) {
    this.state = state;
    this.owner?.(state);
  }

  stop(owner?: Listener) {
    if (owner && this.owner !== owner) return;
    this.generation++;
    this.removeListeners?.();
    this.removeListeners = undefined;
    if (this.audio) {
      this.audio.pause();
      this.audio.removeAttribute('src');
      this.audio.load();
    }
    this.update('idle');
    this.owner = undefined;
    this.url = undefined;
  }

  async toggle(url: string, owner: Listener, onEnded?: () => void) {
    if (this.owner === owner && this.url === url && (this.state === 'playing' || this.state === 'loading')) {
      this.generation++;
      this.audio?.pause();
      this.update('paused');
      return;
    }
    if (this.owner !== owner || this.url !== url || this.state === 'error') {
      this.stop();
      this.owner = owner;
      this.url = url;
      this.audio ??= this.createAudio();
      const audio = this.audio;
      audio.preload = 'none';
      const handlers: Record<string, EventListener> = {
        playing: () => this.update('playing'),
        waiting: () => this.update('loading'),
        pause: () => { if (this.state !== 'error') this.update('paused'); },
        ended: () => { this.update('idle'); onEnded?.(); },
        error: () => { this.generation++; this.update('error'); },
      };
      for (const [event, handler] of Object.entries(handlers)) audio.addEventListener(event, handler);
      this.removeListeners = () => {
        for (const [event, handler] of Object.entries(handlers)) audio.removeEventListener(event, handler);
      };
      audio.src = url;
    }
    const generation = ++this.generation;
    this.update('loading');
    try {
      await this.audio!.play();
      if (generation === this.generation) this.update('playing');
    } catch {
      if (generation === this.generation) this.update('error');
    }
  }
}

export const gitaAudioPlayer = new GitaAudioPlayer();


export type GitaTrack = { chapter: number; verse: number };
export type ListeningSnapshot = { index: number; state: AudioState | 'complete' };

// Queue metadata is lightweight; the shared player loads just the current MP3.
export class GitaAudioQueue {
  private tracks: GitaTrack[] = [];
  private index = 0;
  private state: ListeningSnapshot['state'] = 'idle';
  private chapterOnly = false;
  private player: GitaAudioPlayer;
  private notify: (snapshot: ListeningSnapshot) => void;
  private listener: Listener = state => { this.state = state; this.publish(); };

  constructor(player: GitaAudioPlayer, notify: (snapshot: ListeningSnapshot) => void) {
    this.player = player;
    this.notify = notify;
  }

  private publish() { this.notify({ index: this.index, state: this.state }); }

  setTracks(tracks: GitaTrack[]) {
    this.stop();
    this.tracks = tracks;
    this.index = 0;
    this.publish();
  }

  setChapterOnly(value: boolean) { this.chapterOnly = value; }

  select(index: number) {
    if (!Number.isInteger(index) || index < 0 || index >= this.tracks.length) return;
    this.stop();
    this.index = index;
    this.publish();
  }

  stop() {
    this.player.stop(this.listener);
    this.state = 'idle';
    this.publish();
  }

  toggle() {
    const track = this.tracks[this.index];
    if (!track) return;
    return this.player.toggle(getGitaAudioUrl(track.chapter, track.verse), this.listener, () => {
      const next = this.tracks[this.index + 1];
      if (!next || (this.chapterOnly && next.chapter !== track.chapter)) {
        this.state = 'complete';
        this.publish();
        return;
      }
      this.select(this.index + 1);
      void this.toggle();
    });
  }
}
