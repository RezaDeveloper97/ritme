'use client';

import clsx from 'clsx';
import { useFormatter, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  IconCircle,
  InfoNote,
  ScreenHeader,
  SecondaryButton,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type IconName,
  type Tone,
} from '@/shared/ui';

import { useSymptomPattern } from '../api/queries';
import { heatRuns, typicalBands } from '../model/bars';
import { PATTERN_GROUPS, type PatternGroup, type SymptomPattern, type SymptomPatternItem } from '../model/schema';

/** Icon + tone per symptom code; unknown codes (a newer backend) fall back to a neutral look. */
const SYMPTOM_LOOK: Record<string, { icon: IconName; tone: Tone }> = {
  cramps: { icon: 'symptom', tone: 'period' },
  breast_tenderness: { icon: 'heart', tone: 'bloom' },
  bloating: { icon: 'plus', tone: 'warm' },
  acne: { icon: 'sparkle', tone: 'period' },
  energetic: { icon: 'zap', tone: 'data' },
  fatigue: { icon: 'moon', tone: 'brand' },
  food_craving: { icon: 'apple', tone: 'warm' },
  nausea: { icon: 'drop', tone: 'data' },
  happy: { icon: 'smile', tone: 'warm' },
  calm: { icon: 'sun', tone: 'data' },
  sensitive: { icon: 'heart', tone: 'bloom' },
  anxious: { icon: 'brain', tone: 'brand' },
  sad: { icon: 'moon', tone: 'brand' },
  angry: { icon: 'flame', tone: 'period' },
  frustrated: { icon: 'warning', tone: 'warm' },
  bored: { icon: 'clock', tone: 'neutral' },
  headache: { icon: 'brain', tone: 'brand' },
  stomach_ache: { icon: 'symptom', tone: 'period' },
  pelvic_pain: { icon: 'symptom', tone: 'period' },
  back_pain: { icon: 'user', tone: 'warm' },
  breast_pain: { icon: 'heart', tone: 'bloom' },
  ovarian_pain: { icon: 'target', tone: 'brand' },
};
const lookOf = (key: string) => SYMPTOM_LOOK[key] ?? { icon: 'symptom' as IconName, tone: 'neutral' as Tone };

/**
 * «الگوی علائم» (B-N1-08, `nbl_/nbd_Cycle_Symptoms`) at `/cycle/symptoms`: one
 * heat strip per logged symptom over the user's typical cycle, grouped into
 * symptoms / mood / pain. The backend aligns the cycles and averages the logs
 * (`GET /cycle/symptom-pattern`, ≥ 3 completed cycles); this screen only
 * paints. The copy says co-occurrence, never cause (§11).
 */
export function CycleSymptomsPage() {
  const t = useTranslations('cycle');
  const router = useRouter();
  const mounted = useMounted();
  const query = useSymptomPattern();
  const [group, setGroup] = useState<PatternGroup>('symptoms');
  const data = query.data;
  const loading = !mounted || query.isLoading;

  return (
    <div className="view cyh-page">
      <SkyLayer />
      <div className="scroll cyh-scroll">
        <ScreenHeader
          title={t('symptoms.title')}
          subtitle={data?.ready ? t('symptoms.subtitle', { n: data.cyclesCounted }) : undefined}
          onBack={() => router.push('/cycle')}
          backLabel={t('back')}
        />

        {loading ? (
          <SkeletonGroup label={t('loading')} className="cyh-skel">
            <Skeleton shape="block" className="csp-skel-tabs" />
            <Skeleton shape="card" />
            <Skeleton shape="block" className="cyh-skel-list" />
          </SkeletonGroup>
        ) : query.isError || !data ? (
          <Card className="cyh-error" role="alert">
            <p className="cyh-error-t">{t('symptoms.error')}</p>
            <SecondaryButton block={false} onClick={() => query.refetch()}>
              {t('retry')}
            </SecondaryButton>
          </Card>
        ) : !data.ready ? (
          <EmptyState
            icon="chart"
            title={t('symptoms.notReady.title')}
            body={t('symptoms.notReady.body', { needed: data.cyclesNeeded, n: data.cyclesCounted })}
          />
        ) : (
          <>
            <SegmentedTabs
              label={t('symptoms.tabsLabel')}
              tabs={PATTERN_GROUPS.map((g) => ({ value: g, label: t(`symptoms.tabs.${g}`) }))}
              value={group}
              onChange={setGroup}
              panelId={(g) => `csp-panel-${g}`}
            />
            <div id={`csp-panel-${group}`} role="tabpanel" className="csp-panel">
              <GroupPanel data={data} items={data.groups[group]} />
            </div>
          </>
        )}

        <p className="csp-note">{t('symptoms.note')}</p>
      </div>
    </div>
  );
}

