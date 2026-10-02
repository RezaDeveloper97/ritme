'use client';

import type { ComponentType, ReactNode } from 'react';
import { useTranslations } from 'next-intl';

import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import { PlusFeatureGate } from '@/entities/plus';
import { Icon, type IconName, SecondaryButton, type Tone } from '@/shared/ui';

/** The Plus entitlement of voice logging (backend-go/internal/plus/entitlements.go). */
export const VOICE_FEATURE = 'plus.voice_log';

/**
 * What the log sheet hands the voice recorder (B-N3-05 `features/voice-log`, passed in by the screen —
 * a feature can't import another feature). The recorder never saves on its own: it merges confirmed
 * suggestions into this draft with `setParam`, marks them with `markVoice`, and either opens the matching
 * manual section or runs the sheet's normal `save`.
 */
export interface VoiceLogSlotProps {
  date: string;
  mode: string | null;
  categories: readonly LogCategory[];
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  markVoice: (keys: readonly string[]) => void;
  /** Switches to «ثبت دستی» and opens the category's section. */
  openSection: (category: string) => void;
  /** Accent of a category (review chips use the log sheet's colours). */
  toneOf: (category: string) => Tone;
  /** Icon of a category (CB-VOICE-02 review groups, saved list, today's status). */
  iconOf: (category: string) => IconName;
  /** What is logged on the day (label + summary per entry, keyed `category.param[.item]`). */
  entries: readonly { key: string; label: string; summary: string }[];
  /**
   * Saves the draft (after the merges of the same event have landed). With `onDone` it runs that instead of
   * the sheet's `onSaved` — also at once when nothing changed (CB-VOICE-02: the diary commit follows).
   */
  save: (onDone?: () => void) => void;
  saving: boolean;
  saveError: boolean;
  onManual: () => void;
  /** Leaves the flow from «برگشت به خانه» (closes the sheet / goes home). */
  onDone: () => void;
  /** Record / review / saved are full-height steps: the sheet hides its date strip and tabs meanwhile. */
  onImmersive?: (on: boolean) => void;
}

export type VoiceLogSlot = ComponentType<VoiceLogSlotProps>;

/**
 * «ثبت با صدا» (nbl_Log_Day). With the recorder slot (B-N3-05) it renders the recorder behind the Plus
 * lock; without one (a host that doesn't wire voice) it keeps the explanatory «به‌زودی» card.
 */
export function VoiceTab({ onManual, children }: { onManual: () => void; children?: ReactNode }) {
  const t = useTranslations('logSheet');
  if (children) {
    return (
      <PlusFeatureGate feature={VOICE_FEATURE} className="lday-voice-gate">
        {children}
      </PlusFeatureGate>
    );
  }
  return (
    <PlusFeatureGate feature={VOICE_FEATURE} className="lday-voice-gate">
      <section className="nb-card lday-voice">
        <span className="lday-voice-mic" aria-hidden>
          <Icon name="mic" size={30} strokeWidth={1.8} />
        </span>
        <h3 className="lday-voice-title">{t('voice.title')}</h3>
        <p className="lday-voice-body">{t('voice.body')}</p>
        <span className="lday-voice-soon">{t('voice.soon')}</span>
        <SecondaryButton onClick={onManual}>{t('voice.manual')}</SecondaryButton>
      </section>
    </PlusFeatureGate>
  );
}
