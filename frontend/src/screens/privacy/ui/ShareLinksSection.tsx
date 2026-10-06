'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type ShareLink, useRevokeShareLink, useShareLinks } from '@/entities/health-record';
import type { Locale } from '@/shared/i18n';
import { formatDayMonth, formatLongDate } from '@/shared/lib/date';
import { ListGroup, ListRow, Skeleton, StatusPill } from '@/shared/ui';

/**
 * «لینک‌های گزارش پزشک» (bloom B-N6-04): the owner's doctor-report share links of the last 30 days — status, opens,
 * revoke. Metadata only (the API never returns the token or the report again). Hidden while there is none.
 */
export function ShareLinksSection() {
  const t = useTranslations('recordExport.links');
  const loc = useLocale() as Locale;
  const query = useShareLinks();
  const revoke = useRevokeShareLink();

  if (query.isPending) {
    return (
      <section className="prv-sec" aria-busy="true" aria-label={t('title')}>
        <Skeleton shape="block" />
      </section>
    );
  }
  if (query.isError) {
    return (
      <section className="prv-sec" aria-labelledby="prv-g-links">
        <h2 id="prv-g-links" className="prv-label">
          {t('title')}
        </h2>
        <p className="prv-error" role="alert">
          {t('error')}
        </p>
      </section>
    );
  }
  if (query.data.items.length === 0) return null;

  const status = (l: ShareLink) =>
    l.status === 'active' ? (
      <StatusPill tone="success">{t('active', { date: formatDayMonth(new Date(l.expiresAt), loc) })}</StatusPill>
    ) : (
      <StatusPill tone="neutral">{t(l.status)}</StatusPill>
    );

  return (
    <section className="prv-sec" aria-labelledby="prv-g-links">
      <h2 id="prv-g-links" className="prv-label">
        {t('title')}
      </h2>
      <ListGroup className="prv-list">
        {query.data.items.map((l) => {
          const created = formatLongDate(new Date(l.createdAt), loc);
          return (
            <div key={l.id} className="prv-link-row">
              <ListRow
                icon="fileDoc"
                iconTone={l.status === 'active' ? 'brand' : 'neutral'}
                title={t('created', { date: created })}
                description={t('views', { count: l.views })}
                trailing={status(l)}
              />
              {l.status === 'active' ? (
                <button
                  type="button"
                  className="prv-link-revoke"
                  disabled={revoke.isPending}
                  aria-label={t('revokeLabel', { date: created })}
                  onClick={() => revoke.mutate(l.id)}
                >
                  {revoke.isPending && revoke.variables === l.id ? t('revoking') : t('revoke')}
                </button>
              ) : null}
            </div>
          );
        })}
      </ListGroup>
      <p className="prv-note">{t('hint')}</p>
      {revoke.isError ? (
        <p className="prv-error" role="alert">
          {t('revokeError')}
        </p>
      ) : null}
    </section>
  );
}
