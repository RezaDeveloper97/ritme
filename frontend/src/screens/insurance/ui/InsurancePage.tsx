'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useInsurancePolicies, useInsurancePolicy, useInsurancePolicyById } from '@/entities/insurance'; // CB-INS-03 (+ policy history)

import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Accordion, EmptyState, Icon, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { useInsurancePage } from '../api/insurance';
import { splitInsurance } from '../model/insurance';
import { InsuranceHome } from './InsuranceHome';
import { PolicySetup } from './PolicySetup';

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
              <Accordion key={q.id} title={q.heading} className="ins-q" pilot={pilot(`insurance-info.faq:${q.id}`)}>
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

type View = 'home' | 'edit' | 'info';

/**
 * Insurance at `/insurance`. B-N10-06 owns the read-only info (coverage, claim steps, documents checklist, FAQ, insurer
 * link — admin content, `info_sections` group `insurance`); CB-INS-03 extends this same screen with her own tracker:
 * no policy → the setup form (+ a link to the info), a policy → the home (policy card, remaining coverage, claims), the
 * info staying one tap away. The info view collects and sends nothing personal; neither does the tracker (self-tracking
 * only — nothing is sent to an insurer).
 */
export function InsurancePage() {
  const t = useTranslations('me.insurance');
  const ti = useTranslations('insurance');
  const router = useRouter();
  const policy = useInsurancePolicy();
  const [view, setView] = useState<View | null>(null);
  // Policy history: null = the current policy (GET /policy); another id = that policy of her history.
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const currentState = policy.data ?? null;
  const hasPolicy = Boolean(currentState?.policy);
  const history = useInsurancePolicies(hasPolicy);
  const picked = useInsurancePolicyById(selectedId);
  const state = selectedId !== null && picked.data?.policy ? picked.data : currentState;
  const select = (id: number) => setSelectedId(id === currentState?.policy?.id ? null : id);
  // Until the policy answers (or if the tracker API fails) the page behaves exactly as B-N10-06 did.
  const trackerReady = policy.isSuccess;
  const current: View = !trackerReady ? 'info' : view ?? (hasPolicy ? 'home' : 'edit');
  const showNav = trackerReady && current === 'home';

  let title = t('title');
  let onBack = () => router.push('/services');
  let backLabel = t('back');
  if (trackerReady && current === 'home') {
    title = ti('title');
  } else if (trackerReady && current === 'edit') {
    title = hasPolicy ? ti('setup.editTitle') : ti('setup.title');
    onBack = hasPolicy ? () => setView('home') : () => router.push('/services');
  } else if (trackerReady && current === 'info') {
    title = ti('home.howTitle');
    onBack = () => setView(hasPolicy ? 'home' : 'edit');
    backLabel = ti('home.howBack');
  }

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={title} onBack={onBack} backLabel={backLabel} />
        {!trackerReady && policy.isPending ? (
          <SkeletonGroup label={ti('loading')} className="ins-skel">
            <Skeleton shape="card" />
            <Skeleton shape="card" />
          </SkeletonGroup>
        ) : current === 'home' && state ? (
          <InsuranceHome
            state={state}
            history={history.data ?? null}
            onSelect={select}
            onRenewed={(id) => {
              select(id);
              setView('edit');
            }}
            onEdit={() => setView('edit')}
            onShowInfo={() => setView('info')}
          />
        ) : current === 'edit' ? (
          <PolicySetup key={state?.policy?.id ?? 'new'} policy={state?.policy ?? null} onDone={() => setView('home')} onShowInfo={() => setView('info')} />
        ) : (
          <InsuranceBody />
        )}
      </div>
      {showNav ? <BottomNav /> : null}
    </div>
  );
}
