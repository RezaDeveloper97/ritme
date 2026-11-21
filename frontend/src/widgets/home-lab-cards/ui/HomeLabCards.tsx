'use client';

import { useTranslations } from 'next-intl';

import { type LabService, useLabServices } from '@/entities/partner-lab';
import { Link } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { Card, Icon, StatusPill } from '@/shared/ui';
import '@/shared/ui/styles/hlb.css';

/** One card per service kind: the first active service of «آزمایش در منزل» and of «چکاپ سالانه». */
function pickPerKind(services: readonly LabService[]): LabService[] {
  const seen = new Set<string>();
  return services.filter((s) => (seen.has(s.kind) ? false : (seen.add(s.kind), true)));
}

/**
 * The «پیشنهاد» cards of Today (nbl_/nbd_Prem_TrialHome): home blood draw and the annual checkup. They exist only
 * while the admin catalog has an active service and are never driven by the user's logs, symptoms or results — the
 * same cards show for everyone. Nothing renders while loading, on error, or when nothing is active.
 */
export function HomeLabCards() {
  const t = useTranslations('homeLab');
  const q = useLabServices();
  if (!q.data || q.data.services.length === 0) return null;
  const cards = pickPerKind(q.data.services);

  return (
    <section className="hlb-cards" aria-label={t('card.label')}>
      {cards.map((s) => {
        const checkup = s.kind === 'checkup';
        return (
          <Link key={s.id} href={`/services/home-lab/new?service=${s.id}`} className="hlb-card-link" {...pilot(`home-lab.card:${s.id}`)}>
            <Card className="hlb-card">
              <span className="hlb-card-art" aria-hidden>
                <Icon name={checkup ? 'stetho' : 'flask'} size={26} strokeWidth={1.6} />
              </span>
              <span className="hlb-card-main">
                <StatusPill tone="bloom">{t('card.label')}</StatusPill>
                <span className="hlb-card-title">{s.title}</span>
                <span className="hlb-card-sub">{s.description || t(checkup ? 'card.checkupSub' : 'card.homeDrawSub')}</span>
              </span>
              <span className="hlb-card-cta">{t(checkup ? 'card.book' : 'card.request')}</span>
            </Card>
          </Link>
        );
      })}
    </section>
  );
}
