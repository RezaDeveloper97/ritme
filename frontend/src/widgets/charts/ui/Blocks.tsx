import { clsx } from 'clsx';
import type { ReactNode } from 'react';

import type { Tone } from '@/shared/ui';

/** A report card (flat surface, 24px radius): title row with an optional aside, then the body. */
export function ChartCard({
  title,
  aside,
  icon,
  className,
  children,
}: {
  /** Omitted when the screen header already names the card (An_Symptoms trend). */
  title?: ReactNode;
  /** End of the title row: a strength pill, a «لوتئال: ۶٫۱» chip. */
  aside?: ReactNode;
  /** Start of the title row: an icon disc. */
  icon?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={clsx('nb-card axc-card', className)}>
      {title || aside || icon ? (
        <div className="axc-card-head">
          {icon}
          {title ? <h2 className="axc-card-title">{title}</h2> : null}
          {aside ? <span className="axc-card-aside">{aside}</span> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}

export interface StatTile {
  key: string;
  label: string;
  value: string;
  /** Verdict under the number («طبیعی (۲۴–۳۸)»), coloured by `tone`. */
  sub?: string;
  tone?: Tone;
}

/** The three summary tiles at the top of a report (An_Cycle / An_Period). */
export function StatTiles({ tiles, label }: { tiles: readonly StatTile[]; label: string }) {
  return (
    <dl className="axc-tiles" aria-label={label}>
      {tiles.map((t) => (
        <div key={t.key} className={clsx('nb-card axc-tile', `nb-tone-${t.tone ?? 'data'}`)}>
          <dt className="axc-tile-label">{t.label}</dt>
          <dd className="axc-tile-value">{t.value}</dd>
          {t.sub ? <dd className="axc-tile-sub">{t.sub}</dd> : null}
        </div>
      ))}
    </dl>
  );
}

export interface HBar {
  key: string;
  label: string;
  /** 0–100. */
  pct: number;
  /** Printed at the end («۸۳٪»). */
  valueLabel: string;
  tone: Tone;
}

/** Labelled horizontal bars (An_Period «علائم هم‌زمان با پریود»); readable as a plain list by screen readers. */
export function HBarList({ bars, label }: { bars: readonly HBar[]; label: string }) {
  return (
    <ul className="axc-hbars" aria-label={label}>
      {bars.map((b) => (
        <li key={b.key} className={clsx('axc-hbar', `nb-tone-${b.tone}`)}>
          <span className="axc-hbar-label">{b.label}</span>
          <svg viewBox="0 0 100 10" preserveAspectRatio="none" className="axc-hbar-track" aria-hidden focusable="false">
            <rect className="axc-hbar-bg" x={0} y={0} width={100} height={10} />
            <rect className="axc-hbar-fill" x={0} y={0} width={Math.max(0, Math.min(100, b.pct))} height={10} />
          </svg>
          <span className="axc-hbar-value">{b.valueLabel}</span>
        </li>
      ))}
    </ul>
  );
}
