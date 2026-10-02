'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale, useDirection } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { companionName, hoursUntil, sharedSections } from '../lib/grants';
import type { OwnerCompanion } from '../model/types';
import { PersonBubble } from './FamilyStrip';

interface CompanionCardProps {
  companion: OwnerCompanion;
  onOpen: () => void;
}

/**
 * One companion on Hamdam_List: initial, name, «همسر · فعال», then a chip per
 * shared section («داروها: ویرایش» in turquoise, «پریود: فقط دیدن» in violet).
 * The whole card opens the detail screen.
 */
export function CompanionCard({ companion, onOpen }: CompanionCardProps) {
  const t = useTranslations('companions');
  const rtl = useDirection() === 'rtl';
  const locale = useLocale() as Locale;
  const name = companionName(companion) ?? t('unnamed');
  const chips = sharedSections(companion.grants);
  const hours = companion.invite ? hoursUntil(companion.invite.expiresAt) : null;

  return (
    <button type="button" className="nb-card cmp-card" onClick={onOpen}>
      <span className="cmp-card-head">
        <PersonBubble name={companionName(companion)} tone="companion" />
        <span className="cmp-card-text">
          <b className="cmp-card-name">{name}</b>
          <span className="cmp-card-sub">
            {t('statusLine', { type: t(`types.${companion.type}`), status: t(`status.${companion.status}`) })}
          </span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="cmp-card-chev" />
      </span>
      {chips.length ? (
        <span className="cmp-chips">
          {chips.map(({ section, level }) => (
            <span key={section} className={`cmp-chip is-${level}`}>
              {t('chip', { section: t(`sectionsShort.${section}`), level: t(`chipLevels.${level}`) })}
            </span>
          ))}
        </span>
      ) : (
        <span className="cmp-card-empty">{t('nothingShared')}</span>
      )}
      {companion.status === 'invited' && hours !== null ? (
        <span className={hours > 0 ? 'cmp-card-invite' : 'cmp-card-invite is-expired'}>
          <Icon name="clock" size={14} />
          {hours > 0 ? t('list.expiresIn', { hours: formatNumber(hours, locale) }) : t('list.expired')}
        </span>
      ) : null}
    </button>
  );
}
