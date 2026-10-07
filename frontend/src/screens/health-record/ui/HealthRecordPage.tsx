'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useId, useState } from 'react';

import {
  type HealthRecord,
  type PregnancyEntry,
  type ValueSummary,
  healthRecordKeys,
  useHealthRecord,
} from '@/entities/health-record';
import { QuickEditSheet, type QuickEditField } from '@/features/edit-profile';
import { RecordHomeExtras, RecordHomeSheets } from '@/features/record-documents';
import { type Locale, useRouter } from '@/shared/i18n';
import {
  formatDayMonth,
  formatDecimal,
  formatLongDate,
  formatMonthLabel,
  formatNumber,
  fromApiDate,
  toApiDate,
  today,
  toParts,
} from '@/shared/lib/date';
import {
  Avatar,
  Card,
  EmptyState,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';
import type { IconName } from '@/shared/ui';

import { births, medicationSchedule, regularityKey } from '../model/format';
import { AllergiesSheet, BasicsSheet, ConditionsSheet, PregnanciesSheet } from './RecordSheets';

type SheetId = 'basics' | 'conditions' | 'allergies' | 'pregnancies';

/**
 * «پرونده سلامت من» (bloom B-N6-03, nbl_Record_Summary): the owner's record aggregated by the API, one card per
 * section, user-owned sections editable through sheets. Health data (§11): nothing here is logged or put in a URL.
 */
export function HealthRecordPage() {
  const t = useTranslations('healthRecord');
  const router = useRouter();
  const query = useHealthRecord();
  const loc = useLocale() as Locale;
  const [sheet, setSheet] = useState<SheetId | null>(null);

  let body: ReactNode;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="hrec-skel">
        <Skeleton shape="card" className="hrec-skel-hero" />
        <Skeleton shape="card" className="hrec-skel-card" />
        <Skeleton shape="card" className="hrec-skel-card" />
        <Skeleton shape="card" className="hrec-skel-card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <RecordContent record={query.data} onSheet={setSheet} />;
  }

  const updated = query.data ? updatedLabel(query.data, t, loc) : undefined;
  return (
    <div className="view hrec-page">
      <SkyLayer />
      <div className="scroll hrec-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={updated}
          onBack={() => router.push('/profile')}
          backLabel={t('back')}
        />
        {body}
        <div className="hrec-footer">
          <PrimaryButton icon="fileDoc" onClick={() => router.push('/record/export')}>
            {t('report.cta')}
          </PrimaryButton>
        </div>
      </div>
      {/* Sheets sit outside the scroller: AppSheet is positioned against the screen, not the scrolled content. */}
      {query.data ? <RecordSheets record={query.data} sheet={sheet} onSheet={setSheet} /> : null}
      <RecordHomeSheets />
    </div>
  );
}

type T = ReturnType<typeof useTranslations<'healthRecord'>>;

function updatedLabel(r: HealthRecord, t: T, loc: Locale): string | undefined {
  if (!r.updatedAt) return undefined;
  const at = new Date(r.updatedAt);
  return toApiDate(at) === toApiDate(today()) ? t('updatedToday') : t('updatedOn', { date: formatDayMonth(at, loc) });
}

function RecordContent({ record, onSheet }: { record: HealthRecord; onSheet: (s: SheetId) => void }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const r = record;
  const p = r.person;

  const num = (v: number) => formatNumber(v, loc);
  const meta = [
    p.age !== null ? t('person.age', { age: num(p.age) }) : null,
    p.gender === 'female' || p.gender === 'male' ? t(`person.gender.${p.gender}`) : null,
    t('person.mode', { mode: modeLabel(p.lifeMode, t) }),
  ].filter(Boolean);

  return (
    <>
      <Card variant="hero" className="hrec-person">
        <Avatar name={p.name || t('title')} size="lg" />
        <div className="hrec-person-text">
          <p className="hrec-person-name">{p.name || t('title')}</p>
          <p className="hrec-person-meta">{meta.join(' · ')}</p>
        </div>
      </Card>

      {/* CB-REC-04 (canvas nbl_Rec_Home): allergy card, category grid, needs-review card, «افزودن سند». */}
      <RecordHomeExtras />

      <BasicsCard record={r} onEdit={() => onSheet('basics')} />
      <ConditionsCard record={r} onEdit={() => onSheet('conditions')} />
      <MedicationsCard record={r} onManage={() => router.push('/reminders')} />
      <AllergiesCard record={r} onEdit={() => onSheet('allergies')} />
      <CycleCard record={r} />
      <VitalsCard record={r} onOpen={() => router.push('/vitals')} />
      <PregnanciesCard record={r} onEdit={() => onSheet('pregnancies')} />
      <CheckupsCard record={r} onCheckups={() => router.push('/checkups/history')} onLab={(id) => router.push(`/labs/${id}`)} />

      <InfoNote>{t('note')}</InfoNote>

    </>
  );
}

