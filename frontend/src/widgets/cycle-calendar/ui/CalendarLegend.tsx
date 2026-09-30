import type { LegendKey } from '../model/tones';

interface CalendarLegendProps {
  items: readonly LegendKey[];
  label: (key: LegendKey) => string;
  /** Accessible name of the list, e.g. «راهنمای رنگ‌ها». */
  title: string;
}

/** Colour key under the months — one dot per tone, in the artboard's order. */
export function CalendarLegend({ items, label, title }: CalendarLegendProps) {
  return (
    <div className="nb-card cc-legend">
      <ul className="cc-legend-list" aria-label={title}>
        {items.map((key) => (
          <li key={key} className="cc-legend-item">
            <span aria-hidden className={`cc-legend-dot is-${key}`} />
            {label(key)}
          </li>
        ))}
      </ul>
    </div>
  );
}
