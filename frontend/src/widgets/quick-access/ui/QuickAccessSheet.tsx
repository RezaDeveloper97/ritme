'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { closeSheet } from '@/shared/sheet';
import { Checkbox, Icon, PrimaryButton } from '@/shared/ui';

import { availableShortcuts, QUICK_ACCESS_MAX } from '../model/shortcuts';
import { useQuickAccessStore } from '../model/store';
import { useQuickAccess } from '../model/use-quick-access';

export function QuickAccessSheetTitle() {
  return <>{useTranslations('nav.quickAccess')('editTitle')}</>;
}

/**
 * `?sheet=quick-access` — pick up to {@link QUICK_ACCESS_MAX} Today shortcuts (CB-NAV-03). Starts from what
 * the row shows now; newly ticked ones go to the end. «برگشت به پیش‌فرض» forgets the picks.
 */
export function QuickAccessSheet() {
  const qa = useQuickAccess();
  if (!qa.mode || !qa.owner) return null;
  // Keyed on what the row shows, so a late account/mode load re-seeds the draft.
  return <Editor key={`${qa.owner}:${qa.mode}`} />;
}

function Editor() {
  const t = useTranslations('nav.quickAccess');
  const locale = useLocale() as Locale;
  const qa = useQuickAccess();
  const save = useQuickAccessStore((s) => s.save);
  const [picked, setPicked] = useState<string[]>(() => qa.visible.map((s) => s.id));
  if (!qa.mode || !qa.owner) return null;
  const owner = qa.owner;
  const options = availableShortcuts(qa.mode);
  const full = picked.length >= QUICK_ACCESS_MAX;

  const toggle = (id: string, on: boolean) =>
    setPicked((prev) => (on ? (prev.includes(id) ? prev : [...prev, id]) : prev.filter((x) => x !== id)));

  return (
    <div className="qas">
      <p className="qas-lead">{t('editLead', { max: formatNumber(QUICK_ACCESS_MAX, locale) })}</p>
      <p className="qas-count" aria-live="polite">
        {t('count', { count: formatNumber(picked.length, locale), max: formatNumber(QUICK_ACCESS_MAX, locale) })}
      </p>
      <ul className="qas-list">
        {options.map((s) => {
          const on = picked.includes(s.id);
          return (
            <li key={s.id} className="qas-item">
              <span className={clsx('qac-ic is-sm', `nb-tone-${s.tone}`)} aria-hidden>
                <Icon name={s.icon} size={18} strokeWidth={1.8} />
              </span>
              <Checkbox
                className="qas-check"
                checked={on}
                disabled={!on && full}
                onCheckedChange={(next) => toggle(s.id, next)}
                label={t(`items.${s.id}` as 'items.record')}
              />
            </li>
          );
        })}
      </ul>
      <div className="qas-actions">
        <PrimaryButton
          onClick={() => {
            save(owner, picked);
            closeSheet();
          }}
        >
          {t('save')}
        </PrimaryButton>
        <button
          type="button"
          className="qas-reset"
          onClick={() => {
            save(owner, null);
            closeSheet();
          }}
        >
          {t('reset')}
        </button>
      </div>
    </div>
  );
}
