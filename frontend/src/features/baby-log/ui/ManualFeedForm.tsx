'use client';

import { useId, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { type Locale } from '@/shared/i18n';
import { LocaleNumberField, SecondaryButton } from '@/shared/ui';

import { manualStart, tehranNow } from '../model/live';
import { MAX_AMOUNT_ML, MAX_FEED_MINUTES, type FeedType, type ManualFeedInput } from '../model/types';

const clamp = (n: number | undefined, max: number) => (n === undefined ? undefined : Math.min(max, Math.max(0, n)));

/**
 * «ثبت دستی» under the Log_Feed timer: a feed that already happened (start
 * time today, side minutes for the breast, minutes + ml for bottle / pump).
 * Closed by default so the timer stays the one obvious action.
 */
export function ManualFeedForm({
  type,
  onSave,
  pending,
}: {
  type: FeedType;
  onSave: (input: ManualFeedInput) => void;
  pending: boolean;
}) {
  const t = useTranslations('babyLog');
  const loc = useLocale() as Locale;
  const panelId = useId();
  const [open, setOpen] = useState(false);
  const [time, setTime] = useState('');
  const [left, setLeft] = useState<number | undefined>();
  const [right, setRight] = useState<number | undefined>();
  const [minutes, setMinutes] = useState<number | undefined>();
  const [ml, setMl] = useState<number | undefined>();
  const [invalid, setInvalid] = useState(false);

  const save = () => {
    const now = tehranNow();
    const startedAt = manualStart(time || now.time, now);
    const breastEmpty = type === 'breast' && !(left ?? 0) && !(right ?? 0);
    if (!startedAt || breastEmpty) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    onSave(
      type === 'breast'
        ? { type, startedAt, leftMinutes: left ?? 0, rightMinutes: right ?? 0 }
        : { type, startedAt, durationMinutes: minutes ?? 0, amountMl: ml ?? null },
    );
    setLeft(undefined);
    setRight(undefined);
    setMinutes(undefined);
    setMl(undefined);
    setOpen(false);
  };

  return (
    <div className="bfl-manual">
      <button
        type="button"
        className="bfl-link"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((o) => !o)}
      >
        {t('manual.open')}
      </button>
      {open ? (
        <div id={panelId} className="bfl-manual-body nb-card">
          <label className="fld-label">
            <span className="fld-label-t">{t('manual.time')}</span>
            <input className="field fld-input" type="time" value={time} onChange={(e) => setTime(e.target.value)} />
          </label>
          {type === 'breast' ? (
            <div className="bfl-manual-pair">
              <LocaleNumberField
                label={t('manual.left')}
                locale={loc}
                value={left}
                max={MAX_FEED_MINUTES}
                onChange={(v) => setLeft(clamp(v, MAX_FEED_MINUTES))}
              />
              <LocaleNumberField
                label={t('manual.right')}
                locale={loc}
                value={right}
                max={MAX_FEED_MINUTES}
                onChange={(v) => setRight(clamp(v, MAX_FEED_MINUTES))}
              />
            </div>
          ) : (
            <div className="bfl-manual-pair">
              <LocaleNumberField
                label={t('manual.minutes')}
                locale={loc}
                value={minutes}
                max={MAX_FEED_MINUTES}
                onChange={(v) => setMinutes(clamp(v, MAX_FEED_MINUTES))}
              />
              <LocaleNumberField
                label={t('manual.ml')}
                locale={loc}
                value={ml}
                max={MAX_AMOUNT_ML}
                onChange={(v) => setMl(clamp(v, MAX_AMOUNT_ML))}
              />
            </div>
          )}
          {invalid ? (
            <p className="bfl-error" role="alert">
              {t(type === 'breast' ? 'manual.needMinutes' : 'manual.needTime')}
            </p>
          ) : null}
          <SecondaryButton onClick={save} loading={pending}>
            {t('manual.save')}
          </SecondaryButton>
        </div>
      ) : null}
    </div>
  );
}
