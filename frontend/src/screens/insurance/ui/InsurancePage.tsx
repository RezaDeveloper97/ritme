'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { Accordion, EmptyState, Icon, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { useInsurancePage } from '../api/insurance';
import { splitInsurance } from '../model/insurance';

function InsuranceBody() {
  const t = useTranslations('me.insurance');
  const loc = useLocale() as Locale;
  const query = useInsurancePage(loc);

  if (query.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }
  if (query.isError || !query.data) {
    return (
      <EmptyState
        icon="shield"
        title={t('error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  }
  const page = query.data;
  if (page.sections.length === 0) return <EmptyState icon="shield" title={t('empty')} />;
  const s = splitInsurance(page);
  const link = page.sections.find((b) => b.linkUrl && b.linkLabel) ?? null;

  return (
    <>
      <p className="ins-intro">{t('intro')}</p>
      {s.coverage ? (
        <section className="nb-card ins-card" aria-labelledby="ins-cov-h">
          <h2 id="ins-cov-h" className="ins-h">
            {s.coverage.heading || t('coverage')}
          </h2>
          <p className="ins-body">{s.coverage.body}</p>
        </section>
      ) : null}
      {s.steps.length > 0 ? (
        <section className="nb-card ins-card" aria-labelledby="ins-steps-h">
          <h2 id="ins-steps-h" className="ins-h">
            {s.stepsHeading || t('steps')}
          </h2>
          <ol className="ins-steps">
            {s.steps.map((line, i) => (
              <li key={`${i}-${line}`} className="ins-step">
                <span className="ins-step-n" aria-hidden>
                  {formatNumber(i + 1, loc)}
                </span>
                <span className="ins-step-t">{line}</span>
              </li>
            ))}
          </ol>
        </section>
      ) : null}
      {s.documents.length > 0 ? (
        <section className="nb-card ins-card" aria-labelledby="ins-docs-h">
          <h2 id="ins-docs-h" className="ins-h">
            {s.documentsHeading || t('documents')}
          </h2>
          <ul className="ins-docs">
            {s.documents.map((line, i) => (
              <li key={`${i}-${line}`} className="ins-doc">
                <Icon name="check" size={16} className="ins-doc-icon" />
                <span>{line}</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {s.faq.length > 0 ? (
        <section className="ins-sec" aria-labelledby="ins-faq-h">
          <h2 id="ins-faq-h" className="ins-label">
            {t('faq')}
          </h2>
          <div className="ins-faq">
            {s.faq.map((q) => (
              <Accordion key={q.id} title={q.heading} className="ins-q">
                <p className="ins-body">{q.body}</p>
              </Accordion>
            ))}
          </div>
        </section>
      ) : null}
      {link?.linkUrl ? (
        <a className="ins-link" href={link.linkUrl} target="_blank" rel="noopener noreferrer sponsored">
          <Icon name="globe" size={18} />
          <span>{link.linkLabel ?? t('partnerLink')}</span>
        </a>
      ) : null}
      {s.disclaimer ? <p className="ins-note">{s.disclaimer.body}</p> : null}
      <p className="ins-note">{t('privacy')}</p>
      {page.updatedAt ? (
        <p className="ins-note">{t('updated', { date: formatLongDate(fromApiDate(page.updatedAt), loc) })}</p>
      ) : null}
    </>
  );
}

/**
 * Insurance info (B-N10-06) at `/insurance`: coverage, claim steps, documents checklist, FAQ and an optional
 * insurer link — all admin content (`info_sections` group `insurance`). Collects and sends nothing personal.
 */
export function InsurancePage() {
  const t = useTranslations('me.insurance');
  const router = useRouter();
  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/services')} backLabel={t('back')} />
        <InsuranceBody />
      </div>
    </div>
  );
}
