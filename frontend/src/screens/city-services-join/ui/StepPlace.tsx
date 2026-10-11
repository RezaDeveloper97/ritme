'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { JOIN_AGE_RANGES, JOIN_AMENITIES, type JoinApplication, useSaveJoinStep } from '@/entities/city-service';
import { ChipGroup, PillChip, PrimaryButton, Switch } from '@/shared/ui';

import {
  DAY_KEYS,
  DAYS,
  DEFAULT_CLOSE,
  DEFAULT_OPEN,
  step2Body,
  step2From,
  toggleItem,
  validHours,
} from '../model/join';
import { useAutosave } from '../model/use-autosave';
import { PhotoManager } from './PhotoManager';
import { PinPicker } from './PinPicker';
import { SaveHint, useErrorText } from './parts';

/** Step 2 «مکان و تصاویر» (nbl_Dir_JoinForm): pin, address, photos, about, ages, amenities and weekly hours. */
export function StepPlace({ app, onNext, onSaveError }: { app: JoinApplication; onNext: () => void; onSaveError: (e: unknown) => void }) {
  const t = useTranslations('cityJoin');
  const errorText = useErrorText();
  const save = useSaveJoinStep(app.id);
  const [form, setForm] = useState(() => step2From(app));
  const { state, flush } = useAutosave(form, (v) =>
    save.mutateAsync({ step: 2, body: step2Body(v) }).catch((e) => {
      onSaveError(e);
      throw e;
    }),
  );
  const set = <K extends keyof typeof form>(key: K, value: (typeof form)[K]) => setForm((f) => ({ ...f, [key]: value }));

  const hoursOf = (day: number) => form.hours.find((h) => h.day === day);
  const setHours = (day: number, patch: Partial<{ open: string; close: string }> | null) =>
    set(
      'hours',
      patch === null
        ? form.hours.filter((h) => h.day !== day)
        : [...form.hours.filter((h) => h.day !== day), { day, open: DEFAULT_OPEN, close: DEFAULT_CLOSE, ...hoursOf(day), ...patch }].sort(
            (a, b) => a.day - b.day,
          ),
    );
  const hoursBad = form.hours.some((h) => !validHours(h));

  const next = async () => {
    if (await flush()) onNext();
  };

  return (
    <div className="cjn-stack">
      <section className="card fld-card cjn-fields">
        <PinPicker
          value={form.lat !== null && form.lng !== null ? { lat: form.lat, lng: form.lng } : null}
          onChange={(p) => setForm((f) => ({ ...f, lat: p?.lat ?? null, lng: p?.lng ?? null }))}
        />
        <label className="fld-label">
          <span className="fld-label-t">{t('place.address')}</span>
          <span className="field cjn-area">
            <textarea rows={2} maxLength={300} autoComplete="street-address" value={form.address} placeholder={t('place.addressPh')} onChange={(e) => set('address', e.target.value)} />
          </span>
        </label>
      </section>

      <section className="card fld-card cjn-fields">
        <PhotoManager app={app} />
      </section>

      <section className="card fld-card cjn-fields">
        <label className="fld-label">
          <span className="fld-label-t">{t('place.about')}</span>
          <span className="field cjn-area">
            <textarea rows={4} maxLength={2000} value={form.about} placeholder={t('place.aboutPh')} onChange={(e) => set('about', e.target.value)} />
          </span>
        </label>
        <div className="cjn-group">
          <span className="fld-label-t">{t('place.ages')}</span>
          <ChipGroup label={t('place.ages')}>
            {JOIN_AGE_RANGES.map((code) => (
              <PillChip key={code} mode="multi" pressed={form.ageRanges.includes(code)} onPressedChange={() => set('ageRanges', toggleItem(form.ageRanges, code))}>
                {t(`ages.${code}`)}
              </PillChip>
            ))}
          </ChipGroup>
        </div>
        <div className="cjn-group">
          <span className="fld-label-t">{t('place.amenities')}</span>
          <ChipGroup label={t('place.amenities')}>
            {JOIN_AMENITIES.map((code) => (
              <PillChip key={code} mode="multi" pressed={form.amenities.includes(code)} onPressedChange={() => set('amenities', toggleItem(form.amenities, code))}>
                {t(`amenities.${code}`)}
              </PillChip>
            ))}
          </ChipGroup>
        </div>
      </section>

      <section className="card fld-card cjn-fields" aria-label={t('place.hours')}>
        <h3 className="cjn-label">{t('place.hours')}</h3>
        <ul className="cjn-hours">
          {DAYS.map((day) => {
            const h = hoursOf(day);
            const titleId = `cjn-day-${day}`;
            return (
              <li key={day} className="cjn-day">
                <span id={titleId} className="cjn-day-name">
                  {t(`days.${DAY_KEYS[day]}`)}
                </span>
                {h ? (
                  <span className="cjn-times">
                    <input
                      type="time"
                      dir="ltr"
                      aria-label={`${t(`days.${DAY_KEYS[day]}`)} ${t('place.from')}`}
                      value={h.open}
                      onChange={(e) => setHours(day, { open: e.target.value })}
                    />
                    <span aria-hidden>–</span>
                    <input
                      type="time"
                      dir="ltr"
                      aria-label={`${t(`days.${DAY_KEYS[day]}`)} ${t('place.to')}`}
                      value={h.close}
                      onChange={(e) => setHours(day, { close: e.target.value })}
                    />
                  </span>
                ) : (
                  <span className="cjn-closed">{t('place.closed')}</span>
                )}
                <Switch compact checked={Boolean(h)} labelledBy={titleId} onCheckedChange={(on) => setHours(day, on ? {} : null)} />
              </li>
            );
          })}
        </ul>
        {hoursBad ? (
          <p role="alert" className="cjn-err">
            {t('place.hoursError')}
          </p>
        ) : null}
      </section>

      <SaveHint state={state} />
      {save.isError ? (
        <p className="cjn-err" role="alert">
          {errorText(save.error)}
        </p>
      ) : null}
      <PrimaryButton onClick={() => void next()}>{t('next')}</PrimaryButton>
    </div>
  );
}
