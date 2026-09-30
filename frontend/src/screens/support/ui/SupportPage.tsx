'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import {
  Accordion,
  EmptyState,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SearchField,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { useSupportBoxes } from '../api/support';
import { contactOf, filterFaq } from '../model/support';
import { ReportSheet } from './ReportSheet';

/**
 * Support (B-N1-12, `nbl_Me_Support` / `nbd_Me_Support`) at `/profile/support`:
 * FAQ search, the chat card and «گزارش مشکل» tiles, the FAQ accordion and the
 * phone line. FAQ = admin `help` boxes; chat target and phone = admin
 * `support` boxes (see model/support.ts) — no contact detail lives in code.
 */
export function SupportPage() {
  const t = useTranslations('me.support');
  const router = useRouter();
  const locale = useLocale();
  const faq = useSupportBoxes('help', locale);
  const contact = useSupportBoxes('support', locale);
  const [query, setQuery] = useState('');
  const [reportOpen, setReportOpen] = useState(false);
  const { phone, chat } = contactOf(contact.data ?? []);
  const items = useMemo(() => filterFaq(faq.data ?? [], query), [faq.data, query]);

  const external = (url: string) => (url.startsWith('http') ? { target: '_blank', rel: 'noopener noreferrer' } : {});

  let faqBody;
  if (faq.isPending) {
    faqBody = (
      <SkeletonGroup label={t('loading')} className="sup-skel">
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} shape="block" className="sup-skel-row" />
        ))}
      </SkeletonGroup>
    );
  } else if (faq.isError) {
    faqBody = (
      <EmptyState
        icon="help"
        title={t('error')}
        action={
          <PrimaryButton icon="refresh" loading={faq.isFetching} onClick={() => void faq.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (items.length === 0) {
    faqBody = <p className="sup-empty">{query.trim() ? t('noResults', { query: query.trim() }) : t('empty')}</p>;
  } else {
    faqBody = (
      <div className="sup-faq">
        {items.map((q) => (
          <Accordion key={q.id} title={q.heading} className="sup-q">
            <p className="sup-a">{q.body}</p>
            {q.linkUrl && q.linkLabel ? (
              <a className="sup-a-link" href={q.linkUrl} {...external(q.linkUrl)}>
                {q.linkLabel}
              </a>
            ) : null}
          </Accordion>
        ))}
      </div>
    );
  }

  return (
    <div className="view sup-page">
      <SkyLayer />
      <div className="scroll sup-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/profile')} backLabel={t('back')} />

        <SearchField
          value={query}
          onValueChange={setQuery}
          label={t('search')}
          clearLabel={t('clear')}
          placeholder={t('search')}
        />

        <div className="sup-tiles">
          {chat?.linkUrl ? (
            <a className="sup-tile" href={chat.linkUrl} {...external(chat.linkUrl)}>
              <span className="sup-tile-disc tone-data" aria-hidden>
                <Icon name="chat" size={20} />
              </span>
              <span className="sup-tile-title">{t('chat')}</span>
              <span className="sup-tile-sub">{chat.body}</span>
            </a>
          ) : (
            <div className="sup-tile is-disabled" aria-disabled="true">
              <span className="sup-tile-disc tone-data" aria-hidden>
                <Icon name="chat" size={20} />
              </span>
              <span className="sup-tile-title">{t('chat')}</span>
              <span className="sup-tile-sub">{contact.isPending ? ' ' : t('chatSoon')}</span>
            </div>
          )}
          <button type="button" className="sup-tile" onClick={() => setReportOpen(true)}>
            <span className="sup-tile-disc tone-warm" aria-hidden>
              <Icon name="warning" size={20} />
            </span>
            <span className="sup-tile-title">{t('report')}</span>
            <span className="sup-tile-sub">{t('reportSub')}</span>
          </button>
        </div>

        <section className="sup-sec" aria-labelledby="sup-faq-h">
          <h2 id="sup-faq-h" className="sup-label">
            {t('faq')}
          </h2>
          {faqBody}
        </section>

        {phone?.linkUrl ? (
          <p className="sup-phone">
            {phone.heading}:{' '}
            <a className="sup-phone-num" href={phone.linkUrl}>
              <bdi dir="ltr">{phone.linkLabel}</bdi>
            </a>
            {phone.body ? <span> · {phone.body}</span> : null}
          </p>
        ) : null}
      </div>
      <ReportSheet open={reportOpen} onClose={() => setReportOpen(false)} />
    </div>
  );
}
