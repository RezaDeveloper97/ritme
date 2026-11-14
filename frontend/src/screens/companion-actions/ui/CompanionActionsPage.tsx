'use client';

import { useTranslations } from 'next-intl';

import { useCompanionHome } from '@/entities/companion';
import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { Icon, InfoNote, ScreenHeader, SectionTitle, SkyLayer } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ActionCard, orderedCategories, useSuggestionsMuted } from '@/widgets/companion-support';

/**
 * `/companion/actions` — «اقدام‌های همراهی» (companion v2 compCards): the idea catalogue by support kind, the
 * kinds she prefers first (`settings.action_prefs`, then her goal). Ideas, not tasks; none needs a purchase.
 * The bell mutes the suggestion sections on the home for this viewer (device storage).
 */
export function CompanionActionsPage() {
  const t = useTranslations('companionHome.actions');
  const router = useRouter();
  const mounted = useMounted();
  const home = useCompanionHome();
  const [muted, setMuted] = useSuggestionsMuted();
  // With several partners the first one's preferences order the page (the home picks per partner).
  const settings = home.data?.partners[0]?.link.settings ?? null;
  const categories = orderedCategories(settings);
  const isMuted = mounted && muted;

  return (
    <div className="view cmh-page">
      <SkyLayer />
      <ScreenHeader
        title={t('title')}
        onBack={() => router.replace('/companion')}
        backLabel={t('back')}
        action={
          <button
            type="button"
            className="nb-hbtn"
            aria-pressed={isMuted}
            aria-label={t('mute')}
            onClick={() => setMuted(!isMuted)}
          >
            <Icon name={isMuted ? 'bellPlain' : 'bell'} size={20} strokeWidth={1.8} />
          </button>
        }
      />
      <div className="scroll cmh-scroll">
        <div className="cmh-body">
          <InfoNote>{t('note')}</InfoNote>
          {isMuted ? (
            <p className="cmh2-muted is-static" role="status">
              <Icon name="bellPlain" size={16} />
              {t('mutedStatus')}
            </p>
          ) : null}
          {categories.map((c) => (
            <section key={c.id} className="cmh-sec" aria-labelledby={`cmh2-cat-${c.id}`}>
              <SectionTitle id={`cmh2-cat-${c.id}`} title={t(`categories.${c.id}`)} />
              <ul className="cmh2-acts-scroll">
                {c.cards.map((card) => (
                  <ActionCard key={card.key} cardKey={card.key} icon={card.icon} tone={c.tone} />
                ))}
              </ul>
            </section>
          ))}
        </div>
      </div>
      <BottomNav mode="companion" />
    </div>
  );
}
