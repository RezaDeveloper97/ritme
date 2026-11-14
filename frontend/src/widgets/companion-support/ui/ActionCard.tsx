'use client';

import { useTranslations } from 'next-intl';

import { IconCircle, type IconName, type Tone } from '@/shared/ui';

import type { ActionCardKey } from '../model/actions';

interface ActionCardProps {
  cardKey: ActionCardKey;
  icon: IconName;
  tone: Tone;
}

/** One suggestion card (compCards `actcard`): icon disc, title, a short «اگر دوست داشتی…» line. */
export function ActionCard({ cardKey, icon, tone }: ActionCardProps) {
  const t = useTranslations('companionHome.actions.cards');
  return (
    <li className="nb-card cmh2-act">
      <IconCircle icon={icon} tone={tone} size="md" />
      <b className="cmh2-act-title">{t(`${cardKey}.title`)}</b>
      <span className="cmh2-act-desc">{t(`${cardKey}.desc`)}</span>
    </li>
  );
}
