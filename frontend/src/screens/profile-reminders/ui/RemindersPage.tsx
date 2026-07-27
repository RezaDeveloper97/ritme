'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useState } from 'react';

import {
  type Reminder,
  type ReminderRecurrence,
  type ReminderType,
  useReminderEnums,
  useReminders,
} from '@/entities/reminder';
import {
  useCreateReminder,
  useDeleteReminder,
  useUpdateReminder,
} from '@/features/manage-reminders';
import { formatJalali } from '@/shared/lib/date';
import { type Locale, useRouter } from '@/shared/i18n';
import { Icon, type IconName, NavBack } from '@/shared/ui';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (value: string, loc: Locale) =>
  loc === 'fa' ? value.replace(/[0-9]/g, (d) => FA[Number(d)]) : value;

/** Row icon per reminder type (visual language mirrors the profile rows). */
const TYPE_ICON: Record<ReminderType, IconName> = {
  doctor: 'stetho',
  medication: 'pill',
  appointment: 'calendar',
  custom: 'bell',
};

const ALL_TYPES: ReminderType[] = ['doctor', 'medication', 'appointment', 'custom'];
const ALL_RECURRENCES: ReminderRecurrence[] = ['none', 'daily', 'weekly', 'monthly'];

// Hairline between rows, inset past the icon (logical start — RTL-safe).
function Divider() {
  return <div className="prof-divider" />;
}

// RTL-safe on/off switch: the knob is positioned with a logical inset, so it
// travels the correct way in both directions (CLAUDE.md §12).
function Toggle({
  on,
  disabled,
  label,
  onToggle,
}: {
  on: boolean;
  disabled?: boolean;
  label: string;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={label}
      onClick={onToggle}
      disabled={disabled}
      style={{
        position: 'relative',
        width: 44,
        height: 26,
        borderRadius: 13,
        border: 0,
        padding: 0,
        flexShrink: 0,
        background: on ? 'var(--brand)' : 'var(--track)',
        cursor: disabled ? 'default' : 'pointer',
        opacity: disabled ? 0.55 : 1,
        transition: 'background .2s ease',
      }}
    >
      <span
        style={{
          position: 'absolute',
          top: 2,
          insetInlineStart: on ? 20 : 2,
          width: 22,
          height: 22,
          borderRadius: '50%',
          background: 'var(--surface)',
          boxShadow: '0 1px 3px rgba(17,32,47,.25)',
          transition: 'inset-inline-start .2s ease',
        }}
      />
    </button>
  );
}

