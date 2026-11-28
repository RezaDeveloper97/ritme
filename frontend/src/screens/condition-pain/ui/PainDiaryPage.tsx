'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  type ConditionCatalogItem,
  PAIN_ANALGESIC_EFFECTS,
  PROGRAMS_HUB_PATH,
  type PainDay,
  useConditionCatalog,
  useConditions,
  useEnrolCondition,
  usePainCatalog,
  usePainDay,
  useSavePainDay,
} from '@/entities/condition';
import { useLogTaxonomy } from '@/entities/health-log';
import { getApiErrorMessage, getApiErrorStatus, getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber, toApiDate } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  ChipGroup,
  EmptyState,
  IconCircle,
  InfoNote,
  NumericScale,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import { draftFrom, draftProblem, painLocations, type PainDraft, toggleCode, toInput } from '../model/draft';
import './pain.css';

/** The board's chip order: «کمک کرد» · «کمی» · then «نه». */
const EFFECT_ORDER = [...PAIN_ANALGESIC_EFFECTS].reverse();

/**
 * «دفترچه درد» (`/programs/pain`, CB-COND-03, nbl_Cond_Endo): today's endometriosis pain — where (the log taxonomy's
 * pain locations), how bad (0–10), what kind and what came with it (catalog `pain_types` / `pain_associated`), whether
 * it kept her from work or school, and a painkiller with how much it helped → `PUT /conditions/pain/{date}`. Reached
 * from the programs hub and the heavy-pain nudge. A form: no bottom nav.
 */
export function PainDiaryPage() {
  const t = useTranslations('conditions.pain');
  const router = useRouter();
  const [date] = useState(() => toApiDate(new Date()));
  const day = usePainDay(date);

  let body;
  if (day.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="pain-skel">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} shape="card" className="pain-skel-card" />
        ))}
      </SkeletonGroup>
    );
  } else if (day.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <SecondaryButton icon="refresh" block={false} loading={day.isFetching} onClick={() => void day.refetch()}>
            {t('retry')}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = <PainForm key={day.data.date} date={date} saved={day.data} />;
  }

  return (
    <div className="view pain-page">
      <SkyLayer />
      <div className="scroll pain-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={t('subtitle')}
          onBack={() => router.back(PROGRAMS_HUB_PATH)}
          backLabel={t('back')}
        />
        {body}
      </div>
    </div>
  );
}

function chipLabel(items: ConditionCatalogItem[] | undefined, code: string, fallback: string): string {
  return items?.find((i) => i.code === code)?.title ?? fallback;
}

