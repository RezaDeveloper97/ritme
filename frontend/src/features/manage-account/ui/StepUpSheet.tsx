'use client';

import { useTranslations } from 'next-intl';

import { type StepUpController } from '../model/useStepUp';
import { StepUpPanel } from './StepUpPanel';

/**
 * The step-up dialog in front of the data export (T-M12-06): the same calm
 * bottom card as the delete confirm, holding the code step only.
 */
export function StepUpSheet({ open, onClose, onVerified }: StepUpController) {
  const t = useTranslations('profile.stepUp');
  if (!open) return null;
  return (
    <div role="dialog" aria-modal="true" aria-label={t('title')} className="del-backdrop" onClick={onClose}>
      <div className="card del-card" onClick={(event) => event.stopPropagation()}>
        <StepUpPanel body={t('exportBody')} onVerified={onVerified} onCancel={onClose} />
      </div>
    </div>
  );
}
