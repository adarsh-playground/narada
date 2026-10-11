'use client';

import { useEffect, useRef } from 'react';

const gitaLinks = [
  { href: '/', label: 'Today’s verse' },
  { href: '/read', label: 'Read the Gita' },
  { href: '/listen', label: 'Listen to the Gita' },
];

export function PrimaryNavigation({ current }: { current: string }) {
  const menu = useRef<HTMLDetailsElement>(null);
  const trigger = useRef<HTMLElement>(null);

  useEffect(() => {
    function outside(event: PointerEvent) {
      if (event.target instanceof Node && !menu.current?.contains(event.target) && menu.current) {
        menu.current.open = false;
      }
    }
    function escape(event: KeyboardEvent) {
      if (event.key === 'Escape' && menu.current?.open) {
        menu.current.open = false;
        trigger.current?.focus();
      }
    }
    document.addEventListener('pointerdown', outside);
    document.addEventListener('keydown', escape);
    return () => {
      document.removeEventListener('pointerdown', outside);
      document.removeEventListener('keydown', escape);
    };
  }, []);

  return (
    <nav className="primary-navigation" aria-label="Primary navigation">
      <details className="gita-menu" ref={menu}>
        <summary ref={trigger} className={gitaLinks.some(link => link.href === current) ? 'current' : ''}>
          Gita <span className="gita-menu-chevron" aria-hidden="true">⌄</span>
        </summary>
        <div className="gita-menu-links">
          {gitaLinks.map(link => (
            <a key={link.href} href={link.href} className={current === link.href ? 'current' : ''} aria-current={current === link.href ? 'page' : undefined}>
              {link.label}
            </a>
          ))}
        </div>
      </details>
      <a href="/ask" className={`nav-ask ${current === '/ask' ? 'current' : ''}`} aria-current={current === '/ask' ? 'page' : undefined}>Ask <span aria-hidden="true">↗</span></a>
    </nav>
  );
}
