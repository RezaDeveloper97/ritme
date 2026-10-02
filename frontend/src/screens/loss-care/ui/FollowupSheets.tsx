'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import {
  type LossFollowup,
  lossFieldError,
  lossNoteProblem,
  splitWallClock,
  useLossNote,
  useSaveLossFollowup,
  useSaveLossNote,
  visitAt,
} from '@/entities/loss';
import { getApiSaveErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import {
  type DateParts,
  diffInDays,
  formatNumber,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today,
} from '@/shared/lib/date';
import { AppSheet, openSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  ChipGroup,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
} from '@/shared/ui';

/** Max length of the private note (CB-LOSS-01 `MaxNoteLen`). */
const MAX_NOTE = 5000;

/** Visit times offered as chips (Tehran wall clock). */
const VISIT_TIMES = ['09:00', '10:30', '12:00', '14:00', '16:00', '18:00'] as const;

function SheetError({ message }: { message: string | null | undefined }) {
  return message ? (
    <p className="lsc-error" role="alert">
      {message}
    </p>
  ) : null;
}

/** Bleeding: log today's bleeding in the log sheet, or mark it stopped. */
export function BleedingSheet({ onClose }: { onClose: () => void }) {
  const t = useTranslations('loss.care.bleeding');
  const tc = useTranslations('loss.common');
  const save = useSaveLossFollowup();
  return (
    <AppSheet open onClose={onClose} size="half" title={t('sheetTitle')}>
      <div className="lsc-sheet">
        <p className="lsc-sheet-body">{t('sheetBody')}</p>
        <PrimaryButton
          icon="drop"
          onClick={() => {
            onClose();
            openSheet('log');
          }}
        >
          {t('logToday')}
        </PrimaryButton>
        <SecondaryButton
          icon="check"
          loading={save.isPending}
          onClick={() => save.mutate({ bleedingStopped: true }, { onSuccess: onClose })}
        >
          {t('markStopped')}
        </SecondaryButton>
        <SheetError message={save.isError ? (lossFieldError(save.error) ?? getApiSaveErrorMessage(save.error, tc('saveError'))) : null} />
      </div>
    </AppSheet>
  );
}

function useFutureDay(initial: string | null, locale: Locale) {
  const [parts, setParts] = useState<DateParts | null>(initial ? toParts(fromApiDate(initial), locale) : null);
  const picked = parts ? partsToDate(parts, locale) : null;
  const past = picked ? diffInDays(picked, today()) < 0 : false;
  return { parts, setParts, picked, past };
}

/** Beta hCG: the next test day (a private reminder the day before), or «جواب منفی شد». */
export function BetaSheet({ beta, onClose }: { beta: LossFollowup['beta']; onClose: () => void }) {
  const t = useTranslations('loss.care.beta');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const save = useSaveLossFollowup();
  const day = useFutureDay(beta.nextOn, locale);
  const error = save.isError ? (lossFieldError(save.error) ?? getApiSaveErrorMessage(save.error, tc('saveError'))) : null;
  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={t('sheetTitle')}
      footer={
        <div className="lsc-sheet-btns">
          <PrimaryButton
            loading={save.isPending}
            disabled={!day.picked || day.past}
            onClick={() => day.picked && save.mutate({ betaNextOn: toApiDate(day.picked) }, { onSuccess: onClose })}
          >
            {t('setReminder')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="lsc-sheet">
        <p className="lsc-sheet-body">{t('sheetBody')}</p>
        <CalendarPicker value={day.parts} onSelect={day.setParts} />
        <SheetError message={day.past ? t('datePast') : error} />
        <div className="lsc-sheet-actions">
          <SecondaryButton
            icon="check"
            disabled={save.isPending}
            onClick={() => save.mutate({ betaNegative: true }, { onSuccess: onClose })}
          >
            {t('markNegative')}
          </SecondaryButton>
          {beta.nextOn ? (
            <SecondaryButton
              variant="text"
              disabled={save.isPending}
              onClick={() => save.mutate({ betaNextOn: null }, { onSuccess: onClose })}
            >
              {t('clearReminder')}
            </SecondaryButton>
          ) : null}
        </div>
      </div>
    </AppSheet>
  );
}

/** Follow-up visit: day + time → a private care appointment (`visit_at`, Tehran wall clock). */
export function VisitSheet({ visit, onClose }: { visit: LossFollowup['visit']; onClose: () => void }) {
  const t = useTranslations('loss.care.visit');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const save = useSaveLossFollowup();
  const booked = visit.appointment ? splitWallClock(visit.appointment.scheduledAt) : null;
  const day = useFutureDay(booked?.day ?? visit.suggestedOn, locale);
  const [time, setTime] = useState<string | null>(booked?.time ?? null);
  const times: string[] = booked && !VISIT_TIMES.includes(booked.time as never) ? [booked.time, ...VISIT_TIMES] : [...VISIT_TIMES];
  const error = save.isError ? (lossFieldError(save.error) ?? getApiSaveErrorMessage(save.error, tc('saveError'))) : null;
  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={t('sheetTitle')}
      footer={
        <div className="lsc-sheet-btns">
          <PrimaryButton
            loading={save.isPending}
            disabled={!day.picked || day.past || !time}
            onClick={() =>
              day.picked && time && save.mutate({ visitAt: visitAt(toApiDate(day.picked), time) }, { onSuccess: onClose })
            }
          >
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="lsc-sheet">
        <p className="lsc-sheet-body">{t('sheetBody')}</p>
        <CalendarPicker value={day.parts} onSelect={day.setParts} />
        <p className="lsc-sheet-label">{t('time')}</p>
        <ChipGroup label={t('time')} className="lsc-times">
          {times.map((value) => (
            <PillChip key={value} pressed={time === value} onPressedChange={() => setTime(value)} dir="ltr">
              {formatNumber(value, locale)}
            </PillChip>
          ))}
        </ChipGroup>
        <SheetError message={day.past ? t('timePast') : error} />
        {booked ? (
          <SecondaryButton
            variant="text"
            disabled={save.isPending}
            onClick={() => save.mutate({ visitAt: null }, { onSuccess: onClose })}
          >
            {t('clear')}
          </SecondaryButton>
        ) : null}
      </div>
    </AppSheet>
  );
}

/**
 * The private note: fetched only while this sheet is open and dropped from the
 * cache when it closes. 503 `note_unavailable` (no server key) shows a calm
 * «not available right now» with nothing to type into; 409 offers to delete.
 */
export function NoteSheet({ onClose }: { onClose: () => void }) {
  const t = useTranslations('loss.care.note');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const ids = useId();
  const note = useLossNote(true);
  const save = useSaveLossNote();
  const [text, setText] = useState('');
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (note.isSuccess && !loaded) {
      setText(note.data.note ?? '');
      setLoaded(true);
    }
  }, [note.isSuccess, note.data, loaded]);

  const problem = note.isError ? lossNoteProblem(note.error) : save.isError ? lossNoteProblem(save.error) : null;
  const tooLong = text.length > MAX_NOTE;
  const hasSaved = !!note.data?.note;

  let body;
  if (note.isPending) {
    body = (
      <SkeletonGroup label={tc('loading')}>
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (problem === 'unavailable') {
    body = <p className="lsc-sheet-body">{t('unavailable')}</p>;
  } else if (note.isError && problem === 'unreadable') {
    body = (
      <>
        <p className="lsc-sheet-body">{t('unreadable')}</p>
        <SecondaryButton icon="trash" loading={save.isPending} onClick={() => save.mutate(null, { onSuccess: onClose })}>
          {t('delete')}
        </SecondaryButton>
      </>
    );
  } else if (note.isError) {
    body = (
      <>
        <p className="lsc-sheet-body">{tc('loadError')}</p>
        <SecondaryButton icon="refresh" loading={note.isFetching} onClick={() => void note.refetch()}>
          {tc('retry')}
        </SecondaryButton>
      </>
    );
  } else {
    body = (
      <>
        <label htmlFor={`${ids}-note`} className="sr-only">
          {t('label')}
        </label>
        <textarea
          id={`${ids}-note`}
          rows={8}
          value={text}
          placeholder={t('placeholder')}
          onChange={(e) => setText(e.target.value)}
          className="lsc-note"
          aria-describedby={`${ids}-privacy`}
        />
        <p id={`${ids}-privacy`} className="lsc-sheet-hint">
          {t('privacy')}
        </p>
        <SheetError
          message={
            tooLong
              ? t('tooLong', { max: formatNumber(MAX_NOTE, locale) })
              : save.isError
                ? (lossFieldError(save.error) ?? getApiSaveErrorMessage(save.error, tc('saveError')))
                : null
          }
        />
        {hasSaved ? (
          <SecondaryButton
            variant="text"
            icon="trash"
            disabled={save.isPending}
            onClick={() => save.mutate(null, { onSuccess: onClose })}
          >
            {t('delete')}
          </SecondaryButton>
        ) : null}
      </>
    );
  }

  const editable = note.isSuccess && problem !== 'unavailable';
  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={t('sheetTitle')}
      footer={
        editable ? (
          <div className="lsc-sheet-btns">
            <PrimaryButton
              loading={save.isPending}
              disabled={tooLong || (!text.trim() && !hasSaved)}
              onClick={() => save.mutate(text.trim() ? text : null, { onSuccess: onClose })}
            >
              {t('save')}
            </PrimaryButton>
          </div>
        ) : undefined
      }
    >
      <div className="lsc-sheet">{body}</div>
    </AppSheet>
  );
}
