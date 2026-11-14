'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { formatCredits } from '@/entities/ai-credit';
import type { Locale } from '@/shared/i18n';
import { Card, IconCircle, SecondaryButton } from '@/shared/ui';

import { useAiGate } from '../model/useAiGate';

/**
 * Wraps an AI entry point. Usable → renders the children; otherwise a compact locked state whose button opens
 * the gate sheet (which explains why and where to go). While the balance is unknown the children render —
 * the server decides (fail open).
 */
export function AiGate({ feature, children, className }: { feature: string; children: ReactNode; className?: string }) {
  const t = useTranslations('aiGate');
  const locale = useLocale() as Locale;
  const gate = useAiGate(feature);
  if (gate.canUse) return <>{children}</>;
  return (
    <Card className={className ? `aigate-locked ${className}` : 'aigate-locked'}>
      <IconCircle icon={gate.status === 'disabled' ? 'clock' : 'crown'} tone={gate.status === 'disabled' ? 'neutral' : 'warm'} size="md" />
      <p className="aigate-locked-text">
        {gate.status === 'no_subscription' ? t('locked.subscription') : gate.status === 'disabled' ? t('locked.disabled') : t('locked.credits')}
        {gate.cost !== null && gate.status !== 'disabled' ? ` ${t('locked.cost', { cost: formatCredits(gate.cost, locale) })}` : ''}
      </p>
      <SecondaryButton block={false} onClick={gate.guard}>
        {t('locked.button')}
      </SecondaryButton>
    </Card>
  );
}
