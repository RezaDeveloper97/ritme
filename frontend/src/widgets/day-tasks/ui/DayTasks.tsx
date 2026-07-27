'use client';

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
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
          <span style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>{t('title')}</span>
          {dayItems.length > 0 && (
            <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, fontWeight: 700, color: 'var(--steel)' }}>
              {t('progress', { done: localizeNum(doneCount, loc), total: localizeNum(dayItems.length, loc) })}
              <Icon name="checkCircle" size={16} stroke="var(--green)" />
            </span>
          )}
        </div>
        <p className="sub" style={{ textAlign: 'start', margin: '0 2px 12px' }}>{t('subtitle')}</p>

        {/* Type picker + text field + add button */}
        <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
          {ADD_TYPES.map((it) => {
            const on = type === it.value;
            return (
              <button
                key={it.value}
                type="button"
                onClick={() => setType(it.value)}
                aria-pressed={on}
                style={{
                  display: 'flex', alignItems: 'center', gap: 5, cursor: 'pointer',
                  borderRadius: 10, padding: '6px 10px', fontFamily: 'inherit', fontSize: 12, fontWeight: 700,
                  border: on ? '1px solid var(--brand)' : '1px solid var(--line)',
                  background: on ? 'var(--surface-2)' : 'var(--surface)',
                  color: on ? 'var(--brand)' : 'var(--steel)',
                }}
              >
                <Icon name={it.icon} size={14} stroke="currentColor" />
                {t(`types.${it.value}`)}
              </button>
            );
          })}
        </div>
        <div style={{ display: 'flex', gap: 8, marginBottom: dayItems.length ? 14 : 4 }}>
          <div className="field" style={{ flex: 1, height: 42 }}>
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
            className="btn btn-primary"
            style={{ width: 'auto', padding: '0 16px', height: 42, opacity: !draft.trim() || create.isPending ? 0.5 : 1 }}
          >
            {create.isPending ? t('adding') : t('add')}
          </button>
        </div>

        {dayItems.length === 0 ? (
          <p style={{ textAlign: 'center', fontSize: 12.5, color: 'var(--muted)', padding: '8px 0 2px' }}>
            {t('empty')}
          </p>
        ) : (
          dayItems.map((r) => {
            const done = !r.isActive;
            return (
              <div key={r.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '9px 2px' }}>
                <span className="dot" style={{ width: 32, height: 32, flex: '0 0 auto', background: 'var(--pink-bg)', color: 'var(--brand)' }}>
                  <Icon name={TYPE_ICON[r.type]} size={16} stroke="currentColor" />
                </span>
                <div style={{ flex: 1, textAlign: 'start', minWidth: 0 }}>
                  <div style={{ fontSize: 14, fontWeight: 600, color: done ? 'var(--muted-soft)' : 'var(--ink)', textDecoration: done ? 'line-through' : undefined, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {r.title}
                  </div>
                  {r.subtitle && (
                    <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>{r.subtitle}</div>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => update.mutate({ id: r.id, isActive: done })}
                  aria-label={t('toggle')}
                  aria-pressed={done}
                  className={`cbx${done ? ' on' : ''}`}
                  style={{ appearance: 'none', WebkitAppearance: 'none', padding: 0, flex: '0 0 auto' }}
                >
                  <Icon name="check" size={14} stroke="var(--on-accent)" />
                </button>
                <button
                  type="button"
                  onClick={() => remove.mutate(r.id)}
                  aria-label={t('delete')}
                  className="iconbtn"
                  style={{ color: 'var(--muted)', flex: '0 0 auto' }}
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
