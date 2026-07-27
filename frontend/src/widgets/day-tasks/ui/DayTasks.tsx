'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { useReminders, type Reminder, type ReminderType } from '@/entities/reminder';
import {
  useCreateReminder,
  useDeleteReminder,
  useUpdateReminder,
} from '@/features/manage-reminders';
import type { Locale } from '@/shared/i18n';
import { toApiDate } from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (n: number, loc: Locale) =>
  loc === 'fa' ? String(n).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(n);

// The three kinds of item the day planner offers: a plain to-do plus the two
// health reminders the home page surfaces (doctor / medication). Icons mirror
// the reminder rows elsewhere in the app so a "دارو" looks the same everywhere.
const ADD_TYPES: { value: Extract<ReminderType, 'custom' | 'doctor' | 'medication'>; icon: IconName }[] = [
  { value: 'custom', icon: 'pencil' },
  { value: 'doctor', icon: 'stetho' },
  { value: 'medication', icon: 'pill' },
];

const TYPE_ICON: Record<ReminderType, IconName> = {
  custom: 'pencil',
  doctor: 'stetho',
  medication: 'pill',
  appointment: 'calendar',
};

/** The Gregorian date part ("YYYY-MM-DD") of a reminder's one-off schedule. */
function scheduledDay(reminder: Reminder): string | null {
  return reminder.scheduledAt ? reminder.scheduledAt.slice(0, 10) : null;
}

/**
 * A day-scoped planner block: the doctor/medication reminders and to-dos the
 * user has set for one specific day, with inline add, done-toggle, and delete.
 * "Done" maps to the reminder's `is_active` flag (the API has no separate
 * completed state) — an inactive reminder reads as done and won't notify.
 *
 * The same widget powers the daily-log screen (the day being edited) and the
 * home page (today), so a task added on either surface shows on the other.
 */
export function DayTasks({ date }: { date: Date }) {
  const t = useTranslations('dayTasks');
  const loc = useLocale() as Locale;
  const apiDate = useMemo(() => toApiDate(date), [date]);

  const { data: reminders = [] } = useReminders();
  const create = useCreateReminder();
  const update = useUpdateReminder();
  const remove = useDeleteReminder();

  const [draft, setDraft] = useState('');
  const [type, setType] = useState<ReminderType>('custom');

  // Only the one-off items scheduled for this exact day (§ day-scoped request).
  const dayItems = useMemo(
    () => reminders.filter((r) => scheduledDay(r) === apiDate),
    [reminders, apiDate],
  );
  const doneCount = dayItems.filter((r) => !r.isActive).length;

  const submit = () => {
    const title = draft.trim();
    if (!title || create.isPending) return;
    create.mutate(
      { type, title, scheduledAt: apiDate },
      { onSuccess: () => setDraft('') },
    );
  };

  return (
    <div className="sec-tight">
      <div className="card pad-card-sm">
        <div className="dt-head">
          <span className="dt-title">{t('title')}</span>
          {dayItems.length > 0 && (
            <span className="dt-progress">
              {t('progress', { done: localizeNum(doneCount, loc), total: localizeNum(dayItems.length, loc) })}
              <Icon name="checkCircle" size={16} stroke="var(--green)" />
            </span>
          )}
        </div>
        <p className="sub dt-sub">{t('subtitle')}</p>

        {/* Type picker + text field + add button */}
        <div className="dt-types">
          {ADD_TYPES.map((it) => {
            const on = type === it.value;
            return (
              <button
                key={it.value}
                type="button"
                onClick={() => setType(it.value)}
                aria-pressed={on}
                className="dt-type"
              >
                <Icon name={it.icon} size={14} stroke="currentColor" />
                {t(`types.${it.value}`)}
              </button>
            );
          })}
        </div>
        <div className={clsx('dt-add-row', dayItems.length > 0 && 'has-items')}>
          <div className="field dt-add-field">
            <input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') submit(); }}
              placeholder={t('placeholder')}
              maxLength={120}
            />
          </div>
          <button
            type="button"
            onClick={submit}
            disabled={!draft.trim() || create.isPending}
            className="btn btn-primary dt-add-btn"
          >
            {create.isPending ? t('adding') : t('add')}
          </button>
        </div>

        {dayItems.length === 0 ? (
          <p className="dt-empty">
            {t('empty')}
          </p>
        ) : (
          dayItems.map((r) => {
            const done = !r.isActive;
            return (
              <div key={r.id} className="dt-row">
                <span className="dot dt-row-dot">
                  <Icon name={TYPE_ICON[r.type]} size={16} stroke="currentColor" />
                </span>
                <div className="dt-row-body">
                  <div className={clsx('dt-row-t', done && 'is-done')}>
                    {r.title}
                  </div>
                  {r.subtitle && (
                    <div className="dt-row-s">{r.subtitle}</div>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => update.mutate({ id: r.id, isActive: done })}
                  aria-label={t('toggle')}
                  aria-pressed={done}
                  className={clsx('cbx', 'dt-check', done && 'on')}
                >
                  <Icon name="check" size={14} stroke="var(--on-accent)" />
                </button>
                <button
                  type="button"
                  onClick={() => remove.mutate(r.id)}
                  aria-label={t('delete')}
                  className="iconbtn dt-del"
                >
                  <Icon name="trash" size={16} stroke="currentColor" />
                </button>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
