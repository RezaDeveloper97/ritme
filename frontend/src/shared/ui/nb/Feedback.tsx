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

interface UrgentCardProps {
  title: ReactNode;
  /** The call-to-action, usually a {@link PrimaryButton} or a `tel:` link. */
  action?: ReactNode;
  icon?: IconName;
  className?: string;
  children?: ReactNode;
}

/**
 * `--danger-soft` card for safety messages (call 115). `role=alert` so it is
 * announced when it appears — mount it only when it is actually urgent.
 */
export function UrgentCard({ title, action, icon = 'warning', className, children }: UrgentCardProps) {
  return (
    <div role="alert" className={clsx('nb-urgent', className)}>
      <div className="nb-urgent-head">
        <IconCircle icon={icon} tone="danger" size="md" />
        <div className="nb-urgent-text">
          <p className="nb-urgent-title">{title}</p>
          {children ? <div className="nb-urgent-body">{children}</div> : null}
        </div>
      </div>
      {action ? <div className="nb-urgent-action">{action}</div> : null}
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
