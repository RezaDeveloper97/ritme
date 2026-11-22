'use client';

import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { CycleValuesCard } from '@/entities/cycle';

import { needsPeriodData } from '../model/cycle-data';
import { PredictionsCard } from './PredictionsCard';
import { usePredictionSlots } from './usePredictionSlots';

// Lazy: the sheet is fetched only once the sync nudge is tapped (T-M12-27b).
const QuickEditSheet = dynamic(() => import('@/features/edit-profile').then((m) => m.QuickEditSheet), {
  loading: () => null,
});

/**
 * The «پیش‌بینی‌ها» card on its own (moved off «امروز» to `/analysis`): it reads the same cached
 * queries as the home, so the dates match the ring. Nothing while today's cycle is loading,
 * failed, or has no period on record. `hideFertility` is teen mode (B-N2-03).
 */
export function CyclePredictions({ hideFertility = false }: { hideFertility?: boolean }) {
  const t = useTranslations('home');
  const [cycleSheetOpen, setCycleSheetOpen] = useState(false);
  // Mounted from the first open on, so it keeps its close animation.
  const [cycleSheetEver, setCycleSheetEver] = useState(false);
  const {
    todayQuery,
    limits,
    card,
    nextPeriodSlot,
    windowSlot,
    pmsSlot,
    ovulationDate,
    toPrediction,
    cycleLengthSummary,
  } = usePredictionSlots();
  const todayData = todayQuery.data;
  if (!todayQuery.isSuccess || !todayData) return null;
  if (needsPeriodData(todayData.cycleView ?? null, todayData.calculation ?? null)) return null;

  // The §12 sync nudge: only when recent cycles disagree with the profile value and the
  // engine is still on the user's number — a suggestion never applies itself, she confirms it.
  const view = todayData.cycleView ?? null;
  const suggested = view?.calculatedValues.cycleLength ?? null;
  const logged = view?.profileValues.cycleLength ?? null;
  const syncNudge =
    view &&
    suggested != null &&
    logged != null &&
    suggested !== logged &&
    view.effectiveValues.cycleLength !== suggested ? (
      <CycleValuesCard
        title={t('values.title')}
        loggedLabel={t('values.logged')}
        loggedValue={t('days', { n: logged })}
        suggestion={{
          text: t('values.suggestion', { n: suggested }),
          ctaLabel: t('values.syncCta', { n: suggested }),
          onSync: () => {
            setCycleSheetEver(true);
            setCycleSheetOpen(true);
          },
        }}
        basedOnText={t('values.basedOn', {
          source: t(
            view.effectiveValues.source === 'recent_valid_cycles'
              ? 'values.source.recent_valid_cycles'
              : view.effectiveValues.source === 'profile'
                ? 'values.source.profile'
                : 'values.source.default',
          ),
        })}
      />
    ) : null;

  return (
    <>
      <PredictionsCard
        nextPeriod={toPrediction(nextPeriodSlot)}
        pms={toPrediction(pmsSlot)}
        window={hideFertility ? null : toPrediction(windowSlot)}
        ovulation={hideFertility ? null : ovulationDate}
        ovulationSource={card?.ovulation?.source ?? null}
        hideWindow={card != null && card.fertile == null}
        hideFertility={hideFertility || limits.hideFertility}
        withdrawalBleed={limits.withdrawalBleed}
        note={limits.note}
        length={cycleLengthSummary}
        footer={syncNudge}
      />
      {/* Seeded with what recent cycles measured, so the wheel opens on the
          value the nudge proposes and saving is a single tap. */}
      {cycleSheetEver && (
        <QuickEditSheet
          field={cycleSheetOpen ? 'cycleDuration' : null}
          values={{ cycleDuration: suggested ?? logged ?? null }}
          onClose={() => setCycleSheetOpen(false)}
        />
      )}
    </>
  );
}
