'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ChangeEvent, useEffect, useId, useRef, useState } from 'react';

import {
  BIRTH_RANGES,
  type BirthField,
  CHILD_DELIVERY_TYPES,
  CHILD_PHOTO_TYPES,
  CHILD_SEXES,
  type ChildDeliveryType,
  type ChildSex,
  checkChildPhoto,
  childFieldError,
  deleteChildPhoto,
  MAX_CHILD_AGE_YEARS,
  MAX_CHILD_NAME,
  parseDecimalInput,
  photoErrorCode,
  saveChildPhoto,
  useChild,
  useChildPhoto,
  useChildren,
  useCreateChild,
  useDeleteChild,
  useUpdateChild,
} from '@/entities/child';
import { usePostpartum } from '@/entities/postpartum';
import { useLifeStage } from '@/entities/user';
import { getApiErrorCode, getApiErrorMessage, getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import {
  type DateParts,
  formatDecimal,
  formatLongDate,
  formatNumber,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today as todayDate,
} from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { type ChildFormState, EMPTY_FORM, formFromChild, type FormProblem, toChildInput, validateChildForm } from '../model/form';

type T = ReturnType<typeof useTranslations<'children'>>;

const API_FIELD: Record<string, string> = {
  name: 'name',
  birthDate: 'birth_date',
  weightKg: 'birth_weight_kg',
  lengthCm: 'birth_length_cm',
  headCm: 'birth_head_cm',
};

function Shell({ title, subtitle, onBack, t, children }: { title: string; subtitle?: string; onBack: () => void; t: T; children: React.ReactNode }) {
  return (
    <div className="view chd-screen chd-form-page">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={title} subtitle={subtitle} onBack={onBack} backLabel={t('common.back')} />
        <div className="chd-form">{children}</div>
      </div>
    </div>
  );
}

/**
 * `/children/new` (nbl_v15_AddChild) and `/children/[id]/edit`: name, birth
 * date, sex, birth weight/length/head, delivery type and an optional photo kept
 * on this device only. Edit mode adds «حذف فرزند» behind a confirmation. A
 * shared (spouse) child or a companion account can't edit — they get the
 * read-only note. A form: no bottom nav.
 */
export function ChildFormPage({ childId }: { childId?: number }) {
  const t = useTranslations('children');
  const editing = childId != null;
  const router = useRouter();
  const life = useLifeStage();
  const list = useChildren();
  const child = useChild(editing ? childId : null);

  const back = () => router.push(editing ? `/children/${childId}` : '/children');
  const title = editing ? t('form.titleEdit', { name: child.data?.name ?? '' }) : t('form.title');

  if ((editing && child.isPending) || (!editing && list.isPending) || life.isPending) {
    return (
      <Shell title={title} onBack={back} t={t}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (editing && child.isError) {
    return (
      <Shell title={title} onBack={back} t={t}>
        <EmptyState
          icon="warning"
          title={t('home.notFoundTitle')}
          body={t('home.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/children')}>{t('home.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }

  const readOnly = Boolean(life.data?.companion) || (editing && child.data != null && !child.data.canEdit);
  const atLimit = !editing && list.data != null && !list.data.canAdd;
  if (readOnly || atLimit) {
    return (
      <Shell title={title} onBack={back} t={t}>
        <EmptyState
          icon={readOnly ? 'eye' : 'users'}
          title={readOnly ? t('form.readOnly') : t('form.limitReached', { max: list.data?.maxChildren ?? 10 })}
          action={<SecondaryButton onClick={back}>{t('common.back')}</SecondaryButton>}
        />
      </Shell>
    );
  }

  return (
    <Shell title={title} subtitle={t('form.subtitle')} onBack={back} t={t}>
      <ChildForm
        key={child.data?.id ?? 'new'}
        childId={childId}
        initial={child.data ? formFromChild(child.data) : null}
        ownedCount={list.data?.ownedCount ?? 0}
        t={t}
      />
    </Shell>
  );
}

function ChildForm({
  childId,
  initial,
  ownedCount,
  t,
}: {
  childId?: number;
  initial: ChildFormState | null;
  ownedCount: number;
  t: T;
}) {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const editing = childId != null;
  const create = useCreateChild();
  const update = useUpdateChild(childId ?? 0);
  const mutation = editing ? update : create;
  // A first child right after delivery: the postpartum profile already knows the birth date and delivery type.
  const postpartum = usePostpartum();

  const [form, setForm] = useState<ChildFormState>(initial ?? EMPTY_FORM);
  const [touched, setTouched] = useState(false);
  const [dateOpen, setDateOpen] = useState(false);
  const prefilled = useRef(editing);

  useEffect(() => {
    if (prefilled.current || ownedCount > 0) return;
    const profile = postpartum.data?.profile;
    if (!profile) return;
    prefilled.current = true;
    setForm((f) =>
      f.birthDate ? f : { ...f, birthDate: profile.birthDate, deliveryType: f.deliveryType ?? profile.deliveryType },
    );
  }, [postpartum.data, ownedCount]);

  const today = toApiDate(todayDate());
  const problems = validateChildForm(form, today);
  const problemOf = (field: FormProblem['field']) => problems.find((p) => p.field === field);
  const set = <K extends keyof ChildFormState>(key: K, value: ChildFormState[K]) => setForm((f) => ({ ...f, [key]: value }));

  const errorText = (field: FormProblem['field']): string | undefined => {
    const server = childFieldError(mutation.error, API_FIELD[field]);
    if (server) return server;
    if (!touched) return undefined;
    const p = problemOf(field);
    if (!p) return undefined;
    switch (p.key) {
      case 'nameRequired':
        return t('form.errors.nameRequired');
      case 'nameTooLong':
        return t('form.errors.nameTooLong', { max: formatNumber(MAX_CHILD_NAME, locale) });
      case 'dateRequired':
        return t('form.errors.dateRequired');
      case 'future':
        return t('form.errors.future');
      case 'tooOld':
        return t('form.errors.tooOld', { years: formatNumber(MAX_CHILD_AGE_YEARS, locale) });
      case 'range': {
        const r = BIRTH_RANGES[p.field as BirthField];
        return t('form.errors.range', { min: formatDecimal(r.min, locale), max: formatDecimal(r.max, locale) });
      }
    }
  };

  // Photo: picked file → local preview; written to IndexedDB only after the child exists.
  const photo = usePhotoDraft(childId);

  const submit = () => {
    setTouched(true);
    if (problems.length > 0 || mutation.isPending) return;
    mutation.mutate(toChildInput(form), {
      onSuccess: async (saved) => {
        await photo.commit(saved.id);
        router.replace(`/children/${saved.id}`);
      },
    });
  };

  const onPick = (parts: DateParts) => {
    set('birthDate', toApiDate(partsToDate(parts, locale)));
    setDateOpen(false);
  };

  const limitError = getApiErrorCode(mutation.error) === 'children_limit' ? getApiErrorMessage(mutation.error) : undefined;
  const hasFieldError = Object.values(API_FIELD).some((f) => childFieldError(mutation.error, f));

  return (
    <>
      <PhotoPicker photo={photo} t={t} />

      <Card as="section" className="chd-fields">
        <TextRow
          label={t('form.name')}
          icon="user"
          value={form.name}
          placeholder={t('form.namePlaceholder')}
          maxLength={MAX_CHILD_NAME}
          error={errorText('name')}
          onChange={(v) => set('name', v)}
        />

        <div className="chd-field">
          <span className="chd-label" id="chd-birth-label">
            {t('form.birthDate')}
          </span>
          <button
            type="button"
            className="chd-date-btn"
            aria-expanded={dateOpen}
            aria-labelledby="chd-birth-label chd-birth-value"
            onClick={() => setDateOpen((o) => !o)}
          >
            <span id="chd-birth-value" className={clsx('chd-date-value', !form.birthDate && 'is-empty')}>
              {form.birthDate ? formatLongDate(fromApiDate(form.birthDate), locale) : t('form.pickDate')}
            </span>
            <Icon name="calendar" size={20} />
          </button>
          {dateOpen ? (
            <CalendarPicker
              value={form.birthDate ? toParts(fromApiDate(form.birthDate), locale) : null}
              onSelect={onPick}
            />
          ) : null}
          {errorText('birthDate') ? (
            <p className="chd-error" role="alert">
              {errorText('birthDate')}
            </p>
          ) : null}
        </div>

        <ChoiceRow<ChildSex>
          label={t('form.sex')}
          options={CHILD_SEXES}
          value={form.sex}
          optionLabel={(v) => t(v === 'girl' ? 'form.girl' : 'form.boy')}
          noneLabel={t('form.sexNoSay')}
          onChange={(v) => set('sex', v)}
        />

        <div className="chd-grid2">
          <DecimalRow
            label={t('form.birthWeight')}
            unit={t('form.kg')}
            value={form.weightKg}
            decimals={3}
            error={errorText('weightKg')}
            onChange={(v) => set('weightKg', v)}
          />
          <DecimalRow
            label={t('form.birthLength')}
            unit={t('form.cm')}
            value={form.lengthCm}
            decimals={1}
            error={errorText('lengthCm')}
            onChange={(v) => set('lengthCm', v)}
          />
          <DecimalRow
            label={t('form.birthHead')}
            unit={t('form.cm')}
            value={form.headCm}
            decimals={1}
            error={errorText('headCm')}
            onChange={(v) => set('headCm', v)}
          />
        </div>

        <ChoiceRow<ChildDeliveryType>
          label={t('form.delivery')}
          hint={t('form.deliveryHint')}
          options={CHILD_DELIVERY_TYPES}
          value={form.deliveryType}
          optionLabel={(v) => t(v === 'vaginal' ? 'form.vaginal' : 'form.cesarean')}
          noneLabel={t('form.deliveryNoSay')}
          onChange={(v) => set('deliveryType', v)}
        />
      </Card>

      <Card as="section" className="chd-included">
        <IncludedRow title={t('form.scheduleTitle')} sub={t('form.scheduleSub')} badge={t('form.included')} />
        <IncludedRow title={t('form.remindersTitle')} sub={t('form.remindersSub')} badge={t('form.included')} />
      </Card>

      {(limitError || (mutation.isError && !hasFieldError)) && (
        <p className="chd-error" role="alert">
          {limitError ?? getApiSaveErrorMessage(mutation.error, t('form.saveError'))}
        </p>
      )}
      <PrimaryButton loading={mutation.isPending} onClick={submit}>
        {editing ? t('form.save') : t('form.submit')}
      </PrimaryButton>

      {editing && childId != null ? <DeleteChild childId={childId} name={initial?.name ?? ''} t={t} /> : null}
    </>
  );
}

// ── Photo (device only) ────────────────────────────────────────
interface PhotoDraft {
  childId?: number;
  /** Preview of a freshly picked file (object URL), else null. */
  preview: string | null;
  removed: boolean;
  errorCode: LocalPhotoError | null;
  pick: (file: File) => void;
  remove: () => void;
  commit: (id: number) => Promise<void>;
}

type LocalPhotoError = ReturnType<typeof photoErrorCode>;

function usePhotoDraft(childId?: number): PhotoDraft {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [removed, setRemoved] = useState(false);
  const [errorCode, setErrorCode] = useState<LocalPhotoError | null>(null);

  useEffect(() => {
    if (!file) return undefined;
    const url = URL.createObjectURL(file);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  return {
    childId,
    preview: file ? preview : null,
    removed,
    errorCode,
    pick: (f) => {
      const problem = checkChildPhoto(f);
      if (problem) {
        setErrorCode(problem);
        return;
      }
      setErrorCode(null);
      setRemoved(false);
      setFile(f);
    },
    remove: () => {
      setFile(null);
      setRemoved(true);
      setErrorCode(null);
    },
    commit: async (id) => {
      try {
        if (file) await saveChildPhoto(id, file);
        else if (removed) await deleteChildPhoto(id);
      } catch (error) {
        // The child is saved; the optional photo just didn't make it onto this device.
        setErrorCode(photoErrorCode(error));
      }
    },
  };
}

function PhotoPicker({ photo, t }: { photo: PhotoDraft; t: T }) {
  const input = useRef<HTMLInputElement>(null);
  const stored = useChildPhoto(photo.childId);
  const shown = photo.preview ?? (photo.removed ? null : stored);
  const onChange = (e: ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (f) photo.pick(f);
    e.target.value = '';
  };
  const errorKey = photo.errorCode;
  return (
    <section className="chd-photo" aria-label={t('form.photo')}>
      <button
        type="button"
        className={clsx('chd-photo-btn', shown && 'has-photo')}
        aria-label={shown ? t('form.photoChange') : t('form.photoAdd')}
        onClick={() => input.current?.click()}
      >
        {shown ? (
          // A blob: URL of a local file — next/image can't optimise it.
          // eslint-disable-next-line @next/next/no-img-element
          <img src={shown} alt="" className="chd-photo-img" />
        ) : (
          <>
            <Icon name="camera" size={24} />
            <span>{t('form.photo')}</span>
          </>
        )}
      </button>
      <input
        ref={input}
        type="file"
        accept={CHILD_PHOTO_TYPES.join(',')}
        className="sr-only"
        tabIndex={-1}
        aria-hidden
        onChange={onChange}
      />
      <span className="chd-photo-hint">{t('form.photoHint')}</span>
      {shown ? (
        <button type="button" className="chd-photo-remove" onClick={photo.remove}>
          {t('form.photoRemove')}
        </button>
      ) : null}
      {errorKey ? (
        <p className="chd-error" role="alert">
          {photoErrorText(errorKey, t)}
        </p>
      ) : null}
      <InfoNote icon="shield" className="chd-photo-note">
        {t('form.photoPrivacy')}
      </InfoNote>
    </section>
  );
}

function photoErrorText(code: LocalPhotoError, t: T): string {
  switch (code) {
    case 'too_large':
      return t('form.photoErrors.too_large');
    case 'unsupported_type':
      return t('form.photoErrors.unsupported_type');
    case 'unavailable':
      return t('form.photoErrors.unavailable');
    case 'store_full':
      return t('form.photoErrors.store_full');
    case 'quota':
      return t('form.photoErrors.quota');
    default:
      return t('form.photoErrors.failed');
  }
}

// ── Field rows ─────────────────────────────────────────────────
function TextRow({
  label,
  icon,
  value,
  placeholder,
  maxLength,
  error,
  onChange,
}: {
  label: string;
  icon: 'user';
  value: string;
  placeholder?: string;
  maxLength?: number;
  error?: string;
  onChange: (v: string) => void;
}) {
  const id = useId();
  return (
    <div className="chd-field">
      <label htmlFor={id} className="chd-label">
        {label}
      </label>
      <div className={clsx('chd-input', error && 'is-invalid')}>
        <Icon name={icon} size={18} className="chd-input-icon" />
        <input
          id={id}
          value={value}
          placeholder={placeholder}
          maxLength={maxLength}
          autoComplete="off"
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${id}-err` : undefined}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
      {error ? (
        <p id={`${id}-err`} className="chd-error" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}

function DecimalRow({
  label,
  unit,
  value,
  decimals,
  error,
  onChange,
}: {
  label: string;
  unit: string;
  value: string;
  decimals: number;
  error?: string;
  onChange: (v: string) => void;
}) {
  const id = useId();
  const locale = useLocale() as Locale;
  // Shown in the locale's digits; kept canonical (ASCII, «.») in state.
  const shown = value === '' ? '' : formatDecimal(value, locale);
  return (
    <div className="chd-field">
      <label htmlFor={id} className="chd-label">
        {label}
      </label>
      <div className={clsx('chd-input', error && 'is-invalid')}>
        <input
          id={id}
          type="text"
          inputMode="decimal"
          autoComplete="off"
          value={shown}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${id}-err` : undefined}
          onChange={(e) => onChange(parseDecimalInput(e.target.value, decimals).text)}
        />
        <span className="chd-unit">{unit}</span>
      </div>
      {error ? (
        <p id={`${id}-err`} className="chd-error" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}

function ChoiceRow<V extends string>({
  label,
  hint,
  options,
  value,
  optionLabel,
  noneLabel,
  onChange,
}: {
  label: string;
  hint?: string;
  options: readonly V[];
  value: V | null;
  optionLabel: (v: V) => string;
  noneLabel: string;
  onChange: (v: V | null) => void;
}) {
  return (
    <div className="chd-field">
      <span className="chd-label">{label}</span>
      {hint ? <span className="chd-hint">{hint}</span> : null}
      <ChipGroup label={label}>
        {options.map((o) => (
          <PillChip key={o} pressed={value === o} onPressedChange={() => onChange(o)}>
            {optionLabel(o)}
          </PillChip>
        ))}
        <PillChip pressed={value === null} onPressedChange={() => onChange(null)}>
          {noneLabel}
        </PillChip>
      </ChipGroup>
    </div>
  );
}

/**
 * The artboard's two switches (national schedule, 3-day reminders) are not
 * settings in the API — both always apply — so they render as «فعال» rows,
 * never as switches that would do nothing.
 */
function IncludedRow({ title, sub, badge }: { title: string; sub: string; badge: string }) {
  return (
    <div className="chd-included-row">
      <span className="chd-included-text">
        <b>{title}</b>
        <span>{sub}</span>
      </span>
      <span className="chd-included-badge">
        <Icon name="check" size={14} strokeWidth={2.4} />
        {badge}
      </span>
    </div>
  );
}

// ── Delete ─────────────────────────────────────────────────────
function DeleteChild({ childId, name, t }: { childId: number; name: string; t: T }) {
  const router = useRouter();
  const remove = useDeleteChild(childId);
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className="chd-delete" onClick={() => setOpen(true)}>
        <Icon name="trash" size={18} />
        {t('form.delete')}
      </button>
      <AppSheet
        open={open}
        onClose={() => (remove.isPending ? undefined : setOpen(false))}
        size="half"
        title={t('form.deleteTitle', { name })}
        footer={
          <div className="chd-sheet-btns">
            <SecondaryButton onClick={() => setOpen(false)} disabled={remove.isPending}>
              {t('form.cancel')}
            </SecondaryButton>
            <SecondaryButton
              variant="text"
              className="chd-danger-text"
              loading={remove.isPending}
              onClick={() => remove.mutate(undefined, { onSuccess: () => router.replace('/children') })}
            >
              {t('form.deleteConfirm')}
            </SecondaryButton>
          </div>
        }
      >
        <div className="chd-sheet-body">
          <IconCircle icon="trash" tone="danger" size="lg" />
          <p>{t('form.deleteBody', { name })}</p>
        </div>
        {remove.isError ? (
          <p className="chd-error" role="alert">
            {t('form.deleteError')}
          </p>
        ) : null}
      </AppSheet>
    </>
  );
}
