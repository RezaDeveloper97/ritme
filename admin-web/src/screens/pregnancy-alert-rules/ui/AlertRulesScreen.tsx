'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useContentLanguages } from '@/shared/i18n';
import { cn, formatNumber } from '@/shared/lib';
import { Badge, ErrorState, Panel, RowActions, Skeleton } from '@/shared/ui';

import { useAlertRules } from '../api/alert-rules';
import { levelClass } from '../lib/level';
import { useAlertLabels } from './labels';

/** /pregnancy-alert-rules — the 8 pregnancy alert rules (message group `pregnancy_alert`, admin-api.md §13). */
export function AlertRulesScreen() {
  const t = useTranslations('pregnancyAlertRules');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const labels = useAlertLabels();
  const languages = useContentLanguages().data?.languages ?? [];
  const langName = (code: string) => languages.find((l) => l.code === code)?.name ?? code;
  const query = useAlertRules();

  return (
    <div className="flex flex-col gap-4">
      <Panel title={t('title')} bodyClassName="">
        <p className="panel-note">{t('intro')}</p>
        {query.error && !query.data ? (
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        ) : !query.data ? (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <caption className="sr-only">{t('title')}</caption>
              <thead>
                <tr>
                  <th scope="col">{t('rule')}</th>
                  <th scope="col">{t('level')}</th>
                  <th scope="col">{t('windowDays')}</th>
                  <th scope="col">{tc('status')}</th>
                  <th scope="col">{t('languages')}</th>
                  <th scope="col">
                    <span className="sr-only">{tc('actions')}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {query.data.items.map((r) => (
                  <tr key={r.key} data-clickable="true" onClick={() => router.push(`/pregnancy-alert-rules/${r.key}`)}>
                    <td className="cell-wrap">
                      <span className="flex flex-col">
                        <strong>{labels.rule(r.key)}</strong>
                        <span className="text-xs text-muted">{r.title ?? ''}</span>
                      </span>
                    </td>
                    <td>
                      {r.level ? (
                        <span className={cn('rounded-full px-2 py-0.5 text-xs font-semibold', levelClass(r.level, 'chip'))}>
                          {labels.level(r.level)}
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="cell-num">{r.window_days === null ? '—' : formatNumber(r.window_days, locale)}</td>
                    <td>
                      {!r.configured ? (
                        <Badge tone="amber">{t('notConfigured')}</Badge>
                      ) : r.enabled ? (
                        <Badge tone="green">{t('enabled')}</Badge>
                      ) : (
                        <Badge tone="red">{t('disabled')}</Badge>
                      )}
                    </td>
                    <td>
                      <span className="flex flex-wrap gap-1">
                        {r.missing_locales.map((code) => (
                          <Badge key={code} tone="amber">
                            {t('missingLocale', { locale: langName(code) })}
                          </Badge>
                        ))}
                        {r.missing_locales.length === 0 ? <Badge tone="green">{t('allLocales')}</Badge> : null}
                      </span>
                    </td>
                    <td className="cell-actions" onClick={(e) => e.stopPropagation()}>
                      <RowActions>
                        <Link href={`/pregnancy-alert-rules/${r.key}`} className="btn btn-sm">
                          {tc('edit')}
                        </Link>
                      </RowActions>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
      {query.data?.legend && query.data.legend.rows.length > 0 ? (
        <Panel title={t('legend')}>
          <p className="field-hint mt-0">{t('legendHint')}</p>
          <div className="flex flex-wrap gap-2">
            {query.data.legend.rows.map((row) => (
              <Link key={row.id} href={`/messages/${row.id}`} className="btn btn-sm">
                {t('editLegend', { locale: langName(row.locale) })}
              </Link>
            ))}
          </div>
        </Panel>
      ) : null}
    </div>
  );
}
