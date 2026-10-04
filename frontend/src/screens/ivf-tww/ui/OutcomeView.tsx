'use client';

import { useTranslations } from 'next-intl';
import { useId } from 'react';

import type { IvfNextStep, IvfOutcome } from '@/entities/ivf';
import { useRouter } from '@/shared/i18n';
import { IconCircle, InfoNote, ListGroup, ListRow, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { FOLLOW_UP_HREF, followUps } from '../model/tww';

/**
 * After the outcome is saved. Calm on every result — no celebration, no
 * confetti: positive offers bloom's pregnancy setup (or later); negative offers,
 * only if she wants, CB-LOSS-02's `/loss` path or the IVF home; cancelled → the
 * IVF home. Nothing here is announced to anyone else.
 */
export function OutcomeView({ outcome }: { outcome: IvfOutcome }) {
  const t = useTranslations('ivf');
  const router = useRouter();
  const ids = useId();
  const { result } = outcome;
  const steps = followUps(result, outcome.nextSteps);
  const go = (step: IvfNextStep) => router.push(FOLLOW_UP_HREF[step]);

  return (
    <div className="ivf-body tww-outcome">
      <section className="tww-outcome-head" aria-labelledby={`${ids}-title`}>
        <IconCircle icon={result === 'positive' ? 'heart' : 'sprout'} tone={result === 'positive' ? 'bloom' : 'neutral'} size="lg" />
        <h2 id={`${ids}-title`} className="tww-outcome-title" tabIndex={-1}>
          {t(`tww.outcome.${result}.title`)}
        </h2>
        <p className="tww-outcome-body">{t(`tww.outcome.${result}.body`)}</p>
      </section>

      <InfoNote icon="pill">{t(`tww.outcome.${result}.note`)}</InfoNote>

      {result === 'positive' ? (
        <div className="tww-outcome-actions">
          {steps.includes('pregnancy_setup') ? (
            <PrimaryButton onClick={() => go('pregnancy_setup')}>{t('tww.outcome.positive.cta')}</PrimaryButton>
          ) : null}
          <SecondaryButton variant="text" onClick={() => go('new_cycle')}>
            {t('tww.outcome.positive.later')}
          </SecondaryButton>
        </div>
      ) : (
        <section className="ivf-section" aria-labelledby={`${ids}-next`}>
          <h3 id={`${ids}-next`} className="tww-section-title is-quiet">
            {t('tww.outcome.options')}
          </h3>
          <ListGroup>
            {steps.map((step) =>
              step === 'loss' ? (
                <ListRow
                  key={step}
                  icon="heart"
                  iconTone="period"
                  title={t('tww.outcome.loss.title')}
                  description={t('tww.outcome.loss.body')}
                  onClick={() => go('loss')}
                />
              ) : step === 'new_cycle' ? (
                <ListRow
                  key={step}
                  icon="home"
                  title={t('tww.outcome.newCycle.title')}
                  description={t('tww.outcome.newCycle.body')}
                  onClick={() => go('new_cycle')}
                />
              ) : null,
            )}
          </ListGroup>
        </section>
      )}
    </div>
  );
}
