'use client';
/* eslint-disable @next/next/no-html-link-for-pages */

import { FormEvent, useState } from 'react';

type SearchResult = {
  chunk_id: string;
  kind: 'verse_translation' | 'commentary';
  citation_label: string;
  source?: string;
  text: string;
  verse_references: string[];
  similarity: number;
};

type GroundedAnswer = {
  text: string;
  model: string;
  input_tokens?: number;
  output_tokens?: number;
};

type AskResponse = {
  answer?: GroundedAnswer | null;
  results?: SearchResult[];
  error?: string;
};

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? '';
const suggestions = [
  'How can I do my duty without worrying about the result?',
  'What does the Gita say about a restless mind?',
  'How should I respond when I feel uncertain?',
];

export default function AskPage() {
  const [question, setQuestion] = useState('');
  const [submittedQuestion, setSubmittedQuestion] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [answer, setAnswer] = useState<GroundedAnswer | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [hasSearched, setHasSearched] = useState(false);

  async function ask(event?: FormEvent, suggestedQuestion?: string) {
    event?.preventDefault();
    const query = (suggestedQuestion ?? question).trim();
    if (!query || loading) return;

    setQuestion(query);
    setSubmittedQuestion(query);
    setLoading(true);
    setError('');
    setAnswer(null);
    setHasSearched(true);
    try {
      const response = await fetch(`${apiURL}/api/v1/ask`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: query, scripture: 'BG' }),
      });
      const data = await response.json().catch(() => ({})) as AskResponse;
      if (!response.ok) {
        throw new Error(data.error ?? 'The search could not be completed.');
      }
      setResults(data.results ?? []);
      setAnswer(data.answer ?? null);
    } catch (caught) {
      setResults([]);
      setError(caught instanceof Error ? caught.message : 'The search could not be completed.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="site-shell ask-screen">
      <header className="topbar ask-topbar">
        <a className="brand" href="/" aria-label="Narada home">
          <span className="brand-mark" aria-hidden="true">न</span>
          <span>Narada</span>
        </a>
        <nav aria-label="Primary navigation">
          <a href="/">Today’s verse</a>
          <a href="/read">Read the Gita</a>
          <a className="current" href="/ask">Ask</a>
        </nav>
      </header>

      <section className="ask-page">
        <header className="ask-intro">
          <h1>Ask the Gita.</h1>
        </header>

        <form className="ask-form" onSubmit={ask}>
          <label htmlFor="gita-question">What is on your mind?</label>
          <textarea
            id="gita-question"
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            placeholder="I am doing the work, but I cannot stop worrying about the outcome…"
            rows={1}
            maxLength={1000}
            disabled={loading}
          />
          <div className="ask-form-footer">
            <span>{question.length}/1000</span>
            <button type="submit" disabled={!question.trim() || loading}>
              {loading ? 'Listening…' : 'Find guidance'}
              <span aria-hidden="true">→</span>
            </button>
          </div>
        </form>

        <p className="privacy-note">
          Narada does not collect or store user information. Questions and answers are not linked to you or any user identity.
        </p>

        {!hasSearched && (
          <section className="question-starters" aria-labelledby="question-starters-title">
            <p id="question-starters-title">You might ask</p>
            <div>
              {suggestions.map((suggestion) => (
                <button key={suggestion} onClick={() => ask(undefined, suggestion)}>
                  <span>{suggestion}</span><span aria-hidden="true">↗</span>
                </button>
              ))}
            </div>
          </section>
        )}

        {hasSearched && (
          <section className="ask-results" aria-live="polite" aria-busy={loading}>
            {loading && (
              <div className="search-state">
                <span className="search-pulse" aria-hidden="true">न</span>
                <p>Looking across the Gita and its commentary…</p>
              </div>
            )}

            {!loading && error && (
              <div className="search-state search-error" role="alert">
                <strong>We couldn&apos;t consult the text just now.</strong>
                <p>{error}</p>
                <button onClick={() => ask(undefined, submittedQuestion)}>Try again</button>
              </div>
            )}

            {!loading && !error && results.length === 0 && (
              <div className="search-state">
                <strong>No close passages were found.</strong>
                <p>Try asking the question in a different way.</p>
              </div>
            )}

            {!loading && !error && results.length > 0 && (
              <>
                {answer && (
                  <article className="grounded-answer">
                    <header>
                      <span className="brand-mark" aria-hidden="true">न</span>
                      <div>
                        <span>Grounded answer</span>
                        <small>Generated only from the passages below</small>
                      </div>
                    </header>
                    <p>{answer.text}</p>
                    <footer>
                      <span>AI-generated interpretation</span>
                      {(answer.input_tokens || answer.output_tokens) && (
                        <span>{(answer.input_tokens ?? 0) + (answer.output_tokens ?? 0)} tokens used</span>
                      )}
                    </footer>
                  </article>
                )}
                <div className="evidence-heading">
                  <span>Source passages</span>
                  <span>{results.length} retrieved</span>
                </div>
                <div className="passage-results">
                  {results.map((result, index) => (
                  <article className="search-result" key={result.chunk_id}>
                    <div className="result-number">{String(index + 1).padStart(2, '0')}</div>
                    <div className="result-content">
                      <header>
                        <div>
                          <span className={`result-kind ${result.kind}`}>
                            {result.kind === 'commentary' ? 'Commentary' : 'Translation'}
                          </span>
                          <h3>{result.citation_label}</h3>
                        </div>
                        <small>{result.source}</small>
                      </header>
                      <p>{result.text}</p>
                      <footer>
                        <span>{result.verse_references.join(' · ')}</span>
                        <span>{Math.round(result.similarity * 100)}% related</span>
                      </footer>
                    </div>
                  </article>
                  ))}
                </div>
              </>
            )}
          </section>
        )}
      </section>

      <footer className="site-footer ask-footer">
        <span className="brand-mark" aria-hidden="true">न</span>
      </footer>
    </main>
  );
}
