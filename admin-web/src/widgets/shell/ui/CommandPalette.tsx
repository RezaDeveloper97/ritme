'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useRef, useState } from 'react';

import { Icon } from '@/shared/ui';

import type { NavGroup } from '../model/nav';

/**
 * ⌘K / Ctrl+K palette: type to filter every nav item the role may open, arrows
 * move, Enter navigates, Escape closes (native <dialog> keeps focus inside).
 */
export function CommandPalette({
  groups,
  open,
  onClose,
}: {
  groups: readonly NavGroup[];
  open: boolean;
  onClose: () => void;
}) {
  const t = useTranslations('nav');
  const ts = useTranslations('shell');
  const router = useRouter();
  const ref = useRef<HTMLDialogElement>(null);
  const [query, setQuery] = useState('');
  const [index, setIndex] = useState(0);

  const entries = useMemo(
    () =>
      groups.flatMap((g) =>
        g.items
          .filter((i) => i.ready)
          .map((i) => ({ key: i.key, href: i.href, icon: i.icon, label: t(i.key as 'dashboard'), group: t(g.key as 'groupOverview') })),
      ),
    [groups, t],
  );
  const q = query.trim().toLowerCase();
  const results = q ? entries.filter((e) => `${e.label} ${e.group} ${e.href}`.toLowerCase().includes(q)) : entries;

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) {
      setQuery('');
      setIndex(0);
      d.showModal();
    } else if (!open && d.open) d.close();
  }, [open]);

  const go = (href: string) => {
    onClose();
    router.push(href);
  };
  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setIndex((i) => Math.min(i + 1, results.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter' && results[index]) {
      e.preventDefault();
      go(results[index].href);
    }
  };
  const activeId = results[index] ? `cmd-opt-${results[index].key}` : undefined;

  return (
    <dialog ref={ref} className="dialog palette" aria-label={ts('palette')} onClose={onClose} onClick={(e) => e.target === ref.current && onClose()}>
      <div className="palette-search">
        <Icon name="search" />
        <input
          autoFocus
          type="text"
          role="combobox"
          aria-expanded="true"
          aria-controls="cmd-list"
          aria-activedescendant={activeId}
          aria-label={ts('searchPlaceholder')}
          placeholder={ts('searchPlaceholder')}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setIndex(0);
          }}
          onKeyDown={onKey}
        />
      </div>
      <ul id="cmd-list" role="listbox" aria-label={ts('sections')} className="palette-list">
        {results.length === 0 && <li className="palette-empty">{ts('noResults')}</li>}
        {results.map((r, i) => (
          <li
            key={r.key}
            id={`cmd-opt-${r.key}`}
            role="option"
            aria-selected={i === index}
            className="palette-item"
            onMouseEnter={() => setIndex(i)}
            onClick={() => go(r.href)}
          >
            <Icon name={r.icon} />
            <span className="truncate">{r.label}</span>
            <span className="palette-group">{r.group}</span>
          </li>
        ))}
      </ul>
    </dialog>
  );
}
