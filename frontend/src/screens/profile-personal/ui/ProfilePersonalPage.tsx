'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type UserProfile, useUserProfile } from '@/entities/user';
import {
  DateWheels,
  datePartsToApiDate,
  type UpdateProfileInput,
  useUpdateProfile,
} from '@/features/edit-profile';
import { getApiErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { birthYearRange, type DateParts, toApiDate, today, toParts, todayParts } from '@/shared/lib/date';
import { Icon, NavBack, RulerPicker } from '@/shared/ui';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (value: string | number, loc: Locale) =>
  loc === 'fa' ? String(value).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(value);


// Section title styled like ProfilePage's Group headers (13px, muted, bold).
function SectionLabel({ children }: { children: string }) {
  return (
    <div className="prof-group-t">
      {children}
    </div>
  );
}

/**
 * The editable form, mounted only once the profile has loaded so the wheel and
 * ruler pickers (which read their initial position on mount) start on the
 * user's saved values. Sensitive data (§11): displayed and submitted, never
 * logged.
 */
function PersonalForm({ profile }: { profile: UserProfile }) {
  const t = useTranslations('profileEdit');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const update = useUpdateProfile();

  const health = profile.health;
  const nowParts = todayParts(loc);
  const birthYears = birthYearRange(loc);

  const [name, setName] = useState(profile.name ?? '');
  const [birth, setBirth] = useState<DateParts>(() =>
    // Birthday crosses the API as Gregorian ISO; edit it in the calendar the
    // user's locale reads — Jalali for fa, Gregorian for en (§7).
    health?.birthday
      ? toParts(new Date(health.birthday), loc)
      : { year: nowParts.year - 25, month: 1, day: 1 },
  );
  const [weight, setWeight] = useState(health?.weight ?? 60);
  const [height, setHeight] = useState(health?.height ?? 165);
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSave = () => {
    if (update.isPending) return;
    setLocalError(null);

    const birthday = datePartsToApiDate(birth, loc);
    // The API requires the birthday to be strictly before today; catch it
    // client-side so the user gets a localized message instead of a 422.
    if (birthday >= toApiDate(today())) {
      setLocalError(t('errors.birthdayFuture'));
      return;
    }

    const payload: UpdateProfileInput = {
      birthday,
      weight: Math.round(weight),
      height: Math.round(height),
    };
    const trimmed = name.trim();
    if (trimmed.length > 0) payload.name = trimmed;

    update.mutate(payload, { onSuccess: () => router.back() });
  };

  const errorMessage =
    localError ?? (update.isError ? (getApiErrorMessage(update.error) ?? t('errors.generic')) : null);

  return (
    <>
      <div className="scroll prof-edit-scroll">
        {/* Name */}
        <section className="prof-group is-tight">
          <SectionLabel>{t('personal.nameLabel')}</SectionLabel>
          <div className="field">
            <input
              value={name}
              placeholder={t('personal.namePlaceholder')}
              onChange={(e) => setName(e.target.value)}
            />
            <span className="placeholder-soft">
              <Icon name="pencil" size={18} />
            </span>
          </div>
        </section>

        {/* Birthday — wheels follow the locale's calendar (§7) */}
        <section className="prof-group">
          <SectionLabel>{t('personal.birthdayLabel')}</SectionLabel>
          <div className="card prof-wheel-card">
            <DateWheels
              idPrefix="pp-birth"
              value={birth}
              onChange={setBirth}
              minYear={birthYears.min}
              maxYear={nowParts.year}
            />
          </div>
        </section>

        {/* Weight */}
        <section className="prof-group">
          <SectionLabel>{t('personal.weightLabel')}</SectionLabel>
          <div className="card prof-wheel-card is-tall">
            <RulerPicker
              min={30}
              max={200}
              value={weight}
              unit={t('personal.kgUnit')}
              onChange={setWeight}
              toDisplay={(v) => localizeNum(v, loc)}
            />
          </div>
        </section>

        {/* Height */}
        <section className="prof-group">
          <SectionLabel>{t('personal.heightLabel')}</SectionLabel>
          <div className="card prof-wheel-card is-tall">
            <RulerPicker
              min={100}
              max={220}
              value={height}
              unit={t('personal.cmUnit')}
              onChange={setHeight}
              toDisplay={(v) => localizeNum(v, loc)}
            />
          </div>
        </section>
      </div>

      <div className="prof-edit-actions">
        {errorMessage ? (
          <p role="alert" className="prof-edit-error">
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

/** Edit screen for the account's personal info: name, birthday, weight, height. */
export function ProfilePersonalPage() {
  const t = useTranslations('profileEdit');
  const router = useRouter();
  const { data: profile } = useUserProfile();

  return (
    <div className="view prof-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} label={t('back')} />
        <span className="prof-edit-title">
          {t('personal.title')}
        </span>
        {/* Spacer mirroring the back button keeps the title centered (RTL-safe). */}
        <span className="prof-hdr-spacer" aria-hidden />
      </div>

      {profile ? (
        <PersonalForm profile={profile} />
      ) : (
        <div className="scroll prof-edit-loading">
          <p className="prof-edit-loading-t">{t('loading')}</p>
        </div>
      )}
    </div>
  );
}
