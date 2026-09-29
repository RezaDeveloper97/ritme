'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { PageMeta } from '@/shared/api';
import { formatNumber } from '@/shared/lib';

import { Button } from './Button';
import { Icon } from './Icon';

/** Previous / next pager for a server list's `meta` (admin-api.md §5). */
export function Pagination({ meta, onPage }: { meta: PageMeta | undefined; onPage: (page: number) => void }) {
  const t = useTranslations('table');
  const locale = useLocale();
  if (!meta) return null;
  const last = Math.max(meta.last_page, 1);
  return (
    <nav className="flex flex-wrap items-center gap-3 border-t border-line px-4 py-3 text-ink-3" aria-label={t('page', { page: formatNumber(meta.current_page, locale), last: formatNumber(last, locale) })}>
      <span className="text-[13px]">{t('total', { total: meta.total })}</span>
      <div className="flex-1" />
      <span className="text-[13px] tabular-nums">
        {t('page', { page: formatNumber(meta.current_page, locale), last: formatNumber(last, locale) })}
      </span>
      <Button size="sm" disabled={meta.current_page <= 1} onClick={() => onPage(meta.current_page - 1)}>
        <Icon name="chevronRight" size={16} className="-scale-x-100 rtl:scale-x-100" />
        {t('prev')}
      </Button>
      <Button size="sm" disabled={meta.current_page >= last} onClick={() => onPage(meta.current_page + 1)}>
        {t('next')}
        <Icon name="chevronRight" size={16} className="rtl:-scale-x-100" />
      </Button>
    </nav>
  );
}
