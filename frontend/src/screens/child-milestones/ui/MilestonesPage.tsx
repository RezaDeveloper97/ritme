'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState, type ReactNode } from 'react';

import { useChild } from '@/entities/child';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  Checkbox,
  EmptyState,
  Icon,
  IconCircle,
  PrimaryButton,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { isReadOnlyError, useCheckMilestone, useMilestones } from '../api/queries';
import { chipUnit } from '../model/months';
import type { MilestonesView } from '../model/types';

type T = ReturnType<typeof useTranslations<'children'>>;

function Shell({ header, children }: { header: ReactNode; children: ReactNode }) {
  return (
    <div className="view chd-screen cms-screen">
      <SkyLayer />
      <div className="scroll">
        {header}
        <div className="chd-body">{children}</div>
      </div>
    </div>
  );
}

/**
 * `/children/[id]/milestones` (nbl_v16_Milestones): month chips (the child's
 * band preselected), what was seen this month as a checklist — observation,
 * not a score («هر کودک ریتم خودش را دارد») — play ideas and the calm
 * «با پزشک در میان بگذار» note. A spouse sees the list read-only.
 */
export function ChildMilestonesPage({ id }: { id: number }) {
  const t = useTranslations('children');
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const child = useChild(valid ? id : null);
  const [month, setMonth] = useState<number | null>(null);
  const view = useMilestones(valid ? id : 0, month);
  const check = useCheckMilestone(valid ? id : 0);

  const header = (
    <ScreenHeader
      title={t('section.milestones')}
      subtitle={view.data?.band.label}
      onBack={() => router.push(valid ? `/children/${id}` : '/children')}
      backLabel={t('common.back')}
    />
  );

  if (!valid || getApiErrorStatus(child.error) === 404 || getApiErrorStatus(view.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="sprout"
          title={t('home.notFoundTitle')}
          body={t('home.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/children')}>{t('home.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }

  if (!mounted || child.isPending || view.isPending) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (child.isError || view.isError) {
    return (
      <Shell header={header}>
        <Card className="chd-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="chd-state-text">{t('common.loadError')}</p>
          <SecondaryButton
            icon="refresh"
            block={false}
            loading={view.isFetching}
            onClick={() => {
              void child.refetch();
              void view.refetch();
            }}
          >
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const data = view.data;
  const canEdit = child.data.canEdit;
  return (
    <Shell header={header}>
      <MonthChips data={data} onPick={setMonth} t={t} />
      {child.data.role === 'shared' ? (
        <p className="cgr-readonly">
          <Icon name="eye" size={14} />
          {child.data.ownerName ? t('home.sharedBy', { name: child.data.ownerName }) : t('home.readOnly')}
        </p>
      ) : null}
      <Checklist
        data={data}
        canEdit={canEdit}
        pending={check.isPending ? (check.variables ?? null) : null}
        onToggle={(code, checked) => check.mutate({ code, checked })}
        t={t}
      />
      {check.isError ? (
        <p className="chd-error" role="alert">
          {isReadOnlyError(check.error) ? t('growth.form.readOnly') : t('milestones.saveError')}
        </p>
      ) : null}
      {data.band.activities.length > 0 ? (
        <Card as="section" className="cms-acts" aria-labelledby="cms-acts-title">
          <h2 id="cms-acts-title" className="chd-card-title">
            {t('milestones.activities')}
          </h2>
          <ul className="cms-act-list">
            {data.band.activities.map((a) => (
              <li key={a.code} className="cms-act">
                <IconCircle icon="smile" tone="success" size="sm" />
                <span className="cms-act-text">
                  <b className="cms-act-title">{a.title}</b>
                  {a.body ? <span className="cms-act-body">{a.body}</span> : null}
                </span>
              </li>
            ))}
          </ul>
        </Card>
      ) : null}
      {data.band.doctorNote ? (
        <aside role="note" className="cms-doctor">
          <IconCircle icon="stetho" tone="warm" />
          <p>{data.band.doctorNote}</p>
        </aside>
      ) : null}
    </Shell>
  );
}

// ── Month chips ────────────────────────────────────────────────
function MonthChips({ data, onPick, t }: { data: MilestonesView; onPick: (m: number) => void; t: T }) {
  const locale = useLocale() as Locale;
  const selected = useRef<HTMLButtonElement | null>(null);
  // Keep the selected chip in view (the row scrolls sideways).
  useEffect(() => {
    selected.current?.scrollIntoView({ block: 'nearest', inline: 'center' });
  }, [data.band.months]);
  return (
    <div className="cms-chips" role="group" aria-label={t('milestones.monthsLabel')}>
      {data.bands.map((b) => {
        const on = b.months === data.band.months;
        const { unit, n } = chipUnit(b.months);
        return (
          <button
            key={b.months}
            ref={on ? selected : undefined}
            type="button"
            className={clsx('cms-chip', on && 'is-on', b.current && 'is-current')}
            aria-pressed={on}
            aria-label={b.label}
            onClick={() => onPick(b.months)}
          >
            {t(unit === 'years' ? 'milestones.chipYears' : 'milestones.chipMonths', { n, count: formatNumber(n, locale) })}
          </button>
        );
      })}
    </div>
  );
}

// ── Checklist ──────────────────────────────────────────────────
function Checklist({
  data,
  canEdit,
  pending,
  onToggle,
  t,
}: {
  data: MilestonesView;
  canEdit: boolean;
  pending: { code: string; checked: boolean } | null;
  onToggle: (code: string, checked: boolean) => void;
  t: T;
}) {
  const locale = useLocale() as Locale;
  const items = data.band.items.map((it) => (pending?.code === it.code ? { ...it, checked: pending.checked } : it));
  const checked = items.filter((it) => it.checked).length;
  const total = items.length;
  return (
    <Card as="section" className="cms-card" aria-labelledby="cms-seen">
      <div className="cms-head">
        <ProgressRing
          value={total > 0 ? checked / total : 0}
          label={t('milestones.ring', { checked: formatNumber(checked, locale), total: formatNumber(total, locale) })}
          size={64}
          thickness={7}
          tone="warm"
          className="cvx-ring"
        >
          <span className="cvx-ring-num">
            {formatNumber(checked, locale)}/{formatNumber(total, locale)}
          </span>
        </ProgressRing>
        <div className="cms-head-text">
          <b id="cms-seen" className="cvx-summary-title">
            {t('milestones.seen', { n: checked, count: formatNumber(checked, locale) })}
          </b>
          {data.intro ? <span className="cvx-summary-sub">{data.intro}</span> : null}
        </div>
      </div>
      {total === 0 ? (
        <p className="cvx-empty">{t('milestones.empty')}</p>
      ) : (
        <div className="cms-items" role="group" aria-label={t('milestones.itemsLabel', { label: data.band.label })}>
          {items.map((it) => (
            <Checkbox
              key={it.code}
              className={clsx('cms-item', !canEdit && 'is-readonly')}
              checked={it.checked}
              disabled={!canEdit || pending?.code === it.code}
              label={it.title}
              onCheckedChange={(next) => onToggle(it.code, next)}
            />
          ))}
        </div>
      )}
    </Card>
  );
}
