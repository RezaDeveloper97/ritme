'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { type UserProfile, useUserProfile } from '@/entities/user';
import {
  JalaliDateWheels,
  jalaliPartsToApiDate,
  useUpdateProfile,
} from '@/features/edit-profile';
import { getApiErrorMessage } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { type JalaliParts, toApiDate, today, toJalali, todayJalali } from '@/shared/lib/date';
import { NavBack, WheelPicker } from '@/shared/ui';

// API bounds for the two durations (POST /profile validation).
const CYCLE_MIN = 15;
const CYCLE_MAX = 60;
const PERIOD_MIN = 1;
const PERIOD_MAX = 15;

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

// Section title styled like ProfilePage's Group headers (13px, muted, bold).
function SectionLabel({ children }: { children: string }) {
  return (
    <div style={{ fontSize: 13, fontWeight: 800, color: 'var(--muted)', margin: '0 4px 8px' }}>
      {children}
    </div>
  );
}

/**
 * Single "N days" wheel bounded by the API's validation range. Labels go
 * through ICU so digits localize per locale (§6).
 */
function DaysWheel({
  id,
  min,
  max,
  value,
  onChange,
  label,
}: {
  id: string;
  min: number;
  max: number;
  value: number;
  onChange: (v: number) => void;
  label: (days: number) => string;
}) {
  const items = Array.from({ length: max - min + 1 }, (_, i) => label(min + i));
  return (
    <div style={{ display: 'flex', justifyContent: 'center', position: 'relative' }}>
      {/* The class stretches inset-inline 0..0; a fixed width + auto inline
          margins centers the band in both directions (RTL-safe, §12). */}
      <div className="wheel-band" style={{ width: 150, marginInline: 'auto' }} />
      <WheelPicker
        id={id}
        items={items}
        selectedIndex={clamp(value, min, max) - min}
        width={150}
        onChange={(i) => onChange(min + i)}
      />
    </div>
  );
}

/**
 * The editable form, mounted only once the profile has loaded so the wheels
 * (which read their initial position on mount) start on the user's saved
 * values. Cycle data is sensitive (§11): displayed and submitted, never logged.
 */
function HealthForm({ profile }: { profile: UserProfile }) {
  const t = useTranslations('profileEdit');
  const router = useRouter();
  const update = useUpdateProfile();

  const health = profile.health;
  const nowJalali = todayJalali();

  const [cycleDuration, setCycleDuration] = useState(() =>
    clamp(health?.cycleDuration ?? 28, CYCLE_MIN, CYCLE_MAX),
  );
  const [periodDuration, setPeriodDuration] = useState(() =>
    clamp(health?.periodDuration ?? 6, PERIOD_MIN, PERIOD_MAX),
  );
  const [lastPeriod, setLastPeriod] = useState<JalaliParts>(() =>
    // The API stores Gregorian ISO; the user always edits in Jalali (§7).
    health?.lastPeriodStart ? toJalali(new Date(health.lastPeriodStart)) : todayJalali(),
  );
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSave = () => {
    if (update.isPending) return;
    setLocalError(null);

    const lastPeriodStart = jalaliPartsToApiDate(lastPeriod);
    // The API rejects future dates; catch it client-side so the user gets a
    // localized message instead of a raw 422.
    if (lastPeriodStart > toApiDate(today())) {
      setLocalError(t('errors.lastPeriodFuture'));
      return;
    }

    update.mutate(
      {
        cycle_duration: cycleDuration,
        period_duration: periodDuration,
        last_period_start: lastPeriodStart,
      },
      { onSuccess: () => router.back() },
    );
  };

  const errorMessage =
    localError ?? (update.isError ? (getApiErrorMessage(update.error) ?? t('errors.generic')) : null);

  return (
    <>
      <div className="scroll" style={{ paddingBottom: 12 }}>
        {/* Cycle length */}
        <section style={{ padding: '0 18px', marginTop: 12 }}>
          <SectionLabel>{t('health.cycleLabel')}</SectionLabel>
          <div className="card" style={{ padding: '14px 10px' }}>
            <DaysWheel
              id="ph-cycle"
              min={CYCLE_MIN}
              max={CYCLE_MAX}
              value={cycleDuration}
              onChange={setCycleDuration}
              label={(days) => t('health.dayCount', { days })}
            />
          </div>
        </section>

        {/* Period length */}
        <section style={{ padding: '0 18px', marginTop: 20 }}>
          <SectionLabel>{t('health.periodLabel')}</SectionLabel>
          <div className="card" style={{ padding: '14px 10px' }}>
            <DaysWheel
              id="ph-period"
              min={PERIOD_MIN}
              max={PERIOD_MAX}
              value={periodDuration}
              onChange={setPeriodDuration}
              label={(days) => t('health.dayCount', { days })}
            />
          </div>
        </section>

        {/* Last period start (Jalali, §7) */}
        <section style={{ padding: '0 18px', marginTop: 20 }}>
          <SectionLabel>{t('health.lastPeriodLabel')}</SectionLabel>
          <div className="card" style={{ padding: '14px 10px' }}>
            <JalaliDateWheels
              idPrefix="ph-last"
              value={lastPeriod}
              onChange={setLastPeriod}
              minYear={nowJalali.year - 1}
              maxYear={nowJalali.year}
            />
          </div>
          <p style={{ fontSize: 12, color: 'var(--muted)', margin: '8px 4px 0' }}>
            {t('health.lastPeriodHint')}
          </p>
        </section>
      </div>

      <div style={{ padding: '6px 16px 8px' }}>
        {errorMessage ? (
          <p
            role="alert"
            style={{ fontSize: 13, color: 'var(--danger)', textAlign: 'center', margin: '0 0 8px' }}
          >
            {errorMessage}
          </p>
        ) : null}
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={update.isPending}
        >
          {update.isPending ? t('saving') : t('save')}
        </button>
      </div>
    </>
  );
}

/** Edit screen for cycle & health settings: cycle/period length, last period start. */
export function ProfileHealthPage() {
  const t = useTranslations('profileEdit');
  const router = useRouter();
  const { data: profile } = useUserProfile();

  return (
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} label={t('back')} />
        <span style={{ fontSize: 16, fontWeight: 800, color: 'var(--ink)' }}>
          {t('health.title')}
        </span>
        {/* Spacer mirroring the back button keeps the title centered (RTL-safe). */}
        <span style={{ width: 40 }} aria-hidden />
      </div>

      {profile ? (
        <HealthForm profile={profile} />
      ) : (
        <div className="scroll" style={{ padding: '24px 18px' }}>
          <p style={{ fontSize: 14, color: 'var(--muted)', textAlign: 'center' }}>{t('loading')}</p>
        </div>
      )}
    </div>
  );
}
