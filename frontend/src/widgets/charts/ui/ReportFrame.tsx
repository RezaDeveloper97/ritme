'use client';

import { clsx } from 'clsx';
import type { ReactNode } from 'react';

import { EmptyState, ScreenHeader, SecondaryButton, SegmentedTabs, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

export interface ReportFrameProps<V extends string> {
  title: string;
  subtitle?: ReactNode;
  backLabel: string;
  onBack: () => void;
  /** Range tabs («۲ هفته · ۱ ماه · …»); omitted = no selector. */
  tabs?: { label: string; value: V; options: ReadonlyArray<{ value: V; label: string }>; onChange: (v: V) => void };
  status: 'loading' | 'error' | 'ready';
  /** A new range is loading while the previous one stays on screen. */
  updating?: boolean;
  loadingLabel: string;
  error: { title: string; body: string; retry: string; onRetry: () => void; retrying: boolean };
  /** Mounted after the scroll area (the bottom nav — a sibling widget, so the screen passes it in). */
  after?: ReactNode;
  children: ReactNode;
}

/**
 * Page frame of an analysis detail report (B-N3-09): back header, optional
 * range tabs, skeleton / error states, the report body. The screen supplies
 * every string (this widget has no namespace of its own).
 */
export function ReportFrame<V extends string>({
  title,
  subtitle,
  backLabel,
  onBack,
  tabs,
  status,
  updating,
  loadingLabel,
  error,
  after,
  children,
}: ReportFrameProps<V>) {
  let body: ReactNode;
  if (status === 'loading') {
    body = (
      <SkeletonGroup label={loadingLabel} className="axc-skel">
        <Skeleton shape="block" className="axc-skel-tiles" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (status === 'error') {
    body = (
      <EmptyState
        icon="warning"
        title={error.title}
        body={error.body}
        action={
          <SecondaryButton icon="refresh" onClick={error.onRetry} loading={error.retrying}>
            {error.retry}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = (
      <div className={clsx('axc-body', updating && 'is-updating')} aria-busy={updating || undefined}>
        {children}
      </div>
    );
  }
  return (
    <div className="view axc-page">
      <SkyLayer />
      <div className="scroll axc-scroll">
        <ScreenHeader title={title} subtitle={subtitle} onBack={onBack} backLabel={backLabel} className="axc-hdr" />
        <div className="axc-content">
          {tabs ? <SegmentedTabs label={tabs.label} value={tabs.value} tabs={tabs.options} onChange={tabs.onChange} /> : null}
          {body}
        </div>
      </div>
      {after}
    </div>
  );
}
