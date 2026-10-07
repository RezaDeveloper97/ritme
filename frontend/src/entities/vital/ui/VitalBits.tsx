'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { IconCircle, StatusPill, type IconName, type Tone } from '@/shared/ui';

import { uiTone } from '../model/classify';
import type { GlucoseUnit, VitalClass, VitalReading, VitalType } from '../model/types';
import { useVitalFormat } from './useVitalFormat';

/** Icon + accent of each type (nbl_Vitals_Hub: BP violet dial, heart rate rose, glucose amber drop). */
export const VITAL_LOOK: Record<VitalType, { icon: IconName; tone: Tone }> = {
  bp: { icon: 'gauge', tone: 'brand' },
  hr: { icon: 'heartLine', tone: 'bloom' },
  glucose: { icon: 'glucose', tone: 'warm' },
};

export function VitalIcon({ type, size = 'sm' }: { type: VitalType; size?: 'sm' | 'md' | 'lg' }) {
  return <IconCircle icon={VITAL_LOOK[type].icon} tone={VITAL_LOOK[type].tone} size={size} />;
}

/** The class verdict pill, coloured by the server's tone. */
export function ClassPill({ type, cls, children }: { type: VitalType; cls: VitalClass | null; children?: ReactNode }) {
  const f = useVitalFormat();
  if (!cls && !children) return null;
  return (
    <StatusPill tone={uiTone(cls?.tone)} className="vt-pill">
      {children ?? (cls ? f.classLabel(type, cls.code) : null)}
    </StatusPill>
  );
}

/**
 * One reading in a list (hub «ثبت‌های اخیر», report «همه ثبت‌ها»): icon disc,
 * value line, conditions + when, class pill. `action` is an end control
 * (delete) for editable readings; log-sheet values carry «از ثبت روزانه».
 */
export function ReadingRow({
  reading,
  withType,
  action,
  className,
  glucoseUnit,
}: {
  reading: VitalReading;
  withType: boolean;
  action?: ReactNode;
  className?: string;
  /** The screen's glucose unit, so one list never mixes mg/dL and mmol/L. */
  glucoseUnit?: GlucoseUnit;
}) {
  const f = useVitalFormat();
  const t = useTranslations('vitals');
  return (
    <li className={clsx('vt-row', className)}>
      <VitalIcon type={reading.type} />
      <div className="vt-row-text">
        <b className="vt-row-title">
          <bdi dir="ltr">{f.value(reading, glucoseUnit)}</bdi>
          {reading.bloodPressure?.pulse ? (
            <>
              {t('common.separator')}
              {t('common.pulseValue', { n: f.num(reading.bloodPressure.pulse) })}
            </>
          ) : reading.bloodPressure ? null : (
            <>
              {' '}
              <bdi dir="ltr">{f.unit(reading, glucoseUnit)}</bdi>
            </>
          )}
        </b>
        <span className="vt-row-meta">{f.meta(reading, withType)}</span>
      </div>
      <ClassPill type={reading.type} cls={reading.classification} />
      {action}
    </li>
  );
}
