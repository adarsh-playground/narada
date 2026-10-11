'use client';
/* eslint-disable @next/next/no-html-link-for-pages */

import { PrimaryNavigation } from '../components/primary-navigation';
import { useEffect, useState } from 'react';
import { GitaAudioQueue, gitaAudioPlayer, type GitaTrack, type ListeningSnapshot } from '../../lib/gita-audio';

type Chapter = { number: number; title: string; verse_count: number };
type Verse = { reference: string; original_text: string; transliteration: string };
const apiURL = process.env.NEXT_PUBLIC_API_URL ?? '';

export default function ListenPage() {
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [tracks, setTracks] = useState<GitaTrack[]>([]);
  const [snapshot, setSnapshot] = useState<ListeningSnapshot>({ index: 0, state: 'idle' });
  const [queue] = useState(() => new GitaAudioQueue(gitaAudioPlayer, setSnapshot));
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  const [chapterOnly, setChapterOnly] = useState(false);
  const [verses, setVerses] = useState<Verse[]>([]);
  const track = tracks[snapshot.index];
  const chapterNumber = track?.chapter;

  useEffect(() => {
    const abort = new AbortController();
    fetch(`${apiURL}/api/v1/scriptures/BG/chapters`, { signal: abort.signal })
      .then(async response => {
        if (!response.ok) throw new Error();
        const data = await response.json() as { chapters: Chapter[] };
        const ordered = [...data.chapters].sort((a, b) => a.number - b.number);
        if (!ordered.length || ordered.some(c => !Number.isInteger(c.verse_count) || c.verse_count < 1)) throw new Error();
        if (abort.signal.aborted) return;
        const items = ordered.flatMap(c => Array.from({ length: c.verse_count }, (_, i) => ({ chapter: c.number, verse: i + 1 })));
        setChapters(ordered);
        setTracks(items);
        queue.setTracks(items);
      })
      .catch(() => { if (!abort.signal.aborted) setError('Unable to load chapters. Check that the Narada API is running, then try again.'); })
      .finally(() => { if (!abort.signal.aborted) setLoading(false); });
    return () => abort.abort();
  }, [queue, retry]);

  useEffect(() => {
    const stop = () => queue.stop();
    window.addEventListener('pagehide', stop);
    return () => { window.removeEventListener('pagehide', stop); stop(); };
  }, [queue]);

  useEffect(() => {
    if (!chapterNumber) return;
    const abort = new AbortController();
    fetch(`${apiURL}/api/v1/scriptures/BG/chapters/${chapterNumber}/verses`, { signal: abort.signal })
      .then(async response => {
        if (!response.ok) throw new Error();
        const data = await response.json() as { verses: Verse[] };
        if (!abort.signal.aborted) setVerses(data.verses);
      }).catch(() => { /* Audio can still play when the verse text is unavailable. */ });
    return () => abort.abort();
  }, [chapterNumber]);

  const chapter = chapters.find(c => c.number === chapterNumber);
  const verse = verses.find(v => v.reference === `BG ${chapterNumber}.${track?.verse}`);
  const active = snapshot.state === 'playing' || snapshot.state === 'loading';
  const status = snapshot.state === 'error' ? 'Audio could not play. Press Play to try again.'
    : snapshot.state === 'complete' ? (chapterOnly ? 'Chapter complete.' : 'You have reached the end of the Gita.')
    : snapshot.state === 'loading' ? 'Loading recitation…'
    : snapshot.state === 'playing' ? 'Playing · continues verse by verse'
    : snapshot.state === 'paused' ? 'Paused' : 'Choose a starting verse and press Play.';

  return (
    <main className="site-shell read-screen">
      <header className="topbar read-topbar">
        <a className="brand" href="/" aria-label="Narada home"><span className="brand-mark" aria-hidden="true">न</span><span>Narada</span></a>
        <PrimaryNavigation current="/listen" />
      </header>
      <section className="reader reader-page">
        <div className="section-heading">
          <div><p className="eyebrow">Sanskrit recitations</p><h1>Listen to the Gita.</h1></div>
          <p>Begin with any chapter and verse. Follow the recitations one by one, through a single chapter or all the way to the end.</p>
        </div>
        {loading ? <p role="status">Opening chapters…</p> : error ? (
          <div role="status"><p>{error}</p><button className="listen-retry" onClick={() => { setError(''); setLoading(true); setRetry(n => n + 1); }}>Try again</button></div>
        ) : (
          <article className="reader-card listen-card">
            <header><div><span>Now selected · Chapter {chapterNumber}</span><h3>{chapter?.title}</h3></div><strong>BG {chapterNumber}.{track?.verse}</strong></header>
            <div className="reader-text">
              <div className="listen-selectors">
                <label>Chapter<select value={chapterNumber} onChange={e => queue.select(tracks.findIndex(t => t.chapter === Number(e.target.value)))}>{chapters.map(c => <option key={c.number} value={c.number}>{c.number} · {c.title}</option>)}</select></label>
                <label>Verse<select value={track?.verse} onChange={e => queue.select(tracks.findIndex(t => t.chapter === chapterNumber && t.verse === Number(e.target.value)))}>{tracks.filter(t => t.chapter === chapterNumber).map(t => <option key={t.verse} value={t.verse}>Verse {t.verse}</option>)}</select></label>
                <label>Continue<select value={chapterOnly ? 'chapter' : 'gita'} onChange={e => { const only = e.target.value === 'chapter'; setChapterOnly(only); queue.setChapterOnly(only); }}><option value="gita">Through the Gita</option><option value="chapter">Until this chapter ends</option></select></label>
              </div>
              <p className="listen-status" role="status">{status}</p>
              <div className="listen-actions">
                <button className="listen-play" onClick={() => { void queue.toggle(); }} aria-label={`${active ? 'Pause' : 'Play'} Sanskrit recitation`} aria-pressed={active}>{snapshot.state === 'loading' && <span className="audio-spinner" aria-hidden="true" />}{active ? 'Pause' : snapshot.state === 'paused' ? 'Resume' : 'Play'}</button>
                <button onClick={() => queue.stop()}>Stop</button>
              </div>
              {verse && <><p className="sanskrit">{verse.original_text.replace(/\n\s*\n+/g, '\n')}</p><p className="transliteration">{verse.transliteration}</p></>}
              <p className="listen-hint">Changing chapter or verse stops playback. Press Play to begin from the new location.</p>
            </div>
            <footer>
              <button disabled={snapshot.index === 0} onClick={() => queue.select(snapshot.index - 1)} aria-label="Previous verse">← <span>Previous</span></button>
              <span>{snapshot.index + 1} of {tracks.length} verses</span>
              <button disabled={snapshot.index === tracks.length - 1} onClick={() => queue.select(snapshot.index + 1)} aria-label="Next verse"><span>Next</span> →</button>
            </footer>
          </article>
        )}
      </section>
      <footer className="site-footer"><span className="brand-mark" aria-hidden="true">न</span><nav className="footer-links" aria-label="Footer navigation"><a href="/about#credits">About &amp; Credits</a></nav></footer>
    </main>
  );
}
