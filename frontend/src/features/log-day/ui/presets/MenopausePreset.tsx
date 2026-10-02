'use client';

import { useTranslations } from 'next-intl';

import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import { Card, ChipGroup, Icon, IconCircle, PillChip, SeverityScale, type Tone } from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';

import {
  acceptsNo,
  bleedingChoice,
  bleedingLogged,
  hasRow,
  MENOPAUSE_BLEEDING,
  MENOPAUSE_GROUPS,
  MENOPAUSE_TRIGGERS,
  nextBleeding,
  rowLevel,
  SEVERITY_LEVELS,
  toggleTrigger,
  triggerList,
  withRowLevel,
  type PresetGroup,
  type PresetRow,
  type SeverityLevel,
} from '../../model/menopause-preset';

interface MenopausePresetProps {
  categories: readonly LogCategory[];
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  /** «با صدا بگو» → the sheet's voice tab (B-N3-05). */
  onVoice: () => void;
  /** Voice logging is a Plus feature (`plus.voice_log`): the card carries the badge when locked. */
  voiceLocked: boolean;
  /** «بیشتر بدان» under a logged bleeding → `/menopause/alert` (saves first). */
  onBleedingHelp: () => void;
}

/**
 * The menopause preset of the log sheet (CB-MENO-06, nbl_Meno_Log): «با صدا بگو» card, the five symptom
 * groups as `SeverityScale` rows, the bleeding choice with the post-menopause note and the trigger chips.
 * Edits the sheet's draft like every other section — the sheet's footer saves the day in one PUT.
 */
export function MenopausePreset({ categories, values, setParam, onVoice, voiceLocked, onBleedingHelp }: MenopausePresetProps) {
  const t = useTranslations('logSheet.presets.menopause');
  return (
    <div className="mlog">
      <button type="button" className="mlog-voice" onClick={onVoice}>
        <span className="mlog-voice-mic" aria-hidden>
          <Icon name="mic" size={22} strokeWidth={1.9} />
        </span>
        <span className="mlog-voice-text">
          <span className="mlog-voice-title">
            {t('voice.title')}
            {voiceLocked ? <PlusBadge label={t('voice.plus')} className="mlog-voice-plus" /> : null}
          </span>
          <span className="mlog-voice-body">{t('voice.body')}</span>
        </span>
        <Icon name="chevronLeft" size={18} className="mlog-voice-chev" />
      </button>
      {MENOPAUSE_GROUPS.map((group) => (
        <SymptomGroup key={group.code} group={group} categories={categories} values={values} setParam={setParam} />
      ))}
      <BleedingCard categories={categories} values={values} setParam={setParam} onHelp={onBleedingHelp} />
      <TriggersCard categories={categories} values={values} setParam={setParam} />
    </div>
  );
}

function GroupHead({ id, title, icon, tone }: { id: string; title: string; icon: PresetGroup['icon']; tone: Tone }) {
  return (
    <h3 id={id} className="mlog-head">
      <IconCircle icon={icon} tone={tone} size="sm" />
      <span>{title}</span>
    </h3>
  );
}

type Editable = Pick<MenopausePresetProps, 'categories' | 'values' | 'setParam'>;

function SymptomGroup({ group, categories, values, setParam }: Editable & { group: PresetGroup }) {
  const t = useTranslations('logSheet.presets.menopause');
  const rows = group.rows.filter((row) => hasRow(categories, row));
  if (!rows.length) return null;
  const headId = `mlog-${group.code}`;
  return (
    <Card as="section" className="mlog-card" aria-labelledby={headId}>
      <GroupHead id={headId} title={t(`groups.${group.code}`)} icon={group.icon} tone={group.tone} />
      {rows.map((row) => (
        <SymptomRow key={`${row.category}.${row.param}.${row.item}`} row={row} categories={categories} values={values} setParam={setParam} />
      ))}
    </Card>
  );
}

function SymptomRow({ row, categories, values, setParam }: Editable & { row: PresetRow }) {
  const t = useTranslations('logSheet.presets.menopause');
  const option = categories
    .find((c) => c.code === row.category)
    ?.params.find((p) => p.code === row.param)
    ?.options.find((o) => o.value === row.item);
  // The board's longer row titles («بی‌خوابی یا بیدار شدن شبانه»); the taxonomy label for a language
  // that hasn't translated them yet.
  const label = t.has(`rows.${row.item}`) ? t(`rows.${row.item}`) : (option?.label ?? row.item);
  const noLevel = acceptsNo(categories, row);
  const options = SEVERITY_LEVELS.map((value) => ({ value, label: t(`levels.${value}`) }));
  return (
    <SeverityScale<SeverityLevel>
      className="mlog-row"
      label={label}
      options={options}
      value={rowLevel(values, row)}
      onChange={(level) => setParam(row.category, row.param, withRowLevel(values, row, level, noLevel))}
    />
  );
}

function BleedingCard({ categories, values, setParam, onHelp }: Editable & { onHelp: () => void }) {
  const t = useTranslations('logSheet.presets.menopause');
  const param = categories
    .find((c) => c.code === MENOPAUSE_BLEEDING.category)
    ?.params.find((p) => p.code === MENOPAUSE_BLEEDING.param);
  if (!param) return null;
  const choice = bleedingChoice(values);
  return (
    <Card as="section" className="mlog-card" aria-labelledby="mlog-bleeding">
      <GroupHead id="mlog-bleeding" title={t('groups.bleeding')} icon="drop" tone="period" />
      <ChipGroup label={t('groups.bleeding')}>
        {param.options.map((o) => (
          <PillChip
            key={o.value}
            mode="single"
            tone={o.value === 'none' ? 'data' : 'period'}
            pressed={choice === o.value}
            onPressedChange={() => setParam(MENOPAUSE_BLEEDING.category, MENOPAUSE_BLEEDING.param, nextBleeding(values, o.value))}
          >
            {o.label}
          </PillChip>
        ))}
      </ChipGroup>
      <p className="mlog-note">
        {t('bleeding.note')}
        {bleedingLogged(values) ? (
          <>
            {' '}
            <button type="button" className="mlog-link" onClick={onHelp}>
              {t('bleeding.help')}
            </button>
          </>
        ) : null}
      </p>
    </Card>
  );
}

function TriggersCard({ categories, values, setParam }: Editable) {
  const t = useTranslations('logSheet.presets.menopause');
  const param = categories
    .find((c) => c.code === MENOPAUSE_TRIGGERS.category)
    ?.params.find((p) => p.code === MENOPAUSE_TRIGGERS.param);
  if (!param) return null;
  const picked = triggerList(values);
  return (
    <Card as="section" className="mlog-card" aria-labelledby="mlog-triggers">
      <h3 id="mlog-triggers" className="mlog-head">
        {t('groups.triggers')}
      </h3>
      <ChipGroup label={t('groups.triggers')}>
        {param.options.map((o) => (
          <PillChip
            key={o.value}
            mode="multi"
            tone="brand"
            pressed={picked.includes(o.value)}
            onPressedChange={() => setParam(MENOPAUSE_TRIGGERS.category, MENOPAUSE_TRIGGERS.param, toggleTrigger(values, o.value))}
          >
            {o.label}
          </PillChip>
        ))}
      </ChipGroup>
    </Card>
  );
}
