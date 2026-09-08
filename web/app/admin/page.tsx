'use client';

import { FormEvent, useEffect, useState } from 'react';
import Link from 'next/link';

type Interaction = {
  id: string;
  scripture: string;
  question: string;
  answer_text?: string;
  status: 'pending' | 'completed' | 'failed';
  error_message?: string;
  embedding_model: string;
  answer_model: string;
  prompt_version: string;
  embedding_input_tokens: number;
  answer_input_tokens: number;
  answer_output_tokens: number;
  total_cost_usd: number;
  duration_ms?: number;
  cached: boolean;
  created_at: string;
};

type InteractionPage = {
  interactions: Interaction[];
  summary: {
    total_interactions: number;
    completed: number;
    failed: number;
    pending: number;
    cached: number;
    total_tokens: number;
    total_cost_usd: number;
  };
  total: number;
  limit: number;
  offset: number;
};

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? '';
const pageSize = 50;

function formatCost(value: number) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', minimumFractionDigits: 4, maximumFractionDigits: 6,
  }).format(value);
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US').format(value);
}

export default function AdminPage() {
  const [token, setToken] = useState('');
  const [tokenInput, setTokenInput] = useState('');
  const [data, setData] = useState<InteractionPage | null>(null);
  const [status, setStatus] = useState('');
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const saved = sessionStorage.getItem('narada-admin-token');
    if (!saved) return;
    let active = true;
    queueMicrotask(() => {
      if (active) {
        setLoading(true);
        setToken(saved);
      }
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!token) return;
    const controller = new AbortController();
    const query = new URLSearchParams({ limit: String(pageSize), offset: String(offset) });
    if (status) query.set('status', status);
    fetch(`${apiURL}/api/v1/admin/ask-interactions?${query}`, {
      headers: { Authorization: `Bearer ${token}` },
      signal: controller.signal,
    })
      .then(async (response) => {
        if (response.status === 401) throw new Error('That admin token is not valid.');
        if (!response.ok) throw new Error('The admin history could not be loaded.');
        return response.json() as Promise<InteractionPage>;
      })
      .then(setData)
      .catch((requestError: Error) => {
        if (requestError.name !== 'AbortError') setError(requestError.message);
      })
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, [token, status, offset]);

  function unlock(event: FormEvent) {
    event.preventDefault();
    const nextToken = tokenInput.trim();
    if (!nextToken) return;
    sessionStorage.setItem('narada-admin-token', nextToken);
    setLoading(true);
    setError('');
    setToken(nextToken);
  }

  function lock() {
    sessionStorage.removeItem('narada-admin-token');
    setToken('');
    setTokenInput('');
    setData(null);
    setError('');
  }

  if (!token) {
    return (
      <main className="admin-login">
        <section>
          <Link className="brand" href="/" aria-label="Narada home">
            <span className="brand-mark" aria-hidden="true">न</span><span>Narada</span>
          </Link>
          <p className="admin-kicker">Private administration</p>
          <h1>Ask history</h1>
          <p>Enter the admin token configured on the Narada API.</p>
          <form onSubmit={unlock}>
            <label htmlFor="admin-token">Admin token</label>
            <input id="admin-token" type="password" value={tokenInput}
              onChange={(event) => setTokenInput(event.target.value)} autoComplete="current-password" autoFocus />
            <button type="submit">Open dashboard</button>
          </form>
        </section>
      </main>
    );
  }

  return (
    <main className="admin-page">
      <header className="admin-header">
        <div>
          <Link className="brand" href="/" aria-label="Narada home">
            <span className="brand-mark" aria-hidden="true">न</span><span>Narada</span>
          </Link>
          <span>Administration</span>
        </div>
        <button type="button" onClick={lock}>Lock dashboard</button>
      </header>

      <section className="admin-title">
        <div><p className="admin-kicker">Anonymous usage</p><h1>Ask interactions</h1></div>
        <label>Status
          <select value={status} onChange={(event) => { setLoading(true); setError(''); setStatus(event.target.value); setOffset(0); }}>
            <option value="">All statuses</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="pending">Pending</option>
          </select>
        </label>
      </section>

      {data && (
        <section className="admin-stats" aria-label="Ask history summary">
          <article><span>Questions</span><strong>{formatNumber(data.summary.total_interactions)}</strong></article>
          <article><span>Completed</span><strong>{formatNumber(data.summary.completed)}</strong></article>
          <article><span>Cache hits</span><strong>{formatNumber(data.summary.cached)}</strong></article>
          <article><span>Total tokens</span><strong>{formatNumber(data.summary.total_tokens)}</strong></article>
          <article><span>Estimated cost</span><strong>{formatCost(data.summary.total_cost_usd)}</strong></article>
        </section>
      )}

      {error && <p className="admin-error" role="alert">{error}</p>}
      {loading && !data && <p className="admin-loading">Loading ask history…</p>}

      {data && (
        <section className="admin-history" aria-busy={loading}>
          <div className="admin-table-wrap">
            <table>
              <thead><tr><th>Asked</th><th>Question and answer</th><th>Status</th><th>Tokens</th><th>Cost</th><th>Time</th></tr></thead>
              <tbody>
                {data.interactions.map((item) => {
                  const totalTokens = item.embedding_input_tokens + item.answer_input_tokens + item.answer_output_tokens;
                  return (
                    <tr key={item.id}>
                      <td><time dateTime={item.created_at}>{new Date(item.created_at).toLocaleString()}</time><small>{item.scripture}</small></td>
                      <td className="admin-copy"><strong>{item.question}</strong>
                        {item.answer_text && <details><summary>View answer</summary><p>{item.answer_text}</p></details>}
                        {item.error_message && <p className="admin-row-error">{item.error_message}</p>}
                        <small>{item.answer_model} · {item.prompt_version}</small>
                      </td>
                      <td><span className={`admin-status ${item.status}`}>{item.cached ? 'cached' : item.status}</span></td>
                      <td><strong>{formatNumber(totalTokens)}</strong><small>E {item.embedding_input_tokens} · In {item.answer_input_tokens} · Out {item.answer_output_tokens}</small></td>
                      <td>{formatCost(item.total_cost_usd)}</td>
                      <td>{item.duration_ms === undefined ? '—' : `${item.duration_ms} ms`}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          {data.interactions.length === 0 && <p className="admin-empty">No interactions match this filter.</p>}
          <footer>
            <span>{data.total === 0 ? '0' : `${offset + 1}–${Math.min(offset + pageSize, data.total)}`} of {data.total}</span>
            <div>
              <button type="button" disabled={offset === 0 || loading} onClick={() => { setLoading(true); setError(''); setOffset(Math.max(0, offset - pageSize)); }}>Previous</button>
              <button type="button" disabled={offset + pageSize >= data.total || loading} onClick={() => { setLoading(true); setError(''); setOffset(offset + pageSize); }}>Next</button>
            </div>
          </footer>
        </section>
      )}
    </main>
  );
}
