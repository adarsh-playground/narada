'use client';
/* eslint-disable @next/next/no-html-link-for-pages */

import { useEffect, useState } from 'react';
import { WordMeanings } from '../components/word-meanings';

type Chapter = {
  number: number;
  title: string;
  original_title: string;
  meaning: string;
};

type Verse = {
  reference: string;
  original_text: string;
  transliteration: string;
  word_meanings: string;
  translations: Array<{ source: string; text: string }>;
  commentaries: Array<{ source: string; citation_label?: string; text: string }>;
};

type ChaptersResponse = { chapters: Chapter[] };
type VersesResponse = { verses: Verse[] };

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? '';
const compactSanskrit = (text: string) => text.replace(/\n\s*\n+/g, '\n');

export default function ReadPage() {
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [selectedChapter, setSelectedChapter] = useState(1);
  const [verses, setVerses] = useState<Verse[]>([]);
  const [activeVerse, setActiveVerse] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      fetch(`${apiURL}/api/v1/scriptures/BG/chapters`).then((response) => {
        if (!response.ok) throw new Error();
        return response.json() as Promise<ChaptersResponse>;
      }),
      fetch(`${apiURL}/api/v1/scriptures/BG/chapters/1/verses`).then((response) => {
        if (!response.ok) throw new Error();
        return response.json() as Promise<VersesResponse>;
      }),
    ])
      .then(([chapterData, verseData]) => {
        setChapters(chapterData.chapters);
        setVerses(verseData.verses);
      })
      .catch(() => {
        setChapters([]);
        setVerses([]);
      })
      .finally(() => setLoading(false));
  }, []);

  async function openChapter(chapterNumber: number) {
    setSelectedChapter(chapterNumber);
    setActiveVerse(0);
    setLoading(true);
    try {
      const response = await fetch(
        `${apiURL}/api/v1/scriptures/BG/chapters/${chapterNumber}/verses`,
      );
      if (!response.ok) throw new Error();
      const data = await response.json() as VersesResponse;
      setVerses(data.verses);
    } catch {
      setVerses([]);
    } finally {
      setLoading(false);
    }
  }

  const chapter = chapters.find((item) => item.number === selectedChapter);
  const verse = verses[activeVerse];
  const chapterOptions = chapters.length
    ? chapters
    : Array.from({ length: 18 }, (_, index) => ({ number: index + 1, title: `Chapter ${index + 1}` }));

  return (
    <main className="site-shell read-screen">
      <header className="topbar read-topbar">
        <a className="brand" href="/" aria-label="Narada home">
          <span className="brand-mark" aria-hidden="true">न</span>
          <span>Narada</span>
        </a>
        <nav aria-label="Primary navigation">
          <a href="/">Today’s verse</a>
          <a className="current" href="/read">Read the Gita</a>
          <a href="/ask">Ask</a>
        </nav>
      </header>

      <section className="reader reader-page">
        <div className="section-heading">
          <div>
            <p className="eyebrow">The complete text</p>
            <h1>Read the Gita,<br />one verse at a time.</h1>
          </div>
          <p>Move through all eighteen chapters with the Sanskrit, translation, and Swami Chinmayananda&apos;s commentary together.</p>
        </div>

        <div className="chapter-picker" aria-label="Choose a chapter">
          {chapters.length ? chapters.map((item) => (
            <button
              key={item.number}
              className={item.number === selectedChapter ? 'active' : ''}
              onClick={() => openChapter(item.number)}
              aria-pressed={item.number === selectedChapter}
            >
              <span>{String(item.number).padStart(2, '0')}</span>
              <small>{item.title}</small>
            </button>
          )) : Array.from({ length: 18 }, (_, index) => (
            <span className="chapter-placeholder" key={index}>{String(index + 1).padStart(2, '0')}</span>
          ))}
        </div>

        <select
          className="chapter-select"
          aria-label="Choose a chapter"
          value={selectedChapter}
          onChange={(event) => openChapter(Number(event.target.value))}
        >
          {chapterOptions.map((item) => (
            <option key={item.number} value={item.number}>Chapter {item.number} · {item.title}</option>
          ))}
        </select>

        <article className="reader-card" aria-busy={loading}>
          <header>
            <div>
              <span>Chapter {selectedChapter}</span>
              <h3>{chapter?.title ?? 'Arjuna Visada Yoga'}</h3>
              <p>{chapter?.meaning ?? "Arjuna's Dilemma"}</p>
            </div>
            <strong>{verse?.reference ?? `BG ${selectedChapter}.1`}</strong>
          </header>

          {loading ? (
            <div className="reader-loading">Opening chapter…</div>
          ) : verse ? (
            <div className="reader-text">
              <p className="sanskrit">{compactSanskrit(verse.original_text)}</p>
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
              <div className="meaning-block">
                <span>Word meaning</span>
                <WordMeanings text={verse.word_meanings} />
              </div>
            </div>
          ) : (
            <div className="reader-loading">Start the Narada API to read this chapter.</div>
          )}

          <footer>
            <button
              onClick={() => setActiveVerse((current) => Math.max(0, current - 1))}
              disabled={activeVerse === 0 || loading}
              aria-label="Previous verse"
            >← <span>Previous</span></button>
            <span>{verses.length ? `${activeVerse + 1} of ${verses.length}` : '—'}</span>
            <button
              onClick={() => setActiveVerse((current) => Math.min(verses.length - 1, current + 1))}
              disabled={!verses.length || activeVerse === verses.length - 1 || loading}
              aria-label="Next verse"
            ><span>Next</span> →</button>
          </footer>
        </article>
      </section>

      <footer className="site-footer">
        <span className="brand-mark" aria-hidden="true">न</span>
        <span>701 verses · 18 chapters</span>
      </footer>
    </main>
  );
}
