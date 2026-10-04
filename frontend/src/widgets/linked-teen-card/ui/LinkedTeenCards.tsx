'use client';

import { useTranslations } from 'next-intl';

import { ParentViewCard, useTeenLinked } from '@/entities/teen';
import { SectionTitle, StatusPill } from '@/shared/ui';

/**
 * The parent's side of mother sharing (CB-TEEN-03, nbl_Teen_Parent «مادرت این
 * را می‌بیند»): one read-only card per teen who invited her (GET /teen/linked),
 * on her Today. Only what the teen shares is filled — a week bucket for the
 * next period, the school kit, her note — and nothing here can be changed.
 * Renders nothing for everyone who is nobody's parent (and on a failed read:
 * the card is a courtesy, never a blocker for the home).
 */
export function LinkedTeenCards() {
  const t = useTranslations('teen.parent.linked');
  const query = useTeenLinked();
  const cards = query.data ?? [];
  if (!cards.length) return null;

  return (
    <section className="ltc-sec" aria-labelledby="ltc-title">
      <SectionTitle id="ltc-title" title={t('title')} />
      {cards.map((card) => {
        const name = card.teenName?.trim() || t('someone');
        return (
          <ParentViewCard
            key={card.linkId}
            name={name}
            view={card}
            badge={
              <StatusPill tone="data" icon="eye">
                {t('readOnly')}
              </StatusPill>
            }
            footer={<p className="tnp-view-foot">{t('footer', { name })}</p>}
          />
        );
      })}
    </section>
  );
}