function RecordSheets({
  record: r,
  sheet,
  onSheet,
}: {
  record: HealthRecord;
  sheet: SheetId | null;
  onSheet: (s: SheetId | null) => void;
}) {
  const queryClient = useQueryClient();
  const [quick, setQuick] = useState<QuickEditField | null>(null);
  const close = () => onSheet(null);
  return (
    <>
      <BasicsSheet
        open={sheet === 'basics'}
        onClose={close}
        basics={r.basics.data}
        onEditBody={(field) => {
          close();
          setQuick(field);
        }}
      />
      <ConditionsSheet
        open={sheet === 'conditions'}
        onClose={close}
        conditions={r.conditions.data}
        medications={r.medications.data.profileMedications}
      />
      <AllergiesSheet open={sheet === 'allergies'} onClose={close} allergies={r.allergies.data} />
      <PregnanciesSheet open={sheet === 'pregnancies'} onClose={close} items={r.pregnancies.data.items} />
      <QuickEditSheet
        field={quick}
        values={{ height: r.basics.data.heightCm, weight: r.basics.data.weightKg }}
        onClose={() => {
          setQuick(null);
          // POST /profile invalidates the profile caches only; the record reads the same rows.
          void queryClient.invalidateQueries({ queryKey: healthRecordKeys.all });
        }}
      />
    </>
  );
}

function modeLabel(mode: string, t: T): string {
  const known = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen', 'companion'] as const;
  const m = known.find((k) => k === mode) ?? 'cycle';
  return t(`person.lifeMode.${m}`);
}

/* ── Section card shell ─────────────────────────────────────────────────────────────────────────────────────────── */

interface SectionCardProps {
  icon: IconName;
  tone: Tone;
  title: string;
  /** Shown when the section holds user-owned data. */
  onEdit?: () => void;
  /** A text action instead of «ویرایش» (e.g. «علائم حیاتی» → /vitals). */
  action?: { label: string; onClick: () => void };
  children: ReactNode;
}

function SectionCard({ icon, tone, title, onEdit, action, children }: SectionCardProps) {
  const t = useTranslations('healthRecord');
  const id = useId();
  const link = onEdit ? { label: t('edit'), onClick: onEdit, aria: t('editSection', { section: title }) } : action ? { ...action, aria: action.label } : null;
  return (
    <Card as="section" className="hrec-card" aria-labelledby={id}>
      <div className="hrec-card-head">
        <IconCircle icon={icon} tone={tone} size="md" />
        <h2 id={id} className="hrec-card-title">
          {title}
        </h2>
        {link ? (
          <button type="button" className="hrec-edit" onClick={link.onClick} aria-label={link.aria}>
            {link.label}
          </button>
        ) : null}
      </div>
      {children}
    </Card>
  );
}

function Tile({ label, value, sub }: { label: string; value: ReactNode; sub?: ReactNode }) {
  return (
    <div className="hrec-tile">
      <span className="hrec-tile-label">{label}</span>
      <span className="hrec-tile-value">{value}</span>
      {sub ? <span className="hrec-tile-sub">{sub}</span> : null}
    </div>
  );
}

