'use client';

import { useTranslations } from 'next-intl';
import { useRef, type KeyboardEvent } from 'react';

import {
  ACCESS_LEVELS,
  type AccessLevel,
  COMPANION_SECTIONS,
  type CompanionGrants,
  type CompanionSection,
} from '@/entities/companion';
import { useDirection } from '@/shared/i18n';
import { IconCircle, type IconName, type Tone } from '@/shared/ui';

/** Icon + tone of each section row (Hamdam_Access). */
const SECTION_LOOK: Record<CompanionSection, { icon: IconName; tone: Tone }> = {
  cycle: { icon: 'drop', tone: 'period' },
  symptoms: { icon: 'smile', tone: 'bloom' },
  meds: { icon: 'pill', tone: 'data' },
  appointments: { icon: 'calendar', tone: 'brand' },
  pregnancy: { icon: 'user', tone: 'bloom' },
};

/** Next radio index for an arrow / Home / End key, wrapping (APG radiogroup). */
function radioStep(key: string, index: number, count: number, rtl: boolean): number | null {
  if (key === (rtl ? 'ArrowLeft' : 'ArrowRight') || key === 'ArrowDown') return (index + 1) % count;
  if (key === (rtl ? 'ArrowRight' : 'ArrowLeft') || key === 'ArrowUp') return (index - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return null;
}

interface LevelPickerProps {
  section: CompanionSection;
  value: AccessLevel;
  onChange: (level: AccessLevel) => void;
  disabled?: boolean;
}

/** «نبیند · فقط دیدن · دیدن و ویرایش» as a radiogroup (selected = brand-soft + brand outline). */
function LevelPicker({ section, value, onChange, disabled }: LevelPickerProps) {
  const t = useTranslations('companions');
  const rtl = useDirection() === 'rtl';
  const refs = useRef<Array<HTMLButtonElement | null>>([]);

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const next = radioStep(event.key, index, ACCESS_LEVELS.length, rtl);
    if (next === null) return;
    event.preventDefault();
    onChange(ACCESS_LEVELS[next]);
    refs.current[next]?.focus();
  };

  return (
    <div
      role="radiogroup"
      aria-label={t('flow.access.levelLabel', { section: t(`sections.${section}`) })}
      className="cmp-levels"
    >
      {ACCESS_LEVELS.map((level, index) => (
        <button
          key={level}
          ref={(el) => {
            refs.current[index] = el;
          }}
          type="button"
          role="radio"
          aria-checked={value === level}
          tabIndex={value === level ? 0 : -1}
          disabled={disabled}
          className="cmp-level"
          onClick={() => onChange(level)}
          onKeyDown={(event) => onKeyDown(event, index)}
        >
          {t(`levels.${level}`)}
        </button>
      ))}
    </div>
  );
}

interface AccessEditorProps {
  value: CompanionGrants;
  onChange: (grants: CompanionGrants) => void;
  disabled?: boolean;
}

/** The per-section access card of Hamdam_Access (also the detail screen's editor). */
export function AccessEditor({ value, onChange, disabled }: AccessEditorProps) {
  const t = useTranslations('companions');
  return (
    <div className="nb-card cmp-access">
      {COMPANION_SECTIONS.map((section) => (
        <div key={section} className="cmp-access-row">
          <div className="cmp-access-head">
            <IconCircle icon={SECTION_LOOK[section].icon} tone={SECTION_LOOK[section].tone} size="sm" />
            <span className="cmp-access-title">{t(`sections.${section}`)}</span>
          </div>
          <LevelPicker
            section={section}
            value={value[section]}
            disabled={disabled}
            onChange={(level) => onChange({ ...value, [section]: level })}
          />
        </div>
      ))}
    </div>
  );
}
