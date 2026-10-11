'use client';

import { useEffect, useState } from 'react';
import { getGitaAudioUrl, gitaAudioPlayer, type AudioState } from '../../lib/gita-audio';

export function GitaAudioButton({ reference }: { reference: string }) {
  const [state, setState] = useState<AudioState>('idle');
  useEffect(() => {
    const stop = () => gitaAudioPlayer.stop(setState);
    window.addEventListener('pagehide', stop);
    return () => {
      window.removeEventListener('pagehide', stop);
      stop();
    };
  }, []);
  const match = /^BG\s+(\d+)\.(\d+)$/.exec(reference);
  if (!match) return null;
  const active = state === 'playing' || state === 'loading';
  const label = `${active ? 'Pause' : 'Play'} Sanskrit recitation for ${reference}`;

  return (
    <span className="gita-audio-control">
      <button
        type="button"
        className={`gita-audio-button ${active ? 'is-active' : ''}`}
        aria-label={label}
        aria-pressed={active}
        title={state === 'error' ? 'Audio unavailable. Try again.' : label}
        onClick={() => { void gitaAudioPlayer.toggle(getGitaAudioUrl(Number(match[1]), Number(match[2])), setState); }}
      >
        {state === 'loading' ? <span className="audio-spinner" aria-hidden="true" /> : (
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
            {state === 'playing' ? <path d="M8 5v14M16 5v14" strokeWidth="3" /> : <><path d="M11 5 6 9H3v6h3l5 4V5Z" /><path d="M15 8a6 6 0 0 1 0 8m3-11a10 10 0 0 1 0 14" /></>}
          </svg>
        )}
      </button>
      <span className={state === 'error' ? 'audio-error' : 'audio-status'} role="status">{state === 'error' ? 'Audio unavailable. Try again.' : state === 'loading' ? 'Loading recitation…' : ''}</span>
    </span>
  );
}
