'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { Icon } from './Icon';

/** Debounced (350 ms) search box bound to a list's `q`. */
export function SearchInput({
  value,
  onChange,
  placeholder,
  delay = 350,
}: {
  value: string;
  onChange: (q: string) => void;
  placeholder?: string;
  delay?: number;
}) {
  const t = useTranslations('table');
  const [draft, setDraft] = useState(value);

  // Follow the URL when it changes from outside (back/forward, reset).
  useEffect(() => setDraft(value), [value]);

  useEffect(() => {
    if (draft.trim() === value) return;
    const id = window.setTimeout(() => onChange(draft.trim()), delay);
    return () => window.clearTimeout(id);
  }, [draft, value, onChange, delay]);

  return (
    <label className="relative flex min-w-48 flex-1 items-center sm:max-w-80">
      <span className="sr-only">{t('search')}</span>
      <Icon name="search" size={16} className="pointer-events-none absolute start-3 text-muted" />
      <input
        type="search"
        className="input ps-9"
        value={draft}
        placeholder={placeholder ?? t('searchPlaceholder')}
        onChange={(e) => setDraft(e.target.value)}
      />
    </label>
  );
}