function Row({ label, value, onClick }: { label: ReactNode; value?: ReactNode; onClick?: () => void }) {
  const content = (
    <>
      <span className="hrec-row-label">{label}</span>
      {value !== undefined ? <span className="hrec-row-value">{value}</span> : null}
    </>
  );
  return onClick ? (
    <button type="button" className="hrec-row is-action" onClick={onClick}>
      {content}
    </button>
  ) : (
    <div className="hrec-row">{content}</div>
  );
}

function Empty({ children }: { children: ReactNode }) {
  return <p className="hrec-empty">{children}</p>;
}

/* ── Sections ───────────────────────────────────────────────────────────────────────────────────────────────────── */

function BasicsCard({ record, onEdit }: { record: HealthRecord; onEdit: () => void }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const b = record.basics.data;
  const dash = t('notSet');
  return (
    <SectionCard icon="user" tone="brand" title={t('sections.basics')} onEdit={record.basics.editable ? onEdit : undefined}>
      <div className="hrec-tiles">
        <Tile label={t('basics.height')} value={b.heightCm !== null ? t('basics.heightValue', { value: formatNumber(b.heightCm, loc) }) : dash} />
        <Tile label={t('basics.weight')} value={b.weightKg !== null ? t('basics.weightValue', { value: formatDecimal(b.weightKg, loc) }) : dash} />
        <Tile label={t('basics.bmi')} value={b.bmi ? formatDecimal(b.bmi.value, loc) : dash} />
        <Tile
          label={t('basics.bloodType')}
          value={b.bloodType ? <bdi dir="ltr">{b.bloodType}</bdi> : dash}
          sub={b.bloodTypeSource === 'pregnancy' ? t('basics.bloodFromPregnancy') : undefined}
        />
      </div>
      {record.basics.empty ? <Empty>{t('basics.empty')}</Empty> : null}
    </SectionCard>
  );
}

function ConditionsCard({ record, onEdit }: { record: HealthRecord; onEdit: () => void }) {
  const t = useTranslations('healthRecord');
  const c = record.conditions.data;
  const chronic = (c.chronicIllnesses ?? []).filter(isChronic);
  const gyn = (c.gynConditions ?? []).filter(isGyn);
  return (
    <SectionCard icon="warning" tone="period" title={t('sections.conditions')} onEdit={record.conditions.editable ? onEdit : undefined}>
      {chronic.length + gyn.length === 0 ? (
        <Empty>{c.answered ? t('conditions.empty') : t('notAnswered')}</Empty>
      ) : (
        <ul className="hrec-tags" aria-label={t('sections.conditions')}>
          {chronic.map((code) => (
            <li key={`c-${code}`} className="hrec-tag">
              {t(`conditions.chronicIllness.${code}`)}
            </li>
          ))}
          {gyn.map((code) => (
            <li key={`g-${code}`} className="hrec-tag">
              {t(`conditions.gynCondition.${code}`)}
            </li>
          ))}
        </ul>
      )}
    </SectionCard>
  );
}

const CHRONIC = ['diabetes', 'hypertension', 'thyroid', 'asthma', 'anemia', 'migraine', 'other'] as const;
const GYN = ['pcos', 'endometriosis', 'fibroids', 'recurrent_infections', 'other'] as const;
const PROFILE_MEDS = ['contraceptive_pill', 'iud', 'hormonal_medication'] as const;
const isChronic = (c: string): c is (typeof CHRONIC)[number] => (CHRONIC as readonly string[]).includes(c);
const isGyn = (c: string): c is (typeof GYN)[number] => (GYN as readonly string[]).includes(c);
const isProfileMed = (c: string): c is (typeof PROFILE_MEDS)[number] => (PROFILE_MEDS as readonly string[]).includes(c);