function GroupPanel({ data, items }: { data: SymptomPattern; items: SymptomPatternItem[] }) {
  const t = useTranslations('cycle');
  if (items.length === 0) {
    return <InfoNote>{t('symptoms.emptyGroup')}</InfoNote>;
  }
  const top = items[0];
  return (
    <>
      <HighlightCard item={top} total={data.cyclesCounted} />
      <Card className="csp-card">
        <div className="csp-card-head">
          <h2 className="csp-card-t">{t('symptoms.typical')}</h2>
          <span className="csp-scale">{t('symptoms.scale')}</span>
        </div>
        <TypicalBar data={data} />
        <ul className="csp-rows">
          {items.map((item) => (
            <SymptomRow key={item.key} item={item} total={data.cyclesCounted} />
          ))}
        </ul>
      </Card>
    </>
  );
}

function useSymptomName() {
  const t = useTranslations('cycle');
  // Codes come from the API, so an unknown one (a newer backend) falls back to a generic label.
  return (key: string) => {
    const k = `symptoms.names.${key}` as 'symptoms.names.other';
    return t.has(k) ? t(k) : t('symptoms.names.other');
  };
}

function HighlightCard({ item, total }: { item: SymptomPatternItem; total: number }) {
  const t = useTranslations('cycle');
  const name = useSymptomName();
  const look = lookOf(item.key);
  const w = item.window;
  const text = !w
    ? null
    : w.relation === 'mid'
      ? t('symptoms.window.mid', { n: item.cycles, total, from: w.startDay, to: w.endDay })
      : t(`symptoms.window.${w.relation}`, { n: item.cycles, total, days: w.days });
  return (
    <Card className={clsx('csp-hero', `nb-tone-${look.tone}`)}>
      <div className="csp-hero-b">
        <p className="csp-hero-t">{name(item.key)}</p>
        {text ? <p className="csp-hero-s">{text}</p> : null}
      </div>
      <IconCircle icon={look.icon} tone={look.tone} size="lg" />
    </Card>
  );
}

/** The typical cycle: period, fertile window with the ovulation dot, PMS days; axis day 1 · ovulation · last. */
function TypicalBar({ data }: { data: SymptomPattern }) {
  const t = useTranslations('cycle');
  const format = useFormatter();
  const L = data.cycleLength;
  const b = typicalBands(L, data.periodLength, data.ovulationDay);
  const at = (day: number) => ((day - 1) / L) * 100;
  const span = ([from, to]: [number, number]) => ({ insetInlineStart: `${at(from)}%`, inlineSize: `${((to - from + 1) / L) * 100}%` });
  return (
    <div className="csp-typ">
      <div
        className="csp-typ-bar"
        role="img"
        aria-label={t('symptoms.bandsLabel', { period: b.period[1], ovulation: b.ovulation, length: L })}
      >
        <span className="csp-band is-period" style={span(b.period)} />
        <span className="csp-band is-fertile" style={span(b.fertile)} />
        <span className="csp-band is-pms" style={span(b.pms)} />
        <span className="csp-ovu" style={{ insetInlineStart: `${at(b.ovulation) + 50 / L}%` }} />
      </div>
      <div className="csp-axis" aria-hidden>
        <span>{t('symptoms.day', { n: 1 })}</span>
        <span className="csp-axis-mid" style={{ insetInlineStart: `${at(b.ovulation) + 50 / L}%` }}>
          {format.number(b.ovulation)}
        </span>
        <span>{format.number(L)}</span>
      </div>
    </div>
  );
}

function SymptomRow({ item, total }: { item: SymptomPatternItem; total: number }) {
  const t = useTranslations('cycle');
  const name = useSymptomName();
  const look = lookOf(item.key);
  const L = item.strip.length || 1;
  return (
    <li className="csp-row">
      <div className="csp-row-head">
        <IconCircle icon={look.icon} tone={look.tone} size="sm" />
        <span className="csp-row-t">{name(item.key)}</span>
      </div>
      <div
        className={clsx('csp-strip', `nb-tone-${look.tone}`)}
        role="img"
        aria-label={t('symptoms.stripLabel', { name: name(item.key), n: item.cycles, total })}
      >
        {heatRuns(item.strip).map((run) => (
          <span
            key={run.start}
            className={clsx('csp-heat', `l${run.level}`)}
            style={{ insetInlineStart: `${(run.start / L) * 100}%`, inlineSize: `${(run.length / L) * 100}%` }}
          />
        ))}
      </div>
    </li>
  );
}
