'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { LAB_PLUS_FEATURE, labHref, useLabConsent, useLabs } from '@/entities/lab';
import { findEntitlement, labQuotaOf, PlusFeatureGate, usePlusLocked, usePlusStatus } from '@/entities/plus';
import { LabUploadForm, limitsFrom } from '@/features/upload-lab';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { EmptyState, InfoNote, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { labPaywalled, labQuotaNote } from '../model/quota';

/**
 * `/labs/new` (nbl_Lab_Upload): the upload form (features/upload-lab) with the
 * server's limits. Without Plus the form sits behind the Plus lock; without the
 * versioned AI consent the consent sheet opens first (and again on a 403).
 * The 202 answer routes to processing / verify / the failed state.
 */
export function LabUploadPage() {
  const t = useTranslations('labs');
  const tp = useTranslations('plus.limits.lab');
  const loc = useLocale() as Locale;
  const plus = usePlusStatus();
  const router = useRouter();
  const mounted = useMounted();
  const labs = useLabs();
  const consent = useLabConsent();
  const locked = usePlusLocked(LAB_PLUS_FEATURE);
  const asked = useRef(false);
  const note = labQuotaNote(labQuotaOf(findEntitlement(plus.data, LAB_PLUS_FEATURE)), plus.data?.isPlus ?? false);
  const paywalled = labPaywalled(note, plus.data?.isPlus ?? false);
  const renews = plus.data?.resetsAt ? formatDayMonth(fromApiDate(plus.data.resetsAt.slice(0, 10)), loc) : '';

  useEffect(() => {
    if (!asked.current && !locked && consent.data?.needsConsent) {
      asked.current = true;
      openSheet('lab-consent');
    }
  }, [consent.data, locked]);

  const header = (
    <ScreenHeader title={t('upload.title')} onBack={() => router.push('/labs')} backLabel={t('common.back')} />
  );
  return (
    <div className="view lab-screen">
      <SkyLayer />
      <div className="scroll lab-scroll">
        {header}
        {!mounted || labs.isPending ? (
          <div className="lab-body">
            <SkeletonGroup label={t('common.loading')}>
              <Skeleton shape="block" />
              <Skeleton shape="card" />
              <Skeleton shape="card" />
            </SkeletonGroup>
          </div>
        ) : paywalled && note ? (
          <div className="lab-body">
            <EmptyState
              icon="crown"
              title={tp('spent', { limit: formatNumber(note.key === 'spent' ? note.limit : 0, loc), date: renews })}
              action={<PrimaryButton block={false} onClick={() => router.push('/plus')}>{tp('spentCta')}</PrimaryButton>}
            />
          </div>
        ) : (
          <PlusFeatureGate feature={LAB_PLUS_FEATURE} className="lab-up-gate">
            {note && note.key !== 'spent' ? (
              <InfoNote>
                {tp(note.key, { remaining: formatNumber(note.remaining, loc), limit: formatNumber(note.limit, loc), date: renews })}
              </InfoNote>
            ) : null}
            <LabUploadForm
              limits={limitsFrom(labs.data?.limits)}
              onUploaded={(lab) => router.replace(labHref(lab.id, lab.status))}
              onConsentRequired={() => openSheet('lab-consent')}
            />
          </PlusFeatureGate>
        )}
      </div>
    </div>
  );
}