function MedicationsCard({ record, onManage }: { record: HealthRecord; onManage: () => void }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const m = record.medications.data;
  const listed = (m.profileMedications ?? []).filter(isProfileMed);
  return (
    <SectionCard
      icon="pill"
      tone="data"
      title={t('sections.medications')}
      onEdit={record.medications.editable ? onManage : undefined}
    >
      {m.items.length === 0 && listed.length === 0 ? <Empty>{t('medications.empty')}</Empty> : null}
      {m.items.length > 0 ? (
        <div className="hrec-rows">
          {m.items.map((med) => {
            const s = medicationSchedule(med);
            const parts = [
              s.kind === 'daily' ? t('medications.daily') : t('medications.weekly', { count: s.days }),
              med.times.length ? formatNumber(med.times.join('، '), loc) : null,
              med.notes,
            ].filter(Boolean);
            return (
              <Row
                key={med.id}
                label={med.dose ? `${med.title} ${med.dose}` : med.title}
                value={parts.join(' · ')}
              />
            );
          })}
        </div>
      ) : null}
      {listed.length > 0 ? (
        <Row label={t('medications.profileTitle')} value={listed.map((c) => t(`medications.profile.${c}`)).join('، ')} />
      ) : null}
    </SectionCard>
  );
}

function AllergiesCard({ record, onEdit }: { record: HealthRecord; onEdit: () => void }) {
  const t = useTranslations('healthRecord');
  const a = record.allergies.data;
  let content: ReactNode;
  if (a.items === null) content = <Empty>{t('notAnswered')}</Empty>;
  else if (a.items.length === 0) content = <Empty>{t('allergies.none')}</Empty>;
  else content = <p className="hrec-text">{a.items.join('، ')}</p>;
  return (
    <SectionCard icon="warning" tone="warm" title={t('sections.allergies')} onEdit={record.allergies.editable ? onEdit : undefined}>
      {content}
    </SectionCard>
  );
}

function CycleCard({ record }: { record: HealthRecord }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const c = record.cycle.data;
  const num = (v: number) => formatNumber(v, loc);
  const dash = '—';
  return (
    <SectionCard icon="drop" tone="period" title={t('sections.cycle', { count: c.basedOn })}>
      {record.cycle.empty ? (
        <Empty>{t('cycle.empty')}</Empty>
      ) : (
        <>
          <div className="hrec-tiles">
            <Tile
              label={t('cycle.length')}
              value={
                c.medianCycle !== null
                  ? [t('cycle.lengthValue', { days: num(c.medianCycle) }), c.variation !== null ? t('cycle.variation', { days: num(c.variation) }) : null]
                      .filter(Boolean)
                      .join(' · ')
                  : dash
              }
            />
            <Tile label={t('cycle.period')} value={c.medianPeriod !== null ? t('cycle.periodValue', { days: num(c.medianPeriod) }) : dash} />
            <Tile label={t('cycle.lastPeriod')} value={c.lastPeriodStart ? formatDayMonth(fromApiDate(c.lastPeriodStart), loc) : dash} />
            <Tile label={t('cycle.regularity')} value={t(`cycle.${regularityKey(c.regularity)}`)} />
          </div>
          {c.topSymptoms.length > 0 ? (
            <p className="hrec-foot">{t('cycle.topSymptoms', { list: c.topSymptoms.map((s) => s.label).join('، ') })}</p>
          ) : null}
        </>
      )}
    </SectionCard>
  );
}

function VitalsCard({ record, onOpen }: { record: HealthRecord; onOpen: () => void }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const v = record.vitals.data;
  const num = (n: number) => formatNumber(n, loc);
  const glucose = (g: ValueSummary) =>
    g.inTargetPercent !== null
      ? t('vitals.glucoseTarget', { avg: num(g.avg), percent: num(g.inTargetPercent) })
      : t('vitals.glucoseValue', { avg: num(g.avg) });
  return (
    <SectionCard
      icon="gauge"
      tone="data"
      title={t('sections.vitals', { days: num(v.days) })}
      action={{ label: t('vitals.open'), onClick: onOpen }}
    >
      {record.vitals.empty ? (
        <Empty>{t('vitals.empty')}</Empty>
      ) : (
        <div className="hrec-rows">
          {v.bloodPressure ? (
            <Row
              label={t('vitals.bp')}
              value={t('vitals.bpValue', {
                sys: num(v.bloodPressure.systolic),
                dia: num(v.bloodPressure.diastolic),
                count: v.bloodPressure.readings,
              })}
            />
          ) : null}
          {v.heartRate ? <Row label={t('vitals.hr')} value={t('vitals.hrValue', { bpm: num(v.heartRate.avg) })} /> : null}
          {v.glucoseFasting ? <Row label={t('vitals.fasting')} value={glucose(v.glucoseFasting)} /> : null}
          {v.glucoseAfterMeal ? <Row label={t('vitals.afterMeal')} value={glucose(v.glucoseAfterMeal)} /> : null}
          {v.glucoseOther ? <Row label={t('vitals.otherGlucose')} value={glucose(v.glucoseOther)} /> : null}
        </div>
      )}
    </SectionCard>
  );
}

