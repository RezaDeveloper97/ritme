import { clsx } from 'clsx';

import type { PaperModel } from '../model/paper';

interface ReportPaperProps {
  model: PaperModel;
  questionLabel: string;
  /** «صفحه ۱ از ۲» under the disclaimer (preview only). */
  pageLabel?: string;
  className?: string;
}

/**
 * The report as paper (nbl_Record_Preview): always the printed document — white page, dark ink — in both themes,
 * like the PDF the device draws from the same {@link PaperModel}. Read-only; every value is plain text.
 */
export function ReportPaper({ model: m, questionLabel, pageLabel, className }: ReportPaperProps) {
  return (
    <article className={clsx('rx-paper', className)} aria-label={m.title}>
      <header className="rx-paper-head">
        <div className="rx-paper-brand">
          <h2 className="rx-paper-title">{m.title}</h2>
          <p className="rx-paper-sub">{m.sub}</p>
        </div>
        {m.name || m.meta ? (
          <div className="rx-paper-person">
            {m.name ? <p className="rx-paper-name">{m.name}</p> : null}
            {m.meta ? <p className="rx-paper-meta">{m.meta}</p> : null}
          </div>
        ) : null}
      </header>

      {m.stats.length ? (
        <dl className="rx-paper-stats">
          {m.stats.map((s) => (
            <div key={s.label} className="rx-paper-stat">
              <dt>{s.label}</dt>
              <dd>
                <bdi>{s.value}</bdi>
              </dd>
            </div>
          ))}
        </dl>
      ) : null}

      {m.blocks.map((b, i) => {
        const key = `${b.kind}-${i}`;
        if (b.kind === 'heading') {
          return (
            <h3 key={key} className="rx-paper-h">
              {b.text}
            </h3>
          );
        }
        if (b.kind === 'row') {
          return (
            <div key={key} className={clsx('rx-paper-row', b.head && 'is-head', `cols-${b.cells.length}`)} role="row">
              {b.cells.map((c, j) => (
                <span key={j} className={clsx('rx-paper-cell', j === 0 && 'is-first')} role={b.head ? 'columnheader' : 'cell'}>
                  <bdi>{c}</bdi>
                </span>
              ))}
            </div>
          );
        }
        return (
          <p key={key} className={b.kind === 'empty' ? 'rx-paper-empty' : 'rx-paper-text'}>
            {b.text}
          </p>
        );
      })}

      {m.question ? (
        <p className="rx-paper-question">
          <strong>{questionLabel}</strong> {m.question}
        </p>
      ) : null}
      <p className="rx-paper-foot">
        {m.disclaimer}
        {pageLabel ? ` ${pageLabel}` : ''}
      </p>
    </article>
  );
}
