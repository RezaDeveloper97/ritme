'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useState } from 'react';

import {
  CHECKUP_PERFORMERS,
  CUSTOM_INTERVAL_MONTHS,
  type CheckupDetail,
  useCheckup,
} from '@/entities/checkup';
import {
  useCreateCustomCheckup,
  useDeleteCustomCheckup,
  useUpdateCustomCheckup,
} from '@/features/manage-custom-checkup';
import { getApiErrorStatus, getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import {
  type DateParts,
  formatLongDate,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today,
} from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon } from '@/shared/ui';

import {
  type CustomFormState,
  emptyCustomForm,
  fromDetail,
  NOTE_MAX,
  TITLE_MAX,
  validateCustomForm,
} from '../model/form';

const LIST_HREF = '/checkups';

function Shell({ edit, children }: { edit: boolean; children: ReactNode }) {
  const t = useTranslations('checkups');
  const dir = useDirection();
  return (
    <div className="view rmd-page">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href={LIST_HREF} className="rmd-hdr-btn" aria-label={t('back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{edit ? t('custom.editTitle') : t('custom.newTitle')}</h1>
          </div>
          <span className="rmd-hdr-btn invisible" aria-hidden />
        </header>
        <div className="rmd-body flex flex-col gap-3 pb-8">{children}</div>
      </div>
    </div>
  );
}

/**
 * Custom checkup form — `/checkups/custom/new` and `/checkups/custom/[id]`
 * (edit + delete). Privacy (§11): title and note go to the API only.
 */
export function CustomCheckupFormPage({ id }: { id?: number }) {
  const t = useTranslations('checkups');
  const query = useCheckup(id ?? null);

  if (id === undefined) return <CustomForm />;
  if (query.isPending) {
    return (
      <Shell edit>
        <p className="rmd-state" role="status">
          {t('loading')}
        </p>
      </Shell>
    );
  }
  const notFound = getApiErrorStatus(query.error) === 404 || (query.data && !query.data.isCustom);
  if (query.isError || !query.data || notFound) {
    return (
      <Shell edit>
        <div className="rmd-state" role="alert">
          <p>{notFound ? t('custom.notFound') : t('loadError')}</p>
          {!notFound && (
            <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
              {t('retry')}
            </button>
          )}
        </div>
      </Shell>
    );
  }
  return <CustomForm key={query.data.id} detail={query.data} />;
}

function CustomForm({ detail }: { detail?: CheckupDetail }) {
  const t = useTranslations('checkups');
  const tc = useTranslations('checkups.custom');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const edit = detail !== undefined;

  const [state, setState] = useState<CustomFormState>(() => (detail ? fromDetail(detail) : emptyCustomForm()));
  const [noteDirty, setNoteDirty] = useState(false);
  const [titleError, setTitleError] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [dateOpen, setDateOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [draftDate, setDraftDate] = useState<DateParts | null>(null);

  const create = useCreateCustomCheckup();
  const update = useUpdateCustomCheckup();
  const remove = useDeleteCustomCheckup();
  const saving = create.isPending || update.isPending;

  const set = <K extends keyof CustomFormState>(key: K, value: CustomFormState[K]) =>
    setState((s) => ({ ...s, [key]: value }));
  const back = () => router.push(LIST_HREF);
  // A per-user cap (422 limit_reached) or the write limit (429) shows the server's
  // localized message.
  const onError = (error: unknown) => setFormError(getApiSaveErrorMessage(error, t('saveError')));

  const onSave = () => {
    setFormError(null);
    if (validateCustomForm(state)) {
      setTitleError(true);
      return;
    }
    const input = {
      title: state.title,
      intervalMonths: state.intervalMonths,
      performedBy: state.performedBy,
      lastDoneOn: state.lastDoneOn,
    };
    if (detail) {
      // The detail doesn't carry the note, so only send it when the user wrote one.
      const patch = noteDirty ? { ...input, note: state.note } : input;
      update.mutate({ id: detail.id, patch }, { onSuccess: back, onError });
    } else {
      create.mutate({ ...input, note: state.note }, { onSuccess: back, onError });
    }
  };

  const openDate = () => {
    setDraftDate(toParts(state.lastDoneOn ? fromApiDate(state.lastDoneOn) : today(), locale));
    setDateOpen(true);
  };

  return (
    <Shell edit={edit}>
      <section className="card fld-card flex flex-col gap-3.5">
        <label className="fld-label">
          <span className="fld-label-t">{tc('name')}</span>
          <span className="field">
            <Icon name="stetho" size={18} />
            <input
              value={state.title}
              maxLength={TITLE_MAX}
              placeholder={tc('namePlaceholder')}
              aria-invalid={titleError}
              aria-describedby={titleError ? 'ck-err-title' : undefined}
              onChange={(e) => {
                set('title', e.target.value);
                setTitleError(false);
              }}
            />
          </span>
          {titleError && (
            <p id="ck-err-title" role="alert" className="mt-1.5 text-start text-[12px] font-bold text-(--danger-deep)">
              {tc('titleRequired')}
            </p>
          )}
        </label>

        <div className="flex flex-col gap-2">
          <span className="fld-label-t text-start">{tc('interval')}</span>
          <div role="radiogroup" aria-label={tc('interval')} className="fld-chips">
            {CUSTOM_INTERVAL_MONTHS.map((m) => (
              <button
                key={m}
                type="button"
                role="radio"
                aria-checked={state.intervalMonths === m}
                className={clsx('chip', state.intervalMonths === m && 'on')}
                onClick={() => set('intervalMonths', m)}
              >
                {tc(`intervals.m${m}`)}
              </button>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <span className="fld-label-t text-start">{tc('performedBy')}</span>
          <div role="radiogroup" aria-label={tc('performedBy')} className="fld-chips">
            {CHECKUP_PERFORMERS.map((p) => (
              <button
                key={p}
                type="button"
                role="radio"
                aria-checked={state.performedBy === p}
                className={clsx('chip', state.performedBy === p && 'on')}
                onClick={() => set('performedBy', p)}
              >
                {t(`performedBy.${p}`)}
              </button>
            ))}
          </div>
        </div>

        <div className="fld-row">
          <span className="fld-row-label text-(--ink)">
            {tc('lastDone')} <span className="text-(--ink-3)">{tc('optional')}</span>
          </span>
          <span className="flex items-center gap-1.5">
            <button type="button" className={clsx('chip', state.lastDoneOn && 'on')} onClick={openDate}>
              {state.lastDoneOn ? formatLongDate(fromApiDate(state.lastDoneOn), locale) : tc('pickDate')}
            </button>
            {state.lastDoneOn && (
              <button
                type="button"
                className="iconbtn grid size-8 place-items-center rounded-full bg-(--surface-2) text-(--ink-3)"
                aria-label={tc('clearDate')}
                onClick={() => set('lastDoneOn', null)}
              >
                <Icon name="x" size={14} />
              </button>
            )}
          </span>
        </div>

        <label className="fld-label">
          <span className="fld-label-t">
            {tc('note')} <span className="text-(--ink-3)">{tc('optional')}</span>
          </span>
          <textarea
            className="field fld-textarea h-auto min-h-20 py-3"
            rows={2}
            maxLength={NOTE_MAX}
            value={state.note}
            placeholder={tc('notePlaceholder')}
            onChange={(e) => {
              set('note', e.target.value);
              setNoteDirty(true);
            }}
          />
        </label>
      </section>

      {formError && (
        <p role="alert" className="text-center text-[12.5px] font-bold text-(--danger-deep)">
          {formError}
        </p>
      )}

      <button type="button" className="btn btn-primary w-full" disabled={saving} onClick={onSave}>
        {saving ? tc('saving') : tc('save')}
      </button>
      {edit && (
        <button
          type="button"
          className="btn w-full bg-transparent text-(--danger-deep)"
          onClick={() => setDeleteOpen(true)}
        >
          <Icon name="trash" size={16} />
          {tc('delete')}
        </button>
      )}

      <AppSheet
        open={dateOpen}
        onClose={() => setDateOpen(false)}
        size="half"
        title={tc('lastDone')}
        footer={
          <div className="flex gap-2.5">
            <button type="button" className="btn btn-ghost flex-1" onClick={() => setDateOpen(false)}>
              {tc('cancel')}
            </button>
            <button
              type="button"
              className="btn btn-primary flex-1"
              disabled={!draftDate}
              onClick={() => {
                if (draftDate) set('lastDoneOn', toApiDate(partsToDate(draftDate, locale)));
                setDateOpen(false);
              }}
            >
              {tc('done')}
            </button>
          </div>
        }
      >
        <CalendarPicker value={draftDate} onSelect={setDraftDate} />
      </AppSheet>

      {detail && (
        <AppSheet
          open={deleteOpen}
          onClose={() => setDeleteOpen(false)}
          size="half"
          title={tc('delete')}
          footer={
            <div className="flex gap-2.5">
              <button type="button" className="btn btn-ghost flex-1" onClick={() => setDeleteOpen(false)}>
                {tc('cancel')}
              </button>
              <button
                type="button"
                className="btn btn-primary flex-1"
                disabled={remove.isPending}
                onClick={() => remove.mutate(detail.id, { onSuccess: back })}
              >
                {remove.isPending ? tc('deleting') : tc('delete')}
              </button>
            </div>
          }
        >
          <p className="text-start text-[13.5px] text-(--ink)">{tc('deleteConfirm')}</p>
          {remove.isError && (
            <p role="alert" className="mt-2 text-start text-[12px] font-bold text-(--danger-deep)">
              {getApiSaveErrorMessage(remove.error, t('saveError'))}
            </p>
          )}
        </AppSheet>
      )}
    </Shell>
  );
}
