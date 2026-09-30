'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import {
  EmptyState,
  HeaderButton,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { useInfoPage } from '../api/info-page';
import { splitLegal } from '../model/info-page';

type Tab = 'terms' | 'privacy';

function LegalBody({ tab }: { tab: Tab }) {
  const t = useTranslations('me.legal');
  const loc = useLocale() as Locale;
  const query = useInfoPage(tab, loc);

  if (query.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="lgl-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }
  if (query.isError || !query.data) {
    return (
      <EmptyState
        icon="note"
        title={t('error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  }
  const { summary, sections } = splitLegal(query.data);
  if (summary.length === 0 && sections.length === 0) return <EmptyState icon="note" title={t('empty')} />;

  const anchor = (id: number) => `lgl-${tab}-${id}`;
  const jump = (id: number) => {
    const el = document.getElementById(anchor(id));
    el?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    el?.focus({ preventScroll: true });
  };
  const summaryBox = query.data.sections.find((s) => s.key === 'summary');

  return (
    <>
      {query.data.updatedAt ? (
        <p className="lgl-updated">{t('updated', { date: formatLongDate(fromApiDate(query.data.updatedAt), loc) })}</p>
      ) : null}
      {summary.length > 0 ? (
        <section className="nb-card lgl-summary" aria-labelledby="lgl-sum-h">
          <h2 id="lgl-sum-h" className="lgl-card-h">
            {summaryBox?.heading}
          </h2>
          <ul className="lgl-sum-list">
            {summary.map((line) => (
              <li key={line} className="lgl-sum-item">
                <Icon name="check" size={16} className="lgl-sum-icon" />
                <span>{line}</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {sections.length > 0 ? (
        <nav className="nb-card lgl-toc" aria-labelledby="lgl-toc-h">
          <h2 id="lgl-toc-h" className="lgl-card-h">
            {t('toc')}
          </h2>
          <ol className="lgl-toc-list">
            {sections.map((s, i) => (
              <li key={s.id}>
                <a
                  className="lgl-toc-link"
                  href={`#${anchor(s.id)}`}
                  onClick={(e) => {
                    e.preventDefault();
                    jump(s.id);
                  }}
                >
                  {formatNumber(i + 1, loc)}. {s.heading}
                </a>
              </li>
            ))}
          </ol>
        </nav>
      ) : null}
      {sections.map((s, i) => (
        <section key={s.id} id={anchor(s.id)} tabIndex={-1} className="lgl-sec" aria-labelledby={`${anchor(s.id)}-h`}>
          <h2 id={`${anchor(s.id)}-h`} className="lgl-sec-h">
            {formatNumber(i + 1, loc)}. {s.heading}
          </h2>
          <p className="lgl-sec-body">{s.body}</p>
          {s.linkUrl && s.linkLabel ? (
            <a
              className="lgl-sec-link"
              href={s.linkUrl}
              {...(s.linkUrl.startsWith('http') ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
            >
              {s.linkLabel}
            </a>
          ) : null}
        </section>
      ))}
    </>
  );
}

/**
 * Terms & privacy (B-N1-12, `nbl_Me_Legal` / `nbd_Me_Legal`) at
 * `/profile/legal?tab=terms|privacy`: tabs, «خلاصه در ۳ خط» (the `summary`
 * box), a contents list with in-page anchors and the numbered sections — all
 * admin content (`info_sections` groups `terms` / `privacy`).
 */
export function LegalPage({ initialTab }: { initialTab?: string }) {
  const t = useTranslations('me.legal');
  const router = useRouter();
  const [tab, setTab] = useState<Tab>(initialTab === 'terms' ? 'terms' : 'privacy');

  const share = async () => {
    const url = window.location.href;
    try {
      if (navigator.share) await navigator.share({ title: t('title'), url });
      else await navigator.clipboard.writeText(url);
    } catch {
      // A dismissed share sheet is not an error.
    }
  };

  return (
    <div className="view lgl-page">
      <SkyLayer />
      <div className="scroll lgl-scroll">
        <ScreenHeader
          title={t('title')}
          onBack={() => router.push('/profile/about')}
          backLabel={t('back')}
          action={<HeaderButton label={t('share')} icon="export" onClick={() => void share()} />}
        />
        <SegmentedTabs
          label={t('tabs.label')}
          tabs={[
            { value: 'terms', label: t('tabs.terms') },
            { value: 'privacy', label: t('tabs.privacy') },
          ]}
          value={tab}
          onChange={(v) => {
            setTab(v);
            router.replace(`/profile/legal?tab=${v}`);
          }}
          className="lgl-tabs"
        />
        <LegalBody key={tab} tab={tab} />
      </div>
    </div>
  );
}