function PainForm({ date, saved }: { date: string; saved: PainDay }) {
  const t = useTranslations('conditions.pain');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const ids = { missed: useId(), name: useId(), time: useId() };
  const taxonomy = useLogTaxonomy();
  const types = usePainCatalog('pain_types', locale);
  const associated = usePainCatalog('pain_associated', locale);
  const alerts = useConditionCatalog('condition_alerts', locale);
  const overview = useConditions();
  const enrol = useEnrolCondition();
  const save = useSavePainDay(date);
  const [draft, setDraft] = useState<PainDraft>(() => draftFrom(saved));
  const [touched, setTouched] = useState(false);

  const set = (patch: Partial<PainDraft>) => setDraft((d) => ({ ...d, ...patch }));
  const locations = painLocations(taxonomy.data, saved.locations);
  const typeCodes = types.data?.map((i) => i.code) ?? [];
  const associatedCodes = associated.data?.map((i) => i.code) ?? [];
  // Keep a saved code visible even if the catalog row was retired since.
  for (const c of draft.types) if (!typeCodes.includes(c)) typeCodes.push(c);
  for (const c of draft.associated) if (!associatedCodes.includes(c)) associatedCodes.push(c);

  const endo = overview.data?.programs.find((p) => p.code === 'endo');
  const notEnrolled = endo ? !endo.enrolled : false;
  const problem = draftProblem(draft);
  const scaleAlert = alerts.data?.find((a) => a.code === 'pain_scale');
  const noPain = draft.score === 0;

  const submit = () => {
    setTouched(true);
    if (problem || notEnrolled || save.isPending) return;
    save.mutate(toInput(draft), { onSuccess: () => router.push(PROGRAMS_HUB_PATH) });
  };

  const saveError = save.isError
    ? getApiErrorStatus(save.error) === 422
      ? (getApiErrorMessage(save.error) ?? t('saveError'))
      : getApiSaveErrorMessage(save.error, t('saveError'))
    : null;

  const analgesicName = draft.analgesic.trim();
  const analgesicLine = analgesicName
    ? draft.analgesicTime
      ? t('analgesicSummary', { name: analgesicName, time: formatNumber(draft.analgesicTime, locale) })
      : analgesicName
    : t('analgesicEmpty');

  return (
    <>
      {notEnrolled ? (
        <section className="nb-card pain-join" aria-label={t('joinTitle')}>
          <h2 className="pain-join-title">{t('joinTitle')}</h2>
          <p className="pain-join-body">{t('joinBody')}</p>
          {enrol.isError ? (
            <p className="pain-error" role="alert">
              {getApiSaveErrorMessage(enrol.error, t('saveError'))}
            </p>
          ) : null}
          <SecondaryButton icon="plus" loading={enrol.isPending} onClick={() => enrol.mutate('endo')} {...pilot('programs.pain-join')}>
            {t('join')}
          </SecondaryButton>
        </section>
      ) : null}

      <section className="pain-section" aria-labelledby={`${ids.missed}-where`}>
        <div className="pain-q">
          <h2 className="pain-q-title" id={`${ids.missed}-where`}>
            {t('whereQ')}
          </h2>
          <p className="pain-q-hint">{noPain ? t('whereNoPain') : t('whereHint')}</p>
        </div>
        <div className="nb-card pain-card">
          {taxonomy.isPending ? (
            <Skeleton shape="block" className="pain-skel-chips" />
          ) : locations.length ? (
            <ChipGroup label={t('whereQ')} className="pain-grid" pilot={pilot('programs.pain-locations', 'choice')}>
              {locations.map((l) => (
                <PillChip
                  key={l.code}
                  mode="multi"
                  shape="square"
                  tone="period"
                  disabled={noPain}
                  pressed={!noPain && draft.locations.includes(l.code)}
                  onPressedChange={() => set({ locations: toggleCode(draft.locations, l.code) })}
                  {...pilotOption(l.code)}
                >
                  {l.label}
                </PillChip>
              ))}
            </ChipGroup>
          ) : (
            <p className="pain-q-hint">{t('noLocations')}</p>
          )}
        </div>
        {touched && problem === 'locations' ? (
          <p className="pain-error" role="alert">
            {t('needLocations')}
          </p>
        ) : null}
      </section>

      <section className="pain-section">
        <div className="nb-card pain-card">
          <NumericScale
            min={0}
            max={10}
            value={draft.score}
            onChange={(score) => set({ score })}
            pilot={pilot('programs.pain-score', 'choice')}
            locale={locale}
            label={<span className="pain-q-title">{t('howBad')}</span>}
            minLabel={t('none')}
            maxLabel={t('worst')}
            variant="solid"
            tone="danger"
          />
        </div>
        {touched && problem === 'score' ? (
          <p className="pain-error" role="alert">
            {t('needScore')}
          </p>
        ) : null}
      </section>

      <section className="pain-section" aria-labelledby={`${ids.missed}-type`}>
        <h2 className="pain-q-title" id={`${ids.missed}-type`}>
          {t('typeQ')}
        </h2>
        <div className="nb-card pain-card">
          <ChipGroup label={t('typeQ')} pilot={pilot('programs.pain-types', 'choice')}>
            {typeCodes.map((code) => (
              <PillChip
                key={code}
                mode="multi"
                tone="period"
                pressed={draft.types.includes(code)}
                onPressedChange={() => set({ types: toggleCode(draft.types, code) })}
                {...pilotOption(code)}
              >
                {chipLabel(types.data, code, code)}
              </PillChip>
            ))}
          </ChipGroup>
          {types.isError ? <p className="pain-q-hint">{t('chipsError')}</p> : null}
        </div>
      </section>

      <section className="pain-section" aria-labelledby={`${ids.missed}-with`}>
        <h2 className="pain-q-title" id={`${ids.missed}-with`}>
          {t('withQ')}
        </h2>
        <div className="nb-card pain-card">
          <ChipGroup label={t('withQ')} pilot={pilot('programs.pain-with', 'choice')}>
            {associatedCodes.map((code) => (
              <PillChip
                key={code}
                mode="multi"
                tone="brand"
                pressed={draft.associated.includes(code)}
                onPressedChange={() => set({ associated: toggleCode(draft.associated, code) })}
                {...pilotOption(code)}
              >
                {chipLabel(associated.data, code, code)}
              </PillChip>
            ))}
          </ChipGroup>
          {associated.isError ? <p className="pain-q-hint">{t('chipsError')}</p> : null}
        </div>
      </section>

      <section className="pain-section" aria-labelledby={`${ids.missed}-impact`}>
        <h2 className="pain-q-title" id={`${ids.missed}-impact`}>
          {t('impactQ')}
        </h2>
        <div className="nb-card pain-card pain-impact">
          <div className="pain-row">
            <span className="pain-row-label" id={ids.missed}>
              {t('missed')}
            </span>
            <Switch
              checked={draft.missedActivity}
              onCheckedChange={(missedActivity) => set({ missedActivity })}
              labelledBy={ids.missed}
              pilot={pilot('programs.pain-missed')}
            />
          </div>
          <div className="pain-med">
            <div className="pain-med-head">
              <IconCircle icon="pill" tone="data" />
              <span className="pain-med-titles">
                <b className="pain-med-title">{t('analgesicQ')}</b>
                <span className="pain-med-sub" dir="auto">
                  {analgesicLine}
                </span>
              </span>
            </div>
            <div className="pain-med-fields">
              <label className="pain-field" htmlFor={ids.name}>
                <span className="pain-field-label">{t('analgesicName')}</span>
                <input
                  id={ids.name}
                  className="pain-input"
                  type="text"
                  maxLength={100}
                  autoComplete="off"
                  placeholder={t('analgesicPlaceholder')}
                  value={draft.analgesic}
                  onChange={(e) => set({ analgesic: e.target.value })}
                  {...pilot('programs.pain-analgesic', 'text')}
                />
              </label>
              <label className="pain-field pain-field-time" htmlFor={ids.time}>
                <span className="pain-field-label">{t('analgesicTime')}</span>
                <input
                  id={ids.time}
                  className="pain-input"
                  type="time"
                  dir="ltr"
                  value={draft.analgesicTime}
                  onChange={(e) => set({ analgesicTime: e.target.value })}
                  {...pilot('programs.pain-analgesic-time', 'text')}
                />
              </label>
            </div>
            <ChipGroup label={t('effectLabel')} className="pain-effects" pilot={pilot('programs.pain-effect', 'choice')}>
              {EFFECT_ORDER.map((effect) => (
                <PillChip
                  key={effect}
                  mode="multi"
                  tone="data"
                  pressed={draft.analgesicEffect === effect}
                  onPressedChange={(on) => set({ analgesicEffect: on ? effect : null })}
                  {...pilotOption(effect)}
                >
                  {t(`effect.${effect}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </div>
        </div>
      </section>

      <InfoNote className="pain-note">
        {scaleAlert?.body ? scaleAlert.body : t('scaleNote')}
      </InfoNote>

      <div className="pain-footer">
        {saveError ? (
          <p className="pain-error" role="alert">
            {saveError}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} disabled={notEnrolled} onClick={submit} {...pilot('programs.pain-save')}>
          {t('save')}
        </PrimaryButton>
      </div>
    </>
  );
}
