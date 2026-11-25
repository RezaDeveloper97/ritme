'use client';

import { useTranslations } from 'next-intl';

import { useBackLayer } from '@/shared/back-stack';

import { type StepUpController } from '../model/useStepUp';
import { StepUpPanel } from './StepUpPanel';
import '@/shared/ui/styles/del.css';

/**
 * The step-up dialog in front of the data export (T-M12-06): the same calm
 * bottom card as the delete confirm, holding the code step only.
 */
export function StepUpSheet({ open, onClose, onVerified }: StepUpController) {
  const t = useTranslations('profile.stepUp');
  useBackLayer(open, onClose); // Back closes the dialog, not the screen behind it
  if (!open) return null;
  return (
    <div role="dialog" aria-modal="true" aria-label={t('title')} className="del-backdrop" onClick={onClose}>
      <div className="card del-card" onClick={(event) => event.stopPropagation()}>
        <StepUpPanel body={t('exportBody')} onVerified={onVerified} onCancel={onClose} />
      </div>
    </div>
  );
}
