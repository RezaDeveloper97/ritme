'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import { childFieldError } from '@/entities/child';
import { getApiErrorStatus } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { type DateParts, formatLongDate, fromApiDate, partsToDate, toApiDate, today, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { isReadOnlyError, useVaccineWrite } from '../api/queries';
import { defaultGivenOn } from '../model/schedule';
import type { MarkTarget } from '../model/types';

/** Server cap on a dose note (backend-go `MaxNoteLen`). */
const MAX_NOTE = 500;

interface Props {
  childId: number;
  birthDate: string;
  /** What is being recorded; null = closed. */
  target: MarkTarget | null;
  onClose: () => void;
}

/**
 * «تزریق شد» (v14_MarkDone pattern): the date the shot(s) were given — the
 * due date for a late entry, else today — and an optional note (reactions,
 * site). A visit records all of its doses, a dose only itself.
 */
export function MarkGivenSheet({ childId, birthDate, target, onClose }: Props) {
  const t = useTranslations('children');
  const locale = useLocale() as Locale;
  const write = useVaccineWrite(childId);
  const noteId = useId();
  const [givenOn, setGivenOn] = useState('');
  const [note, setNote] = useState('');
  const [dateOpen, setDateOpen] = useState(false);

  useEffect(() => {
    if (!target) return;
    setGivenOn(defaultGivenOn(target.visit, toApiDate(today()), birthDate));
    setNote(target.kind === 'dose' ? (target.dose.note ?? '') : '');
    setDateOpen(false);
    write.reset();
    // Only when a new target opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target]);

  const onPick = (parts: DateParts) => {
    const iso = toApiDate(partsToDate(parts, locale));
    const now = toApiDate(today());
    setGivenOn(iso > now ? now : iso < birthDate ? birthDate : iso);
    setDateOpen(false);
  };

  const submit = () => {
    if (!target || !givenOn) return;
    const code = target.kind === 'visit' ? target.visit.code : target.dose.code;
    write.mutate({ kind: target.kind, code, givenOn, note }, { onSuccess: onClose });
  };

  const dateError = childFieldError(write.error, 'given_on');
  const noteError = childFieldError(write.error, 'note');
  const failed = write.isError && getApiErrorStatus(write.error) !== 422;
  const title = !target
    ? ''
    : target.kind === 'visit'
      ? t('vaccines.sheet.titleVisit', { label: target.visit.label })
      : t('vaccines.sheet.titleDose', { title: target.dose.title });

  return (
    <AppSheet
      open={target !== null}
      onClose={() => (write.isPending ? undefined : onClose())}
      size="half"
      title={title}
      footer={
        <div className="chd-sheet-btns">
          <SecondaryButton onClick={onClose} disabled={write.isPending}>
            {t('vaccines.sheet.cancel')}
          </SecondaryButton>
          <PrimaryButton loading={write.isPending} onClick={submit}>
            {t('vaccines.sheet.save')}
          </PrimaryButton>
        </div>
      }
    >
      {target ? (
        <div className="cvx-form">
          {target.kind === 'visit' && target.visit.doses.length > 0 ? (
            <div className="cvx-form-doses">
              <span className="chd-label">{t('vaccines.sheet.doses')}</span>
              <ul className="cvx-form-list">
                {target.visit.doses.map((d) => (
                  <li key={d.code}>
                    <Icon name="syringe" size={14} />
                    {d.title}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}

          <div className="chd-field">
            <span className="chd-label" id="cvx-date-label">
              {t('vaccines.sheet.date')}
            </span>
            <button
              type="button"
              className="chd-date-btn"
              aria-expanded={dateOpen}
              aria-labelledby="cvx-date-label cvx-date-value"
              onClick={() => setDateOpen((o) => !o)}
            >
              <span id="cvx-date-value" className="chd-date-value">
                {givenOn ? formatLongDate(fromApiDate(givenOn), locale) : ''}
              </span>
              <Icon name="calendar" size={20} />
            </button>
            {dateOpen && givenOn ? <CalendarPicker value={toParts(fromApiDate(givenOn), locale)} onSelect={onPick} /> : null}
            {dateError ? (
              <p className="chd-error" role="alert">
                {dateError}
              </p>
            ) : null}
          </div>

          <div className="chd-field">
            <label htmlFor={noteId} className="chd-label">
              {t('vaccines.sheet.note')}
            </label>
            <textarea
              id={noteId}
              className="cvx-textarea"
              rows={3}
              maxLength={MAX_NOTE}
              value={note}
              placeholder={t('vaccines.sheet.notePlaceholder')}
              aria-invalid={noteError ? true : undefined}
              onChange={(e) => setNote(e.target.value)}
            />
            {noteError ? (
              <p className="chd-error" role="alert">
                {noteError}
              </p>
            ) : null}
          </div>

          {failed ? (
            <p className="chd-error" role="alert">
              {isReadOnlyError(write.error) ? t('growth.form.readOnly') : t('vaccines.sheet.error')}
            </p>
          ) : null}
        </div>
      ) : null}
    </AppSheet>
  );
}
