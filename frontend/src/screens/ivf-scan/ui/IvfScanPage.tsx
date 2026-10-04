'use client';

import { useId, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import {
  IVF_FOLLICLE_BINS,
  IVF_OVARIES,
  IVF_SCAN_LIMITS,
  type IvfGrowthPoint,
  type IvfOvary,
  type IvfScan,
  type IvfScansView,
  ivfScanErrorMessage,
  useDeleteIvfScan,
  useIvfScans,
  useSaveIvfScan,
} from '@/entities/ivf';
import { getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  InfoNote,
  NumberStepper,
  PrimaryButton,
  ScreenHeader,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { GroupedColumns } from '@/widgets/charts';

import { checkForm, formFrom, growthWindow, isDirty, scanDate, type ScanForm, stimDayOn } from '../model/form';

type T = ReturnType<typeof useTranslations<'ivf'>>;

function Shell({ date, stimDay, children }: { date: string; stimDay: number | null; children: React.ReactNode }) {
  const t = useTranslations('ivf');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const when = formatDayMonth(fromApiDate(date), locale);
  return (
    <div className="view ivs-page">
      <SkyLayer />
      <div className="scroll ivs-scroll">
        <ScreenHeader
          title={t('scan.title')}
          subtitle={stimDay !== null ? t('scan.subtitle', { day: formatNumber(stimDay, locale), date: when }) : when}
          onBack={() => router.push('/ivf')}
          backLabel={t('scan.back')}
        />
        {children}
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
/**
 * «ثبت نتیجه سونو» — `/ivf/scan[?date=Y-m-d]` (nbl_IVF_Scan, CB-IVF-04): the
 * day's follicle counts per ovary per size bin, endometrium + E2, the growth
 * chart (10–14 mm vs ≥ 15 mm per scan day) and the note that interpreting it is
 * the doctor's job. One scan per day; saving again replaces it.
 */
export function IvfScanPage({ date: param }: { date?: string }) {
  const t = useTranslations('ivf');
  const mounted = useMounted();
  const query = useIvfScans();
  // The device's today (Tehran wall clock in practice); a future or malformed `?date=` falls back to it.
  const date = scanDate(param, toApiDate(today()));

  if (!mounted || query.isPending) {
    return (
      <Shell date={date} stimDay={null}>
        <SkeletonGroup label={t('scan.loading')} className="ivs-body">
          <Skeleton shape="card" className="ivs-skel-tall" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell date={date} stimDay={null}>
        <div className="ivs-body">
          <Card>
            <EmptyState
              icon="warning"
              title={t('scan.loadError')}
              action={
                <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
                  {t('scan.retry')}
                </PrimaryButton>
              }
            />
          </Card>
        </div>
      </Shell>
    );
  }

  const view = query.data;
  if (!view.cycle || view.cycle.status !== 'open') {
    return (
      <Shell date={date} stimDay={null}>
        <div className="ivs-body">
          <Card>
            <EmptyState
              icon="note"
              title={t('scan.noCycle.title')}
              body={t('scan.noCycle.body')}
              action={
                <Link href="/ivf" className="nb-btn is-primary is-block">
                  {t('scan.noCycle.action')}
                </Link>
              }
            />
          </Card>
        </div>
      </Shell>
    );
  }

  const saved = view.scans.find((s) => s.date === date);
  return (
    <Shell date={date} stimDay={saved?.stimDay ?? stimDayOn(date, view.stimStartedOn)}>
      {/* Remount on a different saved answer, so the form restarts from it. */}
      <ScanFormView key={`${date}:${saved ? JSON.stringify(saved) : 'new'}`} date={date} saved={saved} view={view} t={t} />
    </Shell>
  );
}

function ScanFormView({ date, saved, view, t }: { date: string; saved: IvfScan | undefined; view: IvfScansView; t: T }) {
  const locale = useLocale() as Locale;
  const base = formFrom(saved, (v) => formatDecimal(v, locale));
  const [form, setForm] = useState<ScanForm>(base);
  const [tried, setTried] = useState(false);
  const [status, setStatus] = useState<'saved' | 'deleted' | null>(null);
  const save = useSaveIvfScan();
  const remove = useDeleteIvfScan();

  const check = checkForm(form, date, saved?.notes ?? null);
  const dirty = isDirty(base, form);
  const busy = save.isPending || remove.isPending;

  const edit = (next: ScanForm) => {
    setForm(next);
    setStatus(null);
  };

  const submit = () => {
    setTried(true);
    if (!check.input || busy) return;
    save.mutate(check.input, { onSuccess: () => setStatus('saved') });
  };

  const onDelete = () => {
    if (busy) return;
    save.reset();
    remove.mutate(date, { onSuccess: () => setStatus('deleted') });
  };

  const saveError = save.isError ? (ivfScanErrorMessage(save.error) ?? getApiSaveErrorMessage(save.error, t('scan.saveError'))) : null;
  const deleteError = remove.isError ? getApiSaveErrorMessage(remove.error, t('scan.deleteError')) : null;

  return (
    <>
      <div className="ivs-body">
        <section className="ivs-section" aria-labelledby="ivs-follicles">
          <div className="ivs-head">
            <h2 id="ivs-follicles" className="ivs-title">
              {t('scan.follicles.title')}
            </h2>
            <p className="ivs-hint">{t('scan.follicles.hint')}</p>
          </div>
          <Card className="ivs-ovaries">
            {IVF_OVARIES.map((ovary) => (
              <OvaryColumn key={ovary} ovary={ovary} form={form} onChange={edit} t={t} />
            ))}
          </Card>
        </section>

        <Card className="ivs-labs">
          <DecimalRow
            label={t('scan.endometrium.label')}
            value={form.endometrium}
            placeholder={t('scan.endometrium.placeholder')}
            suffix={<span className="ivs-unit">{t('scan.endometrium.unit')}</span>}
            error={tried && check.errors.endometrium ? t('scan.endometrium.invalid') : null}
            onChange={(endometrium) => edit({ ...form, endometrium })}
          />
          <DecimalRow
            label={t('scan.e2.label')}
            value={form.e2}
            placeholder={t('scan.e2.placeholder')}
            suffix={
              <button
                type="button"
                className="ivs-unit-btn"
                aria-label={t('scan.e2.unitToggle', { unit: t(`scan.e2.units.${form.e2Unit}`) })}
                onClick={() => edit({ ...form, e2Unit: form.e2Unit === 'pg_ml' ? 'pmol_l' : 'pg_ml' })}
              >
                {t(`scan.e2.units.${form.e2Unit}`)}
              </button>
            }
            error={tried && check.errors.e2 ? t('scan.e2.invalid') : null}
            onChange={(e2) => edit({ ...form, e2 })}
          />
        </Card>

        <GrowthCard growth={view.growth} t={t} />

        <InfoNote className="ivs-note">{t('scan.note')}</InfoNote>

        {saved ? (
          <button type="button" className="ivs-delete" disabled={busy} onClick={onDelete}>
            {t('scan.delete')}
          </button>
        ) : null}
      </div>

      <div className="ivs-footer">
        <p className="ivs-status" role="status">
          {status ? t(`scan.${status}`) : ''}
        </p>
        {saveError || deleteError ? (
          <p className="ivs-error" role="alert">
            {saveError ?? deleteError}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} disabled={!dirty || remove.isPending} onClick={submit}>
          {t('scan.save')}
        </PrimaryButton>
      </div>
    </>
  );
}

function OvaryColumn({ ovary, form, onChange, t }: { ovary: IvfOvary; form: ScanForm; onChange: (f: ScanForm) => void; t: T }) {
  const locale = useLocale() as Locale;
  const titleId = useId();
  const name = t(`scan.ovary.${ovary}`);
  return (
    <div className="ivs-ovary" role="group" aria-labelledby={titleId}>
      <h3 id={titleId} className="ivs-ovary-title">
        {name}
      </h3>
      {IVF_FOLLICLE_BINS.map((bin) => {
        const label = t(`scan.bins.${bin}`);
        return (
          <NumberStepper
            key={bin}
            className="ivs-stepper"
            label={label}
            value={form[ovary][bin]}
            min={0}
            max={IVF_SCAN_LIMITS.follicles}
            onChange={(v) => onChange({ ...form, [ovary]: { ...form[ovary], [bin]: v } })}
            decrementLabel={t('scan.decrease', { ovary: name, bin: label })}
            incrementLabel={t('scan.increase', { ovary: name, bin: label })}
            locale={locale}
          />
        );
      })}
    </div>
  );
}

function DecimalRow({
  label,
  value,
  placeholder,
  suffix,
  error,
  onChange,
}: {
  label: string;
  value: string;
  placeholder: string;
  suffix: React.ReactNode;
  error: string | null;
  onChange: (text: string) => void;
}) {
  const id = useId();
  return (
    <div className="ivs-lab">
      <div className="ivs-lab-row">
        <label htmlFor={id} className="ivs-lab-label">
          {label}
        </label>
        <span className="ivs-lab-field">
          {/* Body font, not Lalezar: Lalezar draws «٫» like «/» (fertility audit #23). */}
          <input
            id={id}
            className="ivs-lab-input"
            inputMode="decimal"
            dir="ltr"
            autoComplete="off"
            value={value}
            placeholder={placeholder}
            aria-invalid={error !== null}
            aria-describedby={error ? `${id}-err` : undefined}
            onChange={(e) => onChange(e.target.value)}
          />
          {suffix}
        </span>
      </div>
      {error ? (
        <p id={`${id}-err`} className="ivs-error">
          {error}
        </p>
      ) : null}
    </div>
  );
}

function GrowthCard({ growth, t }: { growth: readonly IvfGrowthPoint[]; t: T }) {
  const locale = useLocale() as Locale;
  const points = growthWindow(growth);
  const dayLabel = (p: IvfGrowthPoint) =>
    p.stimDay !== null ? t('scan.growth.day', { day: formatNumber(p.stimDay, locale) }) : formatDayMonth(fromApiDate(p.date), locale);
  const mid = t('scan.growth.mid');
  const lead = t('scan.growth.lead');
  return (
    <section className="ivs-section" aria-labelledby="ivs-growth">
      <SectionTitle id="ivs-growth" title={t('scan.growth.title')} />
      <Card className="ivs-growth">
        {points.length ? (
          <>
            <GroupedColumns
              label={t('scan.growth.label', { count: formatNumber(points.length, locale) })}
              tones={['brand', 'bloom']}
              height={120}
              groups={points.map((p) => ({ key: p.date, label: dayLabel(p), values: [p.mid, p.lead] }))}
              table={{
                caption: t('scan.growth.caption'),
                columns: [t('scan.growth.column'), mid, lead],
                rows: points.map((p) => [dayLabel(p), formatNumber(p.mid, locale), formatNumber(p.lead, locale)]),
              }}
            />
            <ul className="ivs-legend" aria-hidden>
              <li className="nb-tone-brand">{mid}</li>
              <li className="nb-tone-bloom">{lead}</li>
            </ul>
          </>
        ) : (
          <p className="ivs-empty">{t('scan.growth.empty')}</p>
        )}
      </Card>
    </section>
  );
}
