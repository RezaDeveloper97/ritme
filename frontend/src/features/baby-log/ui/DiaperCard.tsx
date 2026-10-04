'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, Icon, IconCircle, Skeleton, TileButton } from '@/shared/ui';

import { babyActionError, useDiaperAction, useDiaperDay } from '../api/queries';
import { wallTime } from '../model/live';
import { DIAPER_KINDS, type DiaperKind } from '../model/types';

const ICON: Record<DiaperKind, 'drop' | 'box' | 'sprout'> = { wet: 'drop', dirty: 'box', both: 'sprout' };

/**
 * Diapers on the feeding screen (B-N5-07): one tap logs a change now (wet /
 * dirty / both), today's count and today's changes with delete for a mistake.
 * Read-only for a spouse.
 */
export function DiaperCard({ childId, readOnly }: { childId: number; readOnly: boolean }) {
  const t = useTranslations('babyLog');
  const loc = useLocale() as Locale;
  const day = useDiaperDay(childId);
  const action = useDiaperAction(childId);
  const summary = day.data?.summary;
  const items = day.data?.items ?? [];
  const error = action.isError ? babyActionError(action.error) ?? t('errors.save') : null;

  return (
    <Card as="section" className="bfl-card" id="diapers" aria-labelledby="bfl-diaper-title">
      <div className="bfl-card-head">
        <IconCircle icon="drop" tone="warm" size="sm" />
        <h2 id="bfl-diaper-title" className="bfl-card-title">
          {t('diapers.title')}
        </h2>
      </div>
      {readOnly ? null : (
        <div className="bfl-diaper-tiles">
          {DIAPER_KINDS.map((kind) => (
            <TileButton
              key={kind}
              icon={ICON[kind]}
              tone={kind === 'wet' ? 'data' : kind === 'dirty' ? 'warm' : 'bloom'}
              label={t(`diapers.kinds.${kind}`)}
              layout="compact"
              disabled={action.isPending}
              onClick={() => action.mutate({ kind: 'add', diaper: kind })}
            />
          ))}
        </div>
      )}
      {day.isPending ? (
        <Skeleton shape="line" />
      ) : day.isError ? (
        <p className="bfl-error" role="alert">
          {t('errors.load')}{' '}
          <button type="button" className="bfl-link" onClick={() => void day.refetch()}>
            {t('errors.retry')}
          </button>
        </p>
      ) : (
        <>
          <p className="bfl-card-line" aria-live="polite">
            {summary && summary.count > 0
              ? t('diapers.today', {
                  count: formatNumber(summary.count, loc),
                  wet: formatNumber(summary.wet + summary.both, loc),
                  dirty: formatNumber(summary.dirty + summary.both, loc),
                })
              : t('diapers.none')}
          </p>
          {items.length ? (
            <ul className="bfl-list" aria-label={t('diapers.list')}>
              {items.map((d) => {
                const time = formatNumber(wallTime(d.changedAt), loc);
                const kind = t(`diapers.kinds.${d.kind}`);
                return (
                  <li key={d.id} className="bfl-list-row">
                    <span className="bfl-list-time">{time}</span>
                    <span className="bfl-list-text">{kind}</span>
                    {readOnly ? null : (
                      <button
                        type="button"
                        className="bfl-icon-btn"
                        aria-label={t('diapers.delete', { kind, time })}
                        disabled={action.isPending}
                        onClick={() => action.mutate({ kind: 'delete', diaperId: d.id })}
                      >
                        <Icon name="trash" size={18} />
                      </button>
                    )}
                  </li>
                );
              })}
            </ul>
          ) : null}
        </>
      )}
      {error ? (
        <p className="bfl-error" role="alert">
          {error}
        </p>
      ) : null}
    </Card>
  );
}
