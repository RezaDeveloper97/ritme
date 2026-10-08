'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { useLifeStage } from '@/entities/user';
import { Link, useDirection } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { CHIP_TONE, chipMode } from '../model/chip';

/**
 * Today header mode chip (CB-NAV-03, nbd_Nav_Today / IA_Map «حالت از من › حالت اپ یا چیپ حالت»): a pill with
 * the mode's dot and name that opens the life-stage screen `/profile/mode`. Nothing until the life stage is
 * known (no flash of the wrong mode) and nothing for a companion account.
 */
export function ModeChip({ className }: { className?: string }) {
  const t = useTranslations('nav.modeChip');
  const life = useLifeStage();
  const dir = useDirection();
  const mode = chipMode(life.data);
  if (!mode) return null;
  const name = t(`modes.${mode}`);
  const rtl = dir === 'rtl';
  return (
    <Link href="/profile/mode" className={clsx('mchip', `tone-${CHIP_TONE[mode]}`, className)} aria-label={t('label', { mode: name })}>
      <span className="mchip-dot" aria-hidden />
      <span className="mchip-name">{name}</span>
      <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={14} strokeWidth={2.2} className="mchip-chev" />
    </Link>
  );
}