// ── One reminder row ──────────────────────────────────────────
function ReminderRow({
  reminder,
  loc,
  confirming,
  onAskDelete,
  onCancelDelete,
  onConfirmDelete,
  onToggleActive,
  togglePending,
  deletePending,
}: {
  reminder: Reminder;
  loc: Locale;
  confirming: boolean;
  onAskDelete: () => void;
  onCancelDelete: () => void;
  onConfirmDelete: () => void;
  onToggleActive: () => void;
  togglePending: boolean;
  deletePending: boolean;
}) {
  const t = useTranslations('reminders');

  // Second line: the optional subtitle plus a human schedule summary.
  const scheduleText = (() => {
    if (reminder.recurrence !== 'none') {
      const label = t(`recurrence.${reminder.recurrence}`);
      return reminder.recurrenceTime
        ? t('recurrenceAt', {
            recurrence: label,
            time: localizeNum(reminder.recurrenceTime, loc),
          })
        : label;
    }
    // One-off: show the Jalali date when there is one (§7 — never raw Gregorian).
    return reminder.scheduledAt
      ? formatJalali(new Date(reminder.scheduledAt), loc)
      : null;
  })();

  const secondLine = [reminder.subtitle, scheduleText].filter(Boolean).join(' · ');

  return (
    <div>
      <div className="rem-row">
        <span className={clsx('rem-row-icon', !reminder.isActive && 'is-off')}>
          <Icon name={TYPE_ICON[reminder.type]} size={19} />
        </span>

        <div className="rem-row-body">
          <div className={clsx('rem-row-title', !reminder.isActive && 'is-off')}>
            {reminder.title}
          </div>
          {secondLine ? (
            <div className="rem-row-sub">
              {secondLine}
            </div>
          ) : null}
        </div>

        <Toggle
          on={reminder.isActive}
          disabled={togglePending || deletePending}
          label={t('row.toggle')}
          onToggle={onToggleActive}
        />

        <button
          type="button"
          className="iconbtn"
          aria-label={t('row.delete')}
          onClick={confirming ? onCancelDelete : onAskDelete}
          disabled={deletePending}
          style={{ color: 'var(--danger)', flexShrink: 0, opacity: deletePending ? 0.55 : 1 }}
        >
          <Icon name={confirming ? 'x' : 'trash'} size={18} />
        </button>
      </div>

      {confirming ? (
        <div className="rem-confirm">
          <span className="rem-confirm-q">
            {deletePending ? t('confirmDelete.deleting') : t('confirmDelete.question')}
          </span>
          <button
            type="button"
            onClick={onConfirmDelete}
            disabled={deletePending}
            style={{
              border: 0,
              borderRadius: 10,
              padding: '6px 14px',
              fontSize: 13,
              fontWeight: 700,
              font: 'inherit',
              background: 'rgba(229,72,77,.1)',
              color: 'var(--danger)',
              cursor: deletePending ? 'default' : 'pointer',
              opacity: deletePending ? 0.55 : 1,
            }}
          >
            {t('confirmDelete.yes')}
          </button>
          <button
            type="button"
            onClick={onCancelDelete}
            disabled={deletePending}
            style={{
              border: 0,
              borderRadius: 10,
              padding: '6px 14px',
              fontSize: 13,
              fontWeight: 700,
              font: 'inherit',
              background: 'var(--line)',
              color: 'var(--ink)',
              cursor: deletePending ? 'default' : 'pointer',
            }}
          >
            {t('confirmDelete.no')}
          </button>
        </div>
      ) : null}
    </div>
  );
}