function entryValue(e: PregnancyEntry, t: T, loc: Locale): string {
  const parts: string[] = [];
  if (e.outcome === 'ongoing') {
    return e.date ? t('pregnancies.ongoingDue', { date: formatLongDate(fromApiDate(e.date), loc) }) : t('pregnancies.outcome.ongoing');
  }
  if (e.outcome !== 'birth') parts.push(t(`pregnancies.outcome.${e.outcome}`));
  if (e.date) parts.push(formatNumber(toParts(fromApiDate(e.date), loc).year, loc));
  if (e.babyCount && e.babyCount > 1) parts.push(t('pregnancies.babies', { count: e.babyCount }));
  return parts.join(' · ') || t('pregnancies.outcome.birth');
}

function PregnanciesCard({ record, onEdit }: { record: HealthRecord; onEdit: () => void }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const p = record.pregnancies.data;
  const birthRows = births(p.items);
  const others = p.items.filter((e) => !birthRows.includes(e));
  return (
    <SectionCard icon="mother" tone="bloom" title={t('sections.pregnancies')} onEdit={record.pregnancies.editable ? onEdit : undefined}>
      {record.pregnancies.empty ? (
        <Empty>{t('pregnancies.empty')}</Empty>
      ) : (
        <div className="hrec-rows">
          <Row label={t('pregnancies.count')} value={formatNumber(p.pregnanciesCount, loc)} />
          {birthRows.map((e, i) => (
            <Row key={`b-${e.id ?? i}`} label={t('pregnancies.births')} value={entryValue(e, t, loc)} />
          ))}
          {others.map((e, i) => (
            <Row key={`o-${e.id ?? i}`} label={t(`pregnancies.outcome.${e.outcome}`)} value={e.outcome === 'ongoing' ? entryValue(e, t, loc) : e.date ? formatNumber(toParts(fromApiDate(e.date), loc).year, loc) : undefined} />
          ))}
        </div>
      )}
    </SectionCard>
  );
}

function CheckupsCard({
  record,
  onCheckups,
  onLab,
}: {
  record: HealthRecord;
  onCheckups: () => void;
  onLab: (id: number) => void;
}) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const month = (iso: string) => {
    const d = toParts(fromApiDate(iso), loc);
    return formatMonthLabel(d.year, d.month, loc);
  };
  const result = (code: string) =>
    code === 'normal' || code === 'follow_up' || code === 'pending' ? t(`checkups.result.${code}`) : null;
  const checkups = record.checkups.data.items;
  const labs = record.labs.data.items;
  const empty = record.checkups.empty && record.labs.empty;
  return (
    <SectionCard icon="shield" tone="success" title={t('sections.checkups')}>
      {empty ? (
        <Empty>{t('checkups.empty')}</Empty>
      ) : (
        <div className="hrec-rows">
          {checkups.map((c) => (
            <Row key={`c-${c.id}`} label={c.title} value={[month(c.doneOn), result(c.result)].filter(Boolean).join(' · ')} onClick={onCheckups} />
          ))}
          {labs.map((l) => (
            <Row
              key={`l-${l.id}`}
              label={l.title}
              value={[
                month(l.date),
                l.allNormal
                  ? t('checkups.labAllNormal')
                  : l.attention.length
                    ? l.attention.map((a) => `${a.name} ${a.stateLabel}`).join('، ')
                    : t('checkups.labMarkers', { count: l.markerCount }),
              ].join(' · ')}
              onClick={() => onLab(l.id)}
            />
          ))}
        </div>
      )}
    </SectionCard>
  );
}
