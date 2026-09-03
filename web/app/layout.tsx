import type { Metadata } from 'next';
import { Cormorant_Garamond, Manrope, Noto_Sans_Devanagari } from 'next/font/google';
import './globals.css';

const display = Cormorant_Garamond({
  variable: '--font-display',
  subsets: ['latin'],
  weight: ['500', '600'],
});

const sans = Manrope({ variable: '--font-sans', subsets: ['latin'] });
const devanagari = Noto_Sans_Devanagari({
  variable: '--font-devanagari',
  subsets: ['devanagari'],
});

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000'),
  title: 'Narada · Wisdom from the Bhagavad Gita',
  description: 'A calm, source-grounded way to read and explore the Bhagavad Gita.',
  openGraph: {
    title: 'Narada',
    description: 'Wisdom from the Bhagavad Gita',
    images: ['/og.png'],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Narada',
    description: 'Wisdom from the Bhagavad Gita',
    images: ['/og.png'],
  },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className={`${display.variable} ${sans.variable} ${devanagari.variable}`}>
        {children}
      </body>
    </html>
  );
}
