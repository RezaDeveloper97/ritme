import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { Icon, type IconName } from '../Icon';
import { IconCircle } from './IconCircle';

interface InfoNoteProps {
  icon?: IconName;
  /** Source line under the note (FIGO, IOM, ADA, WHO). */
  source?: ReactNode;
  className?: string;
  children: ReactNode;
}

/** Small footnote card on `--surface-3`, 11.5/600 `--text-3`. */
export function InfoNote({ icon = 'info', source, className, children }: InfoNoteProps) {
  return (
    <aside role="note" className={clsx('nb-note', className)}>
      <Icon name={icon} size={16} className="nb-note-icon" />
      <div className="nb-note-text">
        {children}
        {source ? <div className="nb-note-source">{source}</div> : null}
      </div>
    </aside>
  );
}

export interface Hotline {
  /** Who answers («اورژانس», «صدای مشاور», «اورژانس اجتماعی»). */
  label: ReactNode;
  /** Dialled number, digits only (`115`, `1480`, `123`). */
  number: string;
  /** Number as shown, in locale digits («۱۱۵»). */
  display: ReactNode;
}

interface UrgentCardProps {
  title: ReactNode;
  /** The call-to-action, usually a {@link PrimaryButton} or a `tel:` link. */
  action?: ReactNode;
  /** Extra actions under the primary one (e.g. «نوبت پزشک زنان» next to «تماس با ۱۱۵»). */
  actions?: ReactNode;
  /** One-tap `tel:` pills (115, 1480, 123), each a ≥ 44px link. */
  hotlines?: readonly Hotline[];
  /** Accessible name of the hotline list, e.g. «شماره‌های کمک». */
  hotlinesLabel?: string;
  icon?: IconName;
  /**
   * `card` = the `--danger-soft` alert card; `note` = the quiet bordered
   * safety line on `--surface` (Cond_PMDD hotlines, IVF_TWW OHSS) with no CTA
   * required — the DangerNote of the canvas.
   */
  variant?: 'card' | 'note';
  /**
   * `true` (default for `card`) mounts as `role=alert` and is announced; a
   * standing safety note (default for `note`) is `role=note`.
   */
  urgent?: boolean;
  className?: string;
  children?: ReactNode;
}

/**
 * `--danger-soft` card for safety messages (call 115). `role=alert` so it is
 * announced when it appears — mount it only when it is actually urgent. The
 * `note` variant is the persistent danger note (hotlines, warning signs).
 */
export function UrgentCard({
  title,
  action,
  actions,
  hotlines,
  hotlinesLabel,
  icon = 'warning',
  variant = 'card',
  urgent,
  className,
  children,
}: UrgentCardProps) {
  const isNote = variant === 'note';
  const announce = urgent ?? !isNote;
  return (
    <div role={announce ? 'alert' : 'note'} className={clsx('nb-urgent', isNote && 'is-note', className)}>
      <div className="nb-urgent-head">
        {isNote ? (
          <Icon name={icon} size={18} className="nb-urgent-icon" />
        ) : (
          <IconCircle icon={icon} tone="danger" size="md" />
        )}
        <div className="nb-urgent-text">
          <p className="nb-urgent-title">{title}</p>
          {children ? <div className="nb-urgent-body">{children}</div> : null}
        </div>
      </div>
      {hotlines && hotlines.length ? (
        <ul className="nb-urgent-hotlines" aria-label={hotlinesLabel}>
          {hotlines.map((line) => (
            <li key={line.number}>
              <a className="nb-hotline" href={`tel:${line.number}`}>
                <Icon name="phone" size={16} />
                <span className="nb-hotline-label">{line.label}</span>
                <span className="nb-hotline-num">{line.display}</span>
              </a>
            </li>
          ))}
        </ul>
      ) : null}
      {action ? <div className="nb-urgent-action">{action}</div> : null}
      {actions ? <div className="nb-urgent-actions">{actions}</div> : null}
    </div>
  );
}

interface EmptyStateProps {
  icon: IconName;
  title: ReactNode;
  body?: ReactNode;
  /** CTA under the copy. */
  action?: ReactNode;
  className?: string;
}

/** Illustration disc + title + copy + CTA, centred. */
export function EmptyState({ icon, title, body, action, className }: EmptyStateProps) {
  return (
    <section className={clsx('nb-empty', className)}>
      <span className="nb-empty-disc" aria-hidden>
        <Icon name={icon} size={40} strokeWidth={1.6} />
      </span>
      <h2 className="nb-empty-title">{title}</h2>
      {body ? <p className="nb-empty-body">{body}</p> : null}
      {action ? <div className="nb-empty-action">{action}</div> : null}
    </section>
  );
}

interface SkeletonProps {
  /** `line` text bar · `block` fixed-height box · `card` card-geometry placeholder · `circle` disc. */
  shape?: 'line' | 'block' | 'card' | 'circle';
  /** `short` / `medium` line widths; lines are full width otherwise. */
  width?: 'short' | 'medium' | 'full';
  className?: string;
}

/** One `--surface-3` shimmer shape (static under reduced motion). Decorative. */
export function Skeleton({ shape = 'line', width = 'full', className }: SkeletonProps) {
  return <span aria-hidden className={clsx('nb-skel', `is-${shape}`, width !== 'full' && `w-${width}`, className)} />;
}

interface SkeletonGroupProps {
  /** Announced while loading, e.g. «در حال بارگذاری». */
  label: string;
  className?: string;
  children: ReactNode;
}

/** Loading region: `role=status` + `aria-busy`, with the label for screen readers only. */
export function SkeletonGroup({ label, className, children }: SkeletonGroupProps) {
  return (
    <div role="status" aria-busy="true" aria-live="polite" className={clsx('nb-skel-group', className)}>
      <span className="sr-only">{label}</span>
      {children}
    </div>
  );
}
