'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMemo, useRef, useState, type ChangeEvent } from 'react';

import { courseKeys, useCourses } from '@/entities/course';
import { studentKeys, useGroups } from '@/entities/student';
import { isApiError } from '@/shared/api';
import { useErrorMessage } from '@/shared/i18n';
import { cn, toPersianDigits } from '@/shared/lib';
import { Button, Icon, QueryError, Skeleton, TextField } from '@/shared/ui';

import { sendGrants, type GrantProgress } from '../api/grant-api';
import { DURATIONS, untilRange, untilValid, type DurationChoice } from '../model/duration';
import { MAX_PER_REQUEST, chunk, mergePhones, parseCsv, parsePhones } from '../model/phones';

export type Scope = 'group' | 'course' | 'standalone';
const SCOPES: Scope[] = ['group', 'course', 'standalone'];
const MAX_FILE_BYTES = 1024 * 1024;

interface Props {
  initialScope?: Scope;
  initialTarget?: number | null;
}

/** Ins_AddUser: paste numbers or load a CSV, pick the target and duration, preview, send. */
export function AddStudentsForm({ initialScope = 'group', initialTarget = null }: Props) {
  const t = useTranslations('addStudents');
  const errorMessage = useErrorMessage();
  const qc = useQueryClient();
  const groups = useGroups();
  const courses = useCourses();
  const fileRef = useRef<HTMLInputElement>(null);

  const [text, setText] = useState('');
  const [fileResult, setFileResult] = useState<{ name: string; parsed: ReturnType<typeof parseCsv> } | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [scope, setScope] = useState<Scope>(initialScope);
  const [target, setTarget] = useState<number | null>(initialTarget);
  const [duration, setDuration] = useState<DurationChoice>('unlimited');
  const [until, setUntil] = useState('');
  const [touched, setTouched] = useState(false);

  const typed = useMemo(() => parsePhones(text), [text]);
  const parsed = useMemo(() => (fileResult ? mergePhones(typed, fileResult.parsed) : typed), [typed, fileResult]);
  const range = untilRange();

  const options = useMemo(() => {
    if (scope === 'group') return (groups.data?.groups ?? []).map((g) => ({ id: g.id, label: g.name }));
    const kind = scope === 'course' ? 'course' : 'standalone';
    return (courses.data?.courses ?? [])
      .filter((c) => c.kind === kind && c.status === 'published')
      .map((c) => ({ id: c.id, label: c.title }));
  }, [scope, groups.data, courses.data]);
  const selected = options.find((o) => o.id === target) ?? null;

  const mutation = useMutation({
    mutationFn: () =>
      sendGrants(parsed.valid, { scope: scope === 'group' ? 'group' : 'course', targetId: target!, duration, until }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: studentKeys.all });
      void qc.invalidateQueries({ queryKey: courseKeys.all });
      setText('');
      setFileResult(null);
      setTouched(false);
    },
  });

  const requests = chunk(parsed.valid).length;
  const problems = {
    phones: parsed.valid.length === 0,
    target: selected === null,
    until: duration === 'until' && !untilValid(until),
  };

  function submit() {
    setTouched(true);
    if (problems.phones || problems.target || problems.until) return;
    mutation.mutate();
  }

  async function onFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;
    setFileError(null);
    if (file.size > MAX_FILE_BYTES) return setFileError(t('fileTooBig'));
    if (!/\.(csv|txt)$/i.test(file.name) && !file.type.startsWith('text/')) return setFileError(t('fileType'));
    const parsedFile = parseCsv(await file.text());
    if (parsedFile.valid.length + parsedFile.invalid.length === 0) return setFileError(t('fileEmpty'));
    setFileResult({ name: file.name, parsed: parsedFile });
  }

  function pickScope(next: Scope) {
    setScope(next);
    setTarget(null);
  }

  const done = mutation.isSuccess ? mutation.data : null;
  const partial = mutation.isError ? ((mutation.error as { progress?: GrantProgress }).progress ?? null) : null;
  const picking = scope === 'group' ? groups : courses;
  const f = (n: number) => toPersianDigits(n);

  return (
    <div className="page page--narrow add-users">
      <section className="panel-card form-card">
        <TextField
          multiline
          label={t('phonesLabel')}
          hint={t('phonesHint')}
          placeholder={'0912 345 6789\n0935 111 2233'}
          dir="ltr"
          value={text}
          rows={5}
          onChange={(e) => setText(e.target.value)}
          error={touched && problems.phones ? t('phonesRequired') : undefined}
        />

        <input ref={fileRef} type="file" accept=".csv,.txt,text/csv,text/plain" hidden onChange={(e) => void onFile(e)} />
        <button type="button" className="dropzone dropzone--sm" onClick={() => fileRef.current?.click()}>
          <span className="dropzone__icon">
            <Icon name="upload" size={22} />
          </span>
          <span className="dropzone__title">{fileResult ? fileResult.name : t('uploadFile')}</span>
          <span className="dropzone__hint">{t('uploadHint', { max: f(MAX_PER_REQUEST) })}</span>
        </button>
        {fileResult ? (
          <Button variant="text" onClick={() => setFileResult(null)}>
            {t('removeFile')}
          </Button>
        ) : null}
        {fileError ? <p className="form-error">{fileError}</p> : null}

        {parsed.valid.length + parsed.invalid.length > 0 ? (
          <div className="preview-box" aria-live="polite">
            <div className="chips">
              <span className="access-chip access-chip--data">{t('validCount', { count: f(parsed.valid.length) })}</span>
              {parsed.duplicates > 0 ? (
                <span className="access-chip access-chip--brand">{t('dupCount', { count: f(parsed.duplicates) })}</span>
              ) : null}
              {parsed.invalid.length > 0 ? (
                <span className="access-chip access-chip--warm">{t('invalidCount', { count: f(parsed.invalid.length) })}</span>
              ) : null}
            </div>
            {parsed.invalid.length > 0 ? (
              <p className="field__hint" dir="auto">
                {t('invalidList', { list: parsed.invalid.slice(0, 5).map(toPersianDigits).join('، ') })}
              </p>
            ) : null}
            {requests > 1 ? <p className="field__hint">{t('chunks', { count: f(requests), max: f(MAX_PER_REQUEST) })}</p> : null}
          </div>
        ) : null}
      </section>

      <section className="panel-card form-card">
        <h2 className="panel-card__title">{t('accessTitle')}</h2>
        <div className="choice-row" role="radiogroup" aria-label={t('accessTitle')}>
          {SCOPES.map((s) => (
            <button
              key={s}
              type="button"
              role="radio"
              aria-checked={scope === s}
              className={cn('choice', scope === s && 'is-active')}
              onClick={() => pickScope(s)}
            >
              {t(`scope.${s}`)}
            </button>
          ))}
        </div>
        {picking.isPending ? (
          <Skeleton className="skeleton--line" />
        ) : picking.isError ? (
          <QueryError error={picking.error} onRetry={() => void picking.refetch()} />
        ) : options.length === 0 ? (
          <p className="panel-card__empty">{t(`noTargets.${scope}`)}</p>
        ) : (
          <div className="field">
            <label className="field__label" htmlFor="grant-target">
              {t(`targetLabel.${scope}`)}
            </label>
            <span className="select">
              <select
                id="grant-target"
                className="select__control"
                value={target ?? ''}
                onChange={(e) => setTarget(e.target.value ? Number(e.target.value) : null)}
              >
                <option value="">{t('targetPlaceholder')}</option>
                {options.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.label}
                  </option>
                ))}
              </select>
            </span>
            {touched && problems.target ? <p className="field__error">{t('targetRequired')}</p> : null}
          </div>
        )}
        {scope === 'group' && target ? (
          <p className="field__hint">
            {t('groupNote', { count: f(groups.data?.groups.find((g) => g.id === target)?.courses.length ?? 0) })}
          </p>
        ) : null}

        <h2 className="panel-card__title">{t('durationTitle')}</h2>
        <div className="choice-row" role="radiogroup" aria-label={t('durationTitle')}>
          {DURATIONS.map((d) => (
            <button
              key={d}
              type="button"
              role="radio"
              aria-checked={duration === d}
              className={cn('choice', duration === d && 'is-active')}
              onClick={() => setDuration(d)}
            >
              {t(`duration.${d}`)}
            </button>
          ))}
        </div>
        {duration === 'until' ? (
          <TextField
            type="date"
            dir="ltr"
            label={t('untilLabel')}
            hint={t('untilHint')}
            min={range.min}
            max={range.max}
            value={until}
            onChange={(e) => setUntil(e.target.value)}
            error={touched && problems.until ? t('untilInvalid') : undefined}
          />
        ) : null}
      </section>

      <section className="panel-card">
        <h2 className="panel-card__title">{t('previewTitle')}</h2>
        <p className="preview-quote">
          {selected ? t('previewText', { name: selected.label }) : t('previewTextEmpty')}
        </p>
        <p className="field__hint">{t('previewNote')}</p>
      </section>

      {mutation.isError ? (
        <p className="form-error" role="alert">
          {isApiError(mutation.error) && mutation.error.field('phones') && mutation.error.status === 422
            ? t('dailyCap')
            : isApiError(mutation.error) && mutation.error.field('target_id')
              ? t('targetRejected')
              : errorMessage(mutation.error)}
          {partial && partial.sent > 0 ? ` ${t('partial', { sent: f(partial.sent), total: f(parsed.valid.length) })}` : ''}
        </p>
      ) : null}
      {done ? (
        <p className="success-note" role="status">
          {t('done', { count: f(done.total) })}
          {done.renewed > 0 ? ` ${t('doneRenewed', { count: f(done.renewed) })}` : ''}
        </p>
      ) : null}

      <Button block loading={mutation.isPending} onClick={submit}>
        <Icon name="userPlus" size={20} />
        {parsed.valid.length > 0 ? t('submitCount', { count: f(parsed.valid.length) }) : t('submit')}
      </Button>
    </div>
  );
}
