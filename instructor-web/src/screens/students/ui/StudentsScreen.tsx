'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useState } from 'react';

import { GrantRow, GrantStatusPill, RevokeButton, phoneLabel, targetOf, useStudents, type GrantStatus as Status } from '@/entities/student';
import { cn, toPersianDigits } from '@/shared/lib';
import { Icon, QueryError, Skeleton } from '@/shared/ui';

/** Latin digits + lower-case so «۰۹۱۲» and «0912» find each other. */
function norm(v: string): string {
  return v
    .replace(/[۰-۹]/g, (d) => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d)))
    .replace(/[٠-٩]/g, (d) => String('٠١٢٣٤٥٦٧٨٩'.indexOf(d)))
    .toLowerCase()
    .trim();
}

const FILTERS: Array<Status | 'all'> = ['all', 'active', 'pending', 'expired'];

/** /students — nbl_Ins_Students: everyone given access, filter by status, revoke, add. */
export function StudentsScreen() {
  const t = useTranslations('students');
  const [filter, setFilter] = useState<Status | 'all'>('all');
  const [query, setQuery] = useState('');
  const { data, isPending, isError, error, refetch } = useStudents(filter);
  const all = useStudents('all');
  const counts = all.data?.counts;
  const needle = norm(query);
  const total = counts ? counts.active + counts.pending + counts.expired : null;

  const visible = (data?.students ?? []).filter((s) => {
    if (!needle) return true;
    const target = targetOf(s);
    return norm([s.phone, s.student_name ?? '', target?.name ?? ''].join(' ')).includes(needle);
  });

  return (
    <div className="page">
      <header className="page-head">
        <div className="page-head__text">
          <h1 className="page-head__title">{t('title')}</h1>
          {total !== null ? <p className="page-head__sub">{t('summary', { count: toPersianDigits(total) })}</p> : null}
        </div>
        <Link href="/students/add" className="icon-btn" aria-label={t('add')} title={t('add')}>
          <Icon name="userPlus" />
        </Link>
      </header>

      <input
        type="search"
        className="field__control"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t('search')}
        aria-label={t('search')}
      />

      <div className="segmented" role="tablist" aria-label={t('filtersLabel')}>
        {FILTERS.map((f) => (
          <button
            key={f}
            type="button"
            role="tab"
            aria-selected={filter === f}
            className={cn('segmented__item', filter === f && 'is-active')}
            onClick={() => setFilter(f)}
          >
            {t(`filter.${f}`)}
          </button>
        ))}
      </div>

      {isPending ? (
        <Skeleton className="skeleton--card" />
      ) : isError ? (
        <QueryError error={error} onRetry={() => void refetch()} />
      ) : (
        <section className="panel-card">
          {visible.length === 0 ? (
            <p className="panel-card__empty">{needle ? t('emptySearch') : filter === 'all' ? t('empty') : t('emptyFilter')}</p>
          ) : (
            <ul className="row-list">
              {visible.map((s) => {
                const target = targetOf(s);
                const meta = [
                  s.student_name ? phoneLabel(s.phone) : null,
                  target ? (target.kind === 'group' ? t('inGroup', { name: target.name }) : target.name) : null,
                ].filter(Boolean);
                return (
                  <GrantRow
                    key={s.id}
                    grant={s}
                    meta={meta.join(' · ') || phoneLabel(s.phone)}
                    trailing={
                      <>
                        <GrantStatusPill grant={s} />
                        <RevokeButton grantId={s.id} />
                      </>
                    }
                  />
                );
              })}
            </ul>
          )}
        </section>
      )}
    </div>
  );
}
