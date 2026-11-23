'use client';

import { useTranslations } from 'next-intl';

import { ReadingRow, useVitalReadings } from '@/entities/vital';
import { Link } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { toApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { IconCircle } from '@/shared/ui';
import '@/shared/ui/styles/log.css';

/**
 * The selected day's vitals (BP, heart rate, glucose — timed readings and log-sheet values, newest first) under the
 * day's log recap. Draws nothing on a day without readings, so the calendar stays quiet for most days. Health data
 * never leaves this component (CLAUDE.md §11).
 */
export function DayVitals({ selectedDate }: { selectedDate: Date }) {
  const t = useTranslations('calendar.dayVitals');
  const mounted = useMounted();
  const day = toApiDate(selectedDate);
  const readings = useVitalReadings(null, day, day);
  const items = readings.data ?? [];

  if (!mounted || items.length === 0) return null;

  return (
    <section className="card pad-card cal-vitals" aria-labelledby="cal-vitals-title">
      <header className="cal-vitals-head">
        <IconCircle icon="heartLine" tone="bloom" size="sm" />
        <h3 id="cal-vitals-title" className="cal-vitals-title">
          {t('title')}
        </h3>
        <Link href="/vitals" className="cal-vitals-link" {...pilot('calendar.vitals')}>
          {t('open')}
        </Link>
      </header>
      <ul className="cal-vitals-list">
        {items.map((r, i) => (
          <ReadingRow key={r.id ?? `log-${r.type}-${i}`} reading={r} withType />
        ))}
      </ul>
    </section>
  );
}
