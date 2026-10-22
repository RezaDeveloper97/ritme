'use client';

import { useTranslations } from 'next-intl';

import { datingResultKey, type PregnancyDating } from '../model/dating-result';

/** The one-line outcome after a scan is saved (D-169); renders nothing when there is nothing to say. */
export function DatingResultLine({ dating }: { dating: PregnancyDating | null | undefined }) {
  const t = useTranslations('pregnancyV2.common.datingResult');
  const key = datingResultKey(dating);
  if (!key) return null;
  return (
    <p role="status" className="pgn-body-text">
      {t(key)}
    </p>
  );
}
