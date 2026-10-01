'use client';

import { useQueryClient } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useId, useRef, useState, type KeyboardEvent } from 'react';

import { careKeys } from '@/entities/care-reminder';
import {
  CONTRACEPTION_METHODS,
  type ContraceptionMethod,
  type ContraceptionMethodCode,
  IUD_YEARS_RANGE,
  isPillMethod,
  METHOD_LOOK,
  PACK_TYPES,
  useContraception,
  useSaveContraceptionMethod,
} from '@/entities/contraception';
import { lifeStageKeys } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import {
  ChipGroup,
  EmptyState,
  Icon,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import {
  chooseMethod,
  type DraftField,
  draftProblems,
  draftToPayload,
  initialDraft,
  type SetupDraft,
} from '../model/draft';
import { DateSheet, TimeSheet } from './Pickers';

type DateField = 'packStartedOn' | 'insertedOn' | 'injectedOn';
type OpenPicker = { kind: 'date'; field: DateField; title: string } | { kind: 'time' } | null;

/** Next radio index for an arrow / Home / End key, wrapping (APG radiogroup). */
function radioStep(key: string, index: number, count: number, rtl: boolean): number | null {
  if (key === (rtl ? 'ArrowLeft' : 'ArrowRight') || key === 'ArrowDown') return (index + 1) % count;
  if (key === (rtl ? 'ArrowRight' : 'ArrowLeft') || key === 'ArrowUp') return (index - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return null;
}

/**
 * «روش پیشگیری» (`/contraception/setup`, CB-CONTRA-02, nbl_Contra_Setup): the
 * eight method tiles, then what that method needs — pack type, current pack
 * start and the pill reminder for pills; insertion / injection dates for the
 * long-acting ones. Saving (`PUT /contraception/method`) also switches «track
 * contraception» on. A form: no bottom nav (IA_Nav).
 */
export function ContraceptionSetupPage() {
  const t = useTranslations('contraception.setup');
  const router = useRouter();
  const query = useContraception();
  const saved = query.data?.method ?? null;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="ctr-skel">
        <div className="ctr-methods">
          {Array.from({ length: 8 }, (_, i) => (
            <Skeleton key={i} shape="block" className="ctr-skel-tile" />
          ))}
        </div>
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = <LoadError onRetry={() => void query.refetch()} retrying={query.isFetching} />;
  } else {
    body = <SetupForm saved={saved} />;
  }

  return (
    <div className="view ctr-page">
      <SkyLayer />
      <div className="scroll ctr-scroll is-form">
        <ScreenHeader
          title={t('title')}
          onBack={() => router.push(saved ? '/contraception' : '/profile/mode')}
          backLabel={t('back')}
        />
        {body}
      </div>
    </div>
  );
}

function LoadError({ onRetry, retrying }: { onRetry: () => void; retrying: boolean }) {
  const t = useTranslations('contraception.error');
  return (
    <EmptyState
      icon="shield"
      title={t('title')}
      body={t('body')}
      action={
        <PrimaryButton icon="refresh" loading={retrying} onClick={onRetry}>
          {t('retry')}
        </PrimaryButton>
      }
    />
  );
}

function SetupForm({ saved }: { saved: ContraceptionMethod | null }) {
  const t = useTranslations('contraception.setup');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const save = useSaveContraceptionMethod();
  const todayIso = toApiDate(today());
  const [draft, setDraft] = useState<SetupDraft>(() => initialDraft(saved, todayIso));
  const [picker, setPicker] = useState<OpenPicker>(null);
  const [touched, setTouched] = useState(false);

  const problems = draftProblems(draft, todayIso);
  const blocked = Object.keys(problems).length > 0;
  const method = draft.method;

  const submit = () => {
    setTouched(true);
    if (blocked || !method) return;
    save.mutate(draftToPayload({ ...draft, method }), {
      onSuccess: () => {
        // The save flips «track contraception» and the pill reminder pref, and syncs care reminders.
        void queryClient.invalidateQueries({ queryKey: lifeStageKeys.all });
        void queryClient.invalidateQueries({ queryKey: careKeys.all });
        void queryClient.invalidateQueries({ queryKey: ['notification-settings'] });
        void queryClient.invalidateQueries({ queryKey: ['cycle-settings'] });
        router.replace('/contraception');
      },
    });
  };

  const dateValue = (value: string | null) =>
    value ? formatDayMonth(fromApiDate(value), locale) : t('choose');

  const problemText = (field: DraftField) => {
    const p = problems[field];
    if (!p || (p === 'missing' && !touched)) return null;
    return p === 'future' ? t('futureDate') : field === 'method' ? t('pickMethod') : null;
  };

  return (
    <>
      <div className="ctr-intro">
        <h2 className="ctr-question">{t('question')}</h2>
        <p className="ctr-lead">{t('lead')}</p>
      </div>

      <MethodGrid
        value={method}
        onChange={(m) => setDraft((d) => chooseMethod(d, m, saved))}
      />
      {problemText('method') ? (
        <p className="ctr-error" role="alert">
          {problemText('method')}
        </p>
      ) : null}

      {method ? (
        <section className="nb-card ctr-details" aria-label={t('details')}>
          {method === 'combined_pill' ? (
            <div className="ctr-block">
              <h3 className="ctr-block-title">{t('packType')}</h3>
              <ChipGroup label={t('packType')} className="ctr-packs">
                {PACK_TYPES.map((type) => (
                  <PillChip
                    key={type}
                    className="ctr-chip"
                    pressed={draft.packType === type}
                    onPressedChange={() => setDraft((d) => ({ ...d, packType: type }))}
                  >
                    {t(`packTypes.${type}`)}
                  </PillChip>
                ))}
              </ChipGroup>
            </div>
          ) : null}

          {isPillMethod(method) ? (
            <>
              <FieldRow
                label={t('packStart')}
                value={dateValue(draft.packStartedOn)}
                error={problemText('packStartedOn')}
                onClick={() => setPicker({ kind: 'date', field: 'packStartedOn', title: t('packStart') })}
              />
              <FieldRow
                label={t('reminderTime')}
                value={formatNumber(draft.reminderTime, locale)}
                ltrValue
                onClick={() => setPicker({ kind: 'time' })}
              />
            </>
          ) : null}

          {method === 'copper_iud' || method === 'hormonal_iud' ? (
            <>
              <FieldRow
                label={t('insertedOn')}
                value={dateValue(draft.insertedOn)}
                error={problemText('insertedOn')}
                onClick={() => setPicker({ kind: 'date', field: 'insertedOn', title: t('insertedOn') })}
              />
              <NumberStepper
                className="ctr-stepper"
                label={t('lifetime')}
                unit={t('years')}
                value={draft.iudYears}
                min={IUD_YEARS_RANGE.min}
                max={IUD_YEARS_RANGE.max}
                onChange={(v) => setDraft((d) => ({ ...d, iudYears: v }))}
                decrementLabel={t('decrease', { what: t('lifetime') })}
                incrementLabel={t('increase', { what: t('lifetime') })}
                locale={locale}
              />
            </>
          ) : null}

          {method === 'injection' ? (
            <FieldRow
              label={t('injectedOn')}
              value={dateValue(draft.injectedOn)}
              error={problemText('injectedOn')}
              onClick={() => setPicker({ kind: 'date', field: 'injectedOn', title: t('injectedOn') })}
            />
          ) : null}

          {method === 'implant' ? (
            <FieldRow
              label={t('implantInsertedOn')}
              hint={t('optional')}
              value={dateValue(draft.insertedOn)}
              error={problemText('insertedOn')}
              onClick={() => setPicker({ kind: 'date', field: 'insertedOn', title: t('implantInsertedOn') })}
            />
          ) : null}

          {method === 'condom' || method === 'other' ? <p className="ctr-note">{t('noDetails')}</p> : null}
        </section>
      ) : null}

      <div className="ctr-footer">
        {save.isError ? (
          <p className="ctr-error" role="alert">
            {getApiSaveErrorMessage(save.error, t('saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} disabled={touched && blocked} onClick={submit}>
          {t('save')}
        </PrimaryButton>
      </div>

      {picker?.kind === 'time' ? (
        <TimeSheet
          value={draft.reminderTime}
          onClose={() => setPicker(null)}
          onPick={(clock) => {
            setDraft((d) => ({ ...d, reminderTime: clock }));
            setPicker(null);
          }}
        />
      ) : null}
      {picker?.kind === 'date' ? (
        <DateSheet
          title={picker.title}
          value={draft[picker.field]}
          onClose={() => setPicker(null)}
          onPick={(value) => {
            const field = picker.field;
            setDraft((d) => ({ ...d, [field]: value }));
            setPicker(null);
          }}
        />
      ) : null}
    </>
  );
}

/** The 2-column method tiles as one radiogroup (roving tabindex, arrow keys). */
function MethodGrid({
  value,
  onChange,
}: {
  value: ContraceptionMethodCode | null;
  onChange: (method: ContraceptionMethodCode) => void;
}) {
  const t = useTranslations('contraception.setup');
  const rtl = useDirection() === 'rtl';
  const groupId = useId();
  const refs = useRef<Array<HTMLButtonElement | null>>([]);
  const checkedIndex = value ? CONTRACEPTION_METHODS.indexOf(value) : -1;
  const tabStop = checkedIndex >= 0 ? checkedIndex : 0;

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const next = radioStep(event.key, index, CONTRACEPTION_METHODS.length, rtl);
    if (next === null) return;
    event.preventDefault();
    onChange(CONTRACEPTION_METHODS[next]!);
    refs.current[next]?.focus();
  };

  return (
    <div role="radiogroup" aria-label={t('question')} className="ctr-methods">
      {CONTRACEPTION_METHODS.map((code, index) => {
        const look = METHOD_LOOK[code];
        const checked = code === value;
        return (
          <button
            key={code}
            ref={(el) => {
              refs.current[index] = el;
            }}
            type="button"
            role="radio"
            aria-checked={checked}
            aria-labelledby={`${groupId}-${code}`}
            tabIndex={index === tabStop ? 0 : -1}
            className={clsx('ctr-method', `nb-tone-${look.tone}`)}
            onClick={() => onChange(code)}
            onKeyDown={(event) => onKeyDown(event, index)}
          >
            <Icon name={look.icon} size={22} strokeWidth={1.8} className="ctr-method-icon" />
            <span id={`${groupId}-${code}`} className="ctr-method-label">
              {t(`methods.${code}`)}
            </span>
          </button>
        );
      })}
    </div>
  );
}

/** One «label … value» row of the details card that opens a picker. */
function FieldRow({
  label,
  hint,
  value,
  error,
  ltrValue,
  onClick,
}: {
  label: string;
  hint?: string;
  value: string;
  error?: string | null;
  ltrValue?: boolean;
  onClick: () => void;
}) {
  return (
    <div className="ctr-field-wrap">
      <button type="button" className={clsx('ctr-field', error && 'is-invalid')} onClick={onClick}>
        <span className="ctr-field-label">
          {label}
          {hint ? <span className="ctr-field-hint">{hint}</span> : null}
        </span>
        <span className="ctr-field-value" dir={ltrValue ? 'ltr' : undefined}>
          {value}
        </span>
      </button>
      {error ? (
        <p className="ctr-error is-inline" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}
