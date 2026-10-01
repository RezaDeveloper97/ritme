import { clsx } from 'clsx';
import type { ReactNode } from 'react';

/** The numbers behind a chart, for screen readers (rendered as a visually hidden table). */
export interface DataTable {
  caption: string;
  columns: readonly string[];
  rows: ReadonlyArray<readonly string[]>;
}

interface ChartFigureProps {
  /** One-sentence accessible name of the picture («طول ۵ سیکل اخیر: ۳۸، ۲۹، …»). */
  label: string;
  table?: DataTable;
  className?: string;
  /** The SVG (decorative children are hidden — the figure carries the name). */
  children: ReactNode;
}

/**
 * Shell of every analysis chart: the drawing is one `role="img"` with an
 * accessible name, and the same numbers sit in a visually hidden table so a
 * screen-reader user can read them value by value (WCAG 1.1.1 / 1.3.1).
 */
export function ChartFigure({ label, table, className, children }: ChartFigureProps) {
  return (
    <figure className={clsx('axc-fig', className)}>
      <div role="img" aria-label={label} className="axc-fig-img">
        {children}
      </div>
      {table && table.rows.length ? (
        <table className="sr-only">
          <caption>{table.caption}</caption>
          <thead>
            <tr>
              {table.columns.map((c) => (
                <th key={c} scope="col">
                  {c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {table.rows.map((row, i) => (
              <tr key={i}>
                {row.map((cell, j) =>
                  j === 0 ? (
                    <th key={j} scope="row">
                      {cell}
                    </th>
                  ) : (
                    <td key={j}>{cell}</td>
                  ),
                )}
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </figure>
  );
}
