'use client';

import { useTranslations } from 'next-intl';

import { Button } from '@/shared/ui';

import { hintFor } from '../lib/hints';

/**
 * The group's meta conventions (catalog.md §4) and, when known, an example (the form can insert it
 * into an empty meta field). `compact` (the list) shows the text only.
 */
export function MetaHint({
  group,
  compact = false,
  onInsert,
}: {
  group: string;
  compact?: boolean;
  onInsert?: (json: string) => void;
}) {
  const t = useTranslations('catalog');
  const hint = hintFor(group);
  const example = hint.example && !compact ? JSON.stringify(hint.example, null, 2) : null;
  return (
    <aside className="rounded-xl border border-line bg-surface-2 p-3 text-[13px] text-ink" aria-label={t('metaHintTitle')}>
      <p className="font-bold text-ink">{t('metaHintTitle')}</p>
      <p className="mt-1">{t(`hint.${hint.key}` as 'hint.generic')}</p>
      <p className="mt-1 text-ink-3">{t('hint.conventions')}</p>
      {example ? (
        <>
          <pre dir="ltr" className="mt-2 overflow-x-auto rounded-lg border border-line bg-surface p-2 text-start text-xs">
            {example}
          </pre>
          {onInsert ? (
            <Button size="sm" variant="ghost" className="mt-2" onClick={() => onInsert(example)}>
              {t('insertExample')}
            </Button>
          ) : null}
        </>
      ) : null}
    </aside>
  );
}
