'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { labHref, useLabConsent, useLabs } from '@/entities/lab';
import { AiGate, useAiGate } from '@/features/ai-gate';
import { LabUploadForm, limitsFrom } from '@/features/upload-lab';
import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

/**
 * `/labs/new` (nbl_Lab_Upload): the upload form (features/upload-lab) with the
 * server's limits. Without Plus the form sits behind the Plus lock; without the
 * versioned AI consent the consent sheet opens first (and again on a 403).
 * The 202 answer routes to processing / verify / the failed state.
 */
export function LabUploadPage() {
  const t = useTranslations('labs');
  const router = useRouter();
  const mounted = useMounted();
  const labs = useLabs();
  const consent = useLabConsent();
  const gate = useAiGate('lab_analysis');
  const asked = useRef(false);

  useEffect(() => {
    if (!asked.current && gate.canUse && consent.data?.needsConsent) {
      asked.current = true;
      openSheet('lab-consent');
    }
  }, [consent.data, gate.canUse]);

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
        ) : (
          <AiGate feature="lab_analysis" className="lab-up-gate">
            <LabUploadForm
              limits={limitsFrom(labs.data?.limits)}
              onUploaded={(lab) => router.replace(labHref(lab.id, lab.status))}
              onConsentRequired={() => openSheet('lab-consent')}
              ai={gate}
            />
          </AiGate>
        )}
      </div>
    </div>
  );
}
