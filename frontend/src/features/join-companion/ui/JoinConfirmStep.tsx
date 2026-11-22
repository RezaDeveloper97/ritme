'use client';

import { useTranslations } from 'next-intl';

import { Icon, IconCircle } from '@/shared/ui';

import type { JoinCompanion } from '../model/use-join-companion';
import '@/shared/ui/styles/cmh.css';

function firstLetter(name: string): string {
  return Array.from(name.trim())[0] ?? '';
}

/**
 * compv2 `joinConfirm`: «به چرخه‌ی سارا وصل می‌شوی» and the two boundaries — what you will see (only what
 * she chose, in probabilistic words) and that you can neither log nor change anything, while she can end it.
 */
export function JoinConfirmStep({ join }: { join: JoinCompanion }) {
  const t = useTranslations('companionHome.join.confirm');
  const name = join.ownerName;
  return (
    <div className="cmh2-join">
      <span className="cmh2-her" aria-hidden>
        {name ? firstLetter(name) : <Icon name="heart" size={28} />}
      </span>
      <div className="cmh2-join-intro">
        <h2 className="cmh2-title">{name ? t('title', { name }) : t('titleNoName')}</h2>
        <p className="cmh2-sub">{t('sub')}</p>
      </div>
      <ul className="nb-card cmh2-how">
        <li className="cmh2-how-row">
          <IconCircle icon="eye" tone="brand" size="sm" />
          <span className="cmh2-how-text">
            <b>{t('seesTitle')}</b>
            {name ? t('sees', { name }) : t('seesNoName')}
          </span>
        </li>
        <li className="cmh2-how-row">
          <IconCircle icon="shield" tone="brand" size="sm" />
          <span className="cmh2-how-text">
            <b>{t('boundaryTitle')}</b>
            {t('boundary')}
          </span>
        </li>
      </ul>
    </div>
  );
}
