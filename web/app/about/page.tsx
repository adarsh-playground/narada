/* eslint-disable @next/next/no-html-link-for-pages */

import { PrimaryNavigation } from '../components/primary-navigation';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'About & Credits · Narada',
  description: 'About Narada and the sources and licenses behind its Sanskrit audio.',
};

export default function AboutPage() {
  return (
    <main className="site-shell">
      <header className="topbar">
        <a className="brand" href="/" aria-label="Narada home">
          <span className="brand-mark" aria-hidden="true">न</span>
          <span>Narada</span>
        </a>
        <PrimaryNavigation current="/about" />
      </header>

      <article className="about-page">
        <p className="eyebrow">About Narada</p>
        <h1>Wisdom, one verse at a time.</h1>
        <p className="about-intro">Narada brings together the Bhagavad Gita&apos;s Sanskrit verses, translations, commentary, and recitations in a calm space for reading and reflection.</p>

        <section className="credits-card" aria-labelledby="credits-heading" id="credits">
          <h2 id="credits-heading">Open Source Credits &amp; Licenses</h2>
          <h3>Bhagavad Gita Sanskrit audio</h3>
          <p>The Sanskrit recitations are sourced from <a href="https://huggingface.co/datasets/JDhruv14/Bhagavad-Gita_Audio">Bhagavad-Gita_Audio</a>, created by <strong>Dhruv Jaradi (JDhruv14)</strong>, 2025.</p>
          <p>The dataset is licensed under the <a href="https://www.apache.org/licenses/LICENSE-2.0.txt">Apache License 2.0 (complete license text)</a>.</p>
          <p>Narada converted the original WAV audio recordings to MP3 for streaming.</p>
          <p className="credits-notice">The source dataset&apos;s published files and dataset card do not include a separate copyright statement or NOTICE file. Creator attribution is retained above.</p>
        </section>
      </article>

      <footer className="home-footer about-footer">
        <p>Continue exploring</p>
        <a href="/read">Read the Gita <span aria-hidden="true">→</span></a>
      </footer>
    </main>
  );
}