// ── Inline "new reminder" form ────────────────────────────────
function AddReminderForm({ onDone }: { onDone: () => void }) {
  const t = useTranslations('reminders');
  const { data: enums } = useReminderEnums();
  const create = useCreateReminder();

  const [type, setType] = useState<ReminderType>('doctor');
  const [title, setTitle] = useState('');
  const [subtitle, setSubtitle] = useState('');
  const [recurrence, setRecurrence] = useState<ReminderRecurrence>('none');
  const [time, setTime] = useState('08:00');
  const [failed, setFailed] = useState(false);

  // Prefer the backend-localized labels from /reminders/enums; fall back to
  // the slice's own i18n keys until (or if) they load.
  const typeOptions =
    enums && enums.types.length > 0
      ? enums.types.map((o) => ({ value: o.value, label: o.label }))
      : ALL_TYPES.map((v) => ({ value: v, label: t(`types.${v}`) }));
  const recurrenceOptions =
    enums && enums.recurrences.length > 0
      ? enums.recurrences.map((o) => ({ value: o.value, label: o.label }))
      : ALL_RECURRENCES.map((v) => ({ value: v, label: t(`recurrence.${v}`) }));

  const canSubmit = title.trim().length > 0 && !create.isPending;

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setFailed(false);
    create.mutate(
      {
        type,
        title: title.trim(),
        subtitle: subtitle.trim() ? subtitle.trim() : undefined,
        recurrence,
        recurrenceTime: recurrence !== 'none' && time ? time : undefined,
      },
      {
        onSuccess: onDone,
        onError: () => setFailed(true),
      },
    );
  };

  const selectStyle = {
    width: '100%',
    border: 0,
    background: 'transparent',
    font: 'inherit',
    fontSize: 15,
    color: 'var(--ink)',
    outline: 'none',
  } as const;

  return (
    <form onSubmit={handleSubmit} className="card rem-form">
      <div className="rem-form-title">
        {t('form.title')}
      </div>

      <label className="lbl">{t('form.typeLabel')}</label>
      <div className="field rem-form-field">
        <select
          value={type}
          onChange={(e) => setType(e.target.value as ReminderType)}
          style={selectStyle}
        >
          {typeOptions.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      <label className="lbl">{t('form.titleLabel')}</label>
      <div className="field rem-form-field">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder={t('form.titlePlaceholder')}
          maxLength={120}
        />
      </div>

      <label className="lbl">{t('form.subtitleLabel')}</label>
      <div className="field rem-form-field">
        <input
          value={subtitle}
          onChange={(e) => setSubtitle(e.target.value)}
          placeholder={t('form.subtitlePlaceholder')}
          maxLength={160}
        />
      </div>

      <label className="lbl">{t('form.recurrenceLabel')}</label>
      <div className="field rem-form-field">
        <select
          value={recurrence}
          onChange={(e) => setRecurrence(e.target.value as ReminderRecurrence)}
          style={selectStyle}
        >
          {recurrenceOptions.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      {recurrence !== 'none' ? (
        <>
          <label className="lbl">{t('form.timeLabel')}</label>
          <div className="field rem-form-field">
            <input type="time" value={time} onChange={(e) => setTime(e.target.value)} />
          </div>
        </>
      ) : null}

      {failed ? (
        <div className="rem-form-error">
          {t('form.error')}
        </div>
      ) : null}

      <div className="rem-form-btns">
        <button
          type="submit"
          className="btn btn-primary rem-form-submit"
          disabled={!canSubmit}
        >
          {create.isPending ? t('form.submitting') : t('form.submit')}
        </button>
        <button
          type="button"
          className="btn btn-soft rem-form-cancel"
          onClick={onDone}
          disabled={create.isPending}
        >
          {t('form.cancel')}
        </button>
      </div>
    </form>
  );
}

// ── The screen ────────────────────────────────────────────────
export function RemindersPage() {
  const t = useTranslations('reminders');
  const loc = useLocale() as Locale;
  const router = useRouter();

  const { data: reminders, isLoading, isError } = useReminders();
  const update = useUpdateReminder();
  const remove = useDeleteReminder();

  const [showForm, setShowForm] = useState(false);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);

  const handleToggle = (r: Reminder) => {
    if (update.isPending) return;
    update.mutate({ id: r.id, isActive: !r.isActive });
  };

  const handleDelete = (id: string) => {
    if (remove.isPending) return;
    remove.mutate(id, { onSettled: () => setConfirmingId(null) });
  };

  return (
    <div className="view rem-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} label={t('back')} />
        <span className="rem-title">{t('title')}</span>
        <button
          type="button"
          className="iconbtn rem-add"
          aria-label={showForm ? t('form.cancel') : t('add')}
          onClick={() => setShowForm((v) => !v)}
        >
          <Icon name={showForm ? 'x' : 'plus'} size={22} />
        </button>
      </div>

      <div className="scroll rem-scroll">
        {showForm ? <AddReminderForm onDone={() => setShowForm(false)} /> : null}

        {isLoading ? (
          <div className="card rem-state">
            {t('loading')}
          </div>
        ) : isError ? (
          <div className="card rem-state">
            {t('loadError')}
          </div>
        ) : reminders && reminders.length > 0 ? (
          <div className="card rem-list">
            {reminders.map((r, i) => (
              <div key={r.id}>
                {i > 0 ? <Divider /> : null}
                <ReminderRow
                  reminder={r}
                  loc={loc}
                  confirming={confirmingId === r.id}
                  onAskDelete={() => setConfirmingId(r.id)}
                  onCancelDelete={() => setConfirmingId(null)}
                  onConfirmDelete={() => handleDelete(r.id)}
                  onToggleActive={() => handleToggle(r)}
                  togglePending={update.isPending && update.variables?.id === r.id}
                  deletePending={remove.isPending && remove.variables === r.id}
                />
              </div>
            ))}
          </div>
        ) : (
          <div className="card rem-empty">
            <span className="rem-empty-icon">
              <Icon name="bell" size={26} />
            </span>
            <div className="rem-empty-title">
              {t('empty.title')}
            </div>
            <div className="rem-empty-sub">
              {t('empty.subtitle')}
            </div>
            <button
              type="button"
              className="btn btn-primary rem-empty-cta"
              onClick={() => setShowForm(true)}
            >
              {t('add')}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
