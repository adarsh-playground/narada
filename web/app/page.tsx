'use client';
/* eslint-disable @next/next/no-html-link-for-pages */

import { useState } from 'react';
import { WordMeanings } from './components/word-meanings';

type Verse = {
  reference: string;
  original_text: string;
  transliteration: string;
  word_meanings: string;
  translations: Array<{ source: string; text: string }>;
  commentaries: Array<{ source: string; citation_label?: string; text: string }>;
};

const sampleVerse: Verse = {
  reference: 'BG 2.47',
  original_text:
    'कर्मण्येवाधिकारस्ते मा फलेषु कदाचन।\nमा कर्मफलहेतुर्भूर्मा ते सङ्गोऽस्त्वकर्मणि।।',
  transliteration:
    'karmaṇy-evādhikāras te mā phaleṣhu kadāchana\nmā karma-phala-hetur bhūr mā te saṅgo ’stvakarmaṇi',
  word_meanings:
    'You have a right to perform your prescribed duties, but you are not entitled to the fruits of your actions.',
  translations: [{
    source: 'Swami Chinmayananda',
    text: 'Thy right is to work only, but never to its fruits; let not the fruit-of-action be thy motive, nor let thy attachment be to inaction.',
  }],
  commentaries: [{
    source: 'Swami Chinmayananda',
    citation_label: 'BG 2.47',
    text: 'This verse explains the secret of inspired action: give yourself fully to the work at hand without allowing anxiety about its result to weaken the quality of the action.',
  }],
};

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? '';
const compactSanskrit = (text: string) => text.replace(/\n\s*\n+/g, '\n');

export default function Home() {
  const [verse, setVerse] = useState(sampleVerse);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function findAnotherVerse() {
    setLoading(true);
    setError('');
    try {
      const response = await fetch(`${apiURL}/api/v1/verses/random`);
      if (!response.ok) throw new Error();
      setVerse(await response.json());
    } catch {
      setError('Start the Narada API to discover another verse.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="site-shell">
      <header className="topbar">
        <a className="brand" href="/" aria-label="Narada home">
          <span className="brand-mark" aria-hidden="true">न</span>
          <span>Narada</span>
        </a>
        <nav aria-label="Primary navigation">
          <a className="current" href="/">Today’s verse</a>
          <a href="/read">Read the Gita</a>
          <a href="/ask">Ask</a>
        </nav>
      </header>

      <section className="hero" id="top">
        <article className="verse-card" aria-live="polite" aria-busy={loading}>
          <div className="verse-meta">
            <span>श्रीमद्भगवद्गीता</span>
            <strong>{verse.reference}</strong>
          </div>
          <p className="sanskrit">{compactSanskrit(verse.original_text)}</p>
          <div className="rule" aria-hidden="true"><span>✦</span></div>
          <p className="transliteration">{verse.transliteration}</p>
          {verse.translations?.map((translation) => (
            <section className="reading-passage translation-passage" key={`${translation.source}-${translation.text}`}>
              <div className="passage-label"><span>Translation</span><small>{translation.source}</small></div>
              <p>{translation.text}</p>
            </section>
          ))}
          {verse.commentaries?.map((commentary) => (
            <section className="reading-passage commentary-passage" key={`${commentary.source}-${commentary.citation_label ?? commentary.text}`}>
              <div className="passage-label"><span>Commentary</span><small>{commentary.source}</small></div>
              <p>{commentary.text}</p>
            </section>
          ))}
          <WordMeanings className="meaning" text={verse.word_meanings} />
          {error && <p className="inline-error" role="status">{error}</p>}
          <button className="primary-action" onClick={findAnotherVerse} disabled={loading}>
            <span>{loading ? 'Finding a verse…' : 'Another verse'}</span>
            <span aria-hidden="true">↗</span>
          </button>
        </article>
      </section>

      <footer className="home-footer">
        <p>Ready to go deeper?</p>
        <a href="/read">Read all 18 chapters <span aria-hidden="true">→</span></a>
      </footer>
    </main>
  );
}
