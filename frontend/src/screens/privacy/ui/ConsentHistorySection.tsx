'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale } from '@/shared/i18n';
import { formatLongDate } from '@/shared/lib/date';
import { ListGroup, ListRow, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { useConsentHistory } from '../api/history';
import './prv.css';

/**
 * «سابقهٔ رضایت‌ها» (T-M10-23, D-174): the user's own consent ledger, newest
 * first — what she accepted or withdrew, where, and when. Read-only: the
 * ledger is append-only on the server.
 */
export function ConsentHistorySection() {
  const t = useTranslations('account.history');
  const loc = useLocale() as Locale;
  const history = useConsentHistory();

  let body;
  if (history.isPending) {
    body = (
      <SkeletonGroup label={t('title')}>
        <Skeleton shape="line" width="medium" />
        <Skeleton shape="line" />
      </SkeletonGroup>
    );
  } else if (history.isError) {
    body = (
      <>
        <p className="prv-error" role="alert">
          {t('error')}
        </p>
        <SecondaryButton icon="refresh" onClick={() => void history.refetch()}>
          {t('retry')}
        </SecondaryButton>
      </>
    );
  } else if (!history.data.length) {
    body = <p className="prv-note">{t('empty')}</p>;
  } else {
    body = (
      <ListGroup className="prv-list">
        {history.data.map((e) => (
          <ListRow
            key={e.id}
            icon={e.action === 'grant' ? 'checkCircle' : 'x'}
            iconTone={e.action === 'grant' ? 'data' : 'neutral'}
            title={e.title ?? t('untitled')}
            description={t('line', {
              action: t(e.action),
              source: t(`sources.${e.source}`),
              date: e.createdAt ? formatLongDate(new Date(e.createdAt), loc) : '',
            })}
          />
        ))}
      </ListGroup>
    );
  }

  return (
    <section className="prv-sec" aria-labelledby="prv-g-history">
      <h2 id="prv-g-history" className="prv-label">
        {t('title')}
      </h2>
      <p className="prv-note">{t('lead')}</p>
      {body}
    </section>
  );
}
