'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { LAB_PLUS_FEATURE, labHref, type LabListItem, useLabConsent, useLabs } from '@/entities/lab';
import { usePlusLocked } from '@/entities/plus';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  Card,
  HeaderButton,
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  type Tone,
} from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';

const HERO_TAGS: readonly { key: 'blood' | 'hormone' | 'thyroid' | 'urine'; tone: Tone }[] = [
  { key: 'blood', tone: 'brand' },
  { key: 'hormone', tone: 'bloom' },
  { key: 'thyroid', tone: 'data' },
  { key: 'urine', tone: 'warm' },
];
const PREVIEW = 4;

type T = ReturnType<typeof useTranslations<'labs'>>;

function statusPill(lab: LabListItem, t: T, locale: Locale): { tone: Tone; label: string } {
  switch (lab.status) {
    case 'ready':
      return lab.allNormal
        ? { tone: 'data', label: t('intro.allNormal') }
        : lab.attentionCount > 0
          ? { tone: 'danger', label: t('intro.outOfRange', { n: formatNumber(lab.attentionCount, locale) }) }
          : { tone: 'neutral', label: t('intro.noRange') };
    case 'needs_review':
      return { tone: 'warm', label: t('intro.needsReview') };
    case 'failed':
      return { tone: 'danger', label: t('intro.failed') };
    default:
      return { tone: 'brand', label: t('intro.processing') };
  }
}

function Shell({ header, children, footer }: { header: ReactNode; children: ReactNode; footer?: ReactNode }) {
  return (
    <div className="view lab-screen">
      <SkyLayer />
      <div className="scroll lab-scroll">
        {header}
        <div className="lab-body">{children}</div>
        {footer}
      </div>
    </div>
  );
}

/**
 * `/labs` (nbl_Lab_Intro): what the lab analysis does, the previous analyses
 * (status per lab, tap → processing / verify / result) and «بارگذاری آزمایش
 * جدید» — through the Plus gate (`plus.lab_ai`) and the consent sheet when the
 * versioned consent is missing or outdated.
 */
export function LabIntroPage() {
  const t = useTranslations('labs');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const labs = useLabs();
  const consent = useLabConsent();
  const locked = usePlusLocked(LAB_PLUS_FEATURE);
  const [all, setAll] = useState(false);

  const startUpload = () => {
    if (locked) router.push('/plus');
    else if (consent.data && !consent.data.needsConsent) router.push('/labs/new');
    else openSheet('lab-consent', 'upload');
  };

  const header = (
    <ScreenHeader
      title={t('intro.title')}
      onBack={() => router.push('/analysis/labs')}
      backLabel={t('common.back')}
      action={<HeaderButton icon="info" label={t('intro.about')} onClick={() => openSheet('lab-consent')} />}
    />
  );
  const footer = (
    <div className="lab-footer">
      <PrimaryButton icon="export" onClick={startUpload}>
        {t('intro.upload')}
        {locked ? <PlusBadge label={t('intro.plus')} className="lab-cta-plus" /> : null}
      </PrimaryButton>
    </div>
  );

  const list = labs.data?.labs ?? [];
  const shown = all ? list : list.slice(0, PREVIEW);
  return (
    <Shell header={header} footer={footer}>
      <Card className="lab-hero" variant="hero">
        <span className="lab-hero-disc" aria-hidden>
          <Icon name="flask" size={34} strokeWidth={1.8} />
        </span>
        <h2 className="lab-hero-title">{t('intro.heroTitle')}</h2>
        <p className="lab-hero-body">{t('intro.heroBody')}</p>
        <ul className="lab-hero-tags" aria-label={t('intro.tagsLabel')}>
          {HERO_TAGS.map((tag) => (
            <li key={tag.key}>
              <StatusPill tone={tag.tone}>{t(`intro.tags.${tag.key}`)}</StatusPill>
            </li>
          ))}
        </ul>
      </Card>

      {!mounted || labs.isPending ? (
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="card" />
        </SkeletonGroup>
      ) : labs.isError ? (
        <Card className="lab-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="lab-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} loading={labs.isFetching} onClick={() => void labs.refetch()}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      ) : list.length ? (
        <Card as="section" className="lab-history" aria-labelledby="lab-history-title">
          <div className="lab-card-head">
            <h2 id="lab-history-title" className="lab-card-title">
              {t('intro.history')}
            </h2>
            {list.length > PREVIEW ? (
              <button type="button" className="lab-link" aria-expanded={all} onClick={() => setAll((v) => !v)}>
                {all ? t('intro.less') : t('intro.all')}
              </button>
            ) : null}
          </div>
          <ul className="lab-rows">
            {shown.map((lab) => {
              const pill = statusPill(lab, t, locale);
              return (
                <li key={lab.id}>
                  <button type="button" className="lab-row" onClick={() => router.push(labHref(lab.id, lab.status))}>
                    <IconCircle icon="flask" tone={lab.status === 'ready' && lab.allNormal ? 'data' : 'brand'} />
                    <span className="lab-row-text">
                      <span className="lab-row-title">{lab.title}</span>
                      <span className="lab-row-sub">
                        {lab.markerCount > 0
                          ? t('intro.rowSub', {
                              date: formatLongDate(fromApiDate(lab.date), locale),
                              n: formatNumber(lab.markerCount, locale),
                            })
                          : formatLongDate(fromApiDate(lab.date), locale)}
                      </span>
                    </span>
                    <StatusPill tone={pill.tone} className="lab-row-pill">
                      {pill.label}
                    </StatusPill>
                  </button>
                </li>
              );
            })}
          </ul>
        </Card>
      ) : (
        <Card className="lab-empty">
          <p className="lab-empty-title">{t('intro.emptyTitle')}</p>
          <p className="lab-empty-body">{t('intro.emptyBody')}</p>
        </Card>
      )}

      <InfoNote>{t('common.disclaimer')}</InfoNote>
    </Shell>
  );
}
