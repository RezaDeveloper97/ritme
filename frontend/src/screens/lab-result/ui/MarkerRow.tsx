'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatLabValue, formatReference, type LabMarker, MarkerStatePill, RangeBar, stateTone } from '@/entities/lab';
import type { Locale } from '@/shared/i18n';

/** One marker of the result: name, value + unit in the state's tone, the state pill, the range bar and «مرجع: …». A button to the marker detail. */
export function MarkerRow({ marker, onOpen }: { marker: LabMarker; onOpen: () => void }) {
  const t = useTranslations('labs.result');
  const locale = useLocale() as Locale;
  const value = formatLabValue(marker, locale);
  const ref = formatReference(marker.reference, locale);
  const tone = stateTone(marker.state);
  return (
    <li>
      <button type="button" className="lab-mrow" onClick={onOpen}>
        <span className="lab-mrow-top">
          <span className="lab-mrow-name">{marker.name}</span>
          <span className={`lab-mrow-value nb-tone-${tone}`}>
            <span className="lab-mrow-num">{value}</span>
            {marker.unit ? <span className="lab-mrow-unit">{marker.unit}</span> : null}
          </span>
          <MarkerStatePill state={marker.state} label={marker.stateLabel} />
        </span>
        <RangeBar
          value={marker.value}
          low={marker.reference.low}
          high={marker.reference.high}
          tone={tone}
          label={ref ? t('rangeLabel', { value, ref }) : value}
        />
        {ref ? <span className="lab-mrow-ref">{t('reference', { ref })}</span> : null}
      </button>
    </li>
  );
}
