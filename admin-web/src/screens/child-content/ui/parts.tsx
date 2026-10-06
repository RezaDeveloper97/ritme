'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { useCurrentAdmin } from '@/features/auth';
import { Badge, Button, LinkTabs, RowActions, Switch, confirm, toast, useNotifyError } from '@/shared/ui';

import { usePatchChildItem, useRemoveChildItem, type ChildItem } from '../api/child-content';
import { KINDS, isIncomplete, type ChildKind } from '../lib/meta';

/** Child catalogs are clinical content: only super admins write (admin-api.md §18; the API answers 403 otherwise). */
export function useCanWrite(): boolean {
  return useCurrentAdmin()?.role === 'super';
}

export type ChildPage = 'vaccines' | 'milestones' | 'learn' | 'who';
const PAGES: ChildPage[] = ['vaccines', 'milestones', 'learn', 'who'];

/** Tabs across the «کودک» pages. */
export function ChildTabs({ active }: { active: ChildPage }) {
  const t = useTranslations('childContent');
  return (
    <LinkTabs
      label={t('tabsLabel')}
      items={PAGES.map((p) => ({ key: p, href: `/children-content/${p}`, label: t(`pages.${p}`), active: p === active }))}
    />
  );
}

/** «نیاز به بازبینی» / «بازبینی‌شده», plus «ناقص» when the app would skip the row. */
export function ReviewBadges({ kind, row }: { kind: ChildKind; row: ChildItem }) {
  const t = useTranslations('childContent');
  return (
    <span className="flex flex-wrap gap-1">
      {row.needs_review ? <Badge tone="amber">{t('needsReview')}</Badge> : <Badge tone="green">{t('reviewed')}</Badge>}
      {row.is_active ? null : <Badge tone="red">{t('inactive')}</Badge>}
      {isIncomplete(kind, row.meta) ? <Badge tone="red">{t('incomplete')}</Badge> : null}
    </span>
  );
}

/** Notice for editors: they read, a super admin writes. */
export function ReadOnlyNotice() {
  const t = useTranslations('childContent');
  return (
    <p className="m-0 rounded-lg border border-line bg-[var(--amber-soft)] px-3 py-2 text-sm text-[var(--amber-deep)]" role="note">
      {t('readOnly')}
    </p>
  );
}

/** Row commands: «بازبینی شد» toggle, active switch, edit, delete (super only; editors get «مشاهده»). */
export function ItemActions({ kind, row, editHref }: { kind: ChildKind; row: ChildItem; editHref: string }) {
  const t = useTranslations('childContent');
  const tc = useTranslations('crud');
  const canWrite = useCanWrite();
  const notifyError = useNotifyError();
  const group = KINDS[kind].group;
  const patch = usePatchChildItem(group);
  const remove = useRemoveChildItem(group);
  const busy = patch.isPending && patch.variables?.row.id === row.id;

  if (!canWrite) {
    return (
      <RowActions>
        <Link href={editHref} className="btn btn-sm">
          {t('view')}
        </Link>
      </RowActions>
    );
  }

  const review = () =>
    patch.mutate(
      { row, patch: { needs_review: !row.needs_review } },
      { onSuccess: () => toast.success(row.needs_review ? t('markedReviewed') : t('markedNeedsReview')), onError: notifyError },
    );
  const toggle = (on: boolean) =>
    patch.mutate({ row, patch: { is_active: on } }, { onSuccess: () => toast.success(tc('statusChanged')), onError: notifyError });
  const del = async () => {
    if (!(await confirm({ message: t('confirmDelete'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
    remove.mutate(row.id, { onSuccess: () => toast.success(tc('deleted')), onError: notifyError });
  };

  return (
    <RowActions>
      <Button size="sm" variant={row.needs_review ? 'primary' : 'ghost'} loading={busy} onClick={review}>
        {row.needs_review ? t('markReviewed') : t('markNeedsReview')}
      </Button>
      <Switch label={row.is_active ? tc('active') : tc('inactive')} checked={row.is_active} disabled={busy} onChange={toggle} />
      <Link href={editHref} className="btn btn-sm">
        {tc('edit')}
      </Link>
      <Button size="sm" variant="danger" loading={remove.isPending && remove.variables === row.id} onClick={del}>
        {tc('delete')}
      </Button>
    </RowActions>
  );
}

/** `/children-content/items/<kind>/<id>` and `/children-content/items/<kind>/new?…`. */
export function itemHref(kind: ChildKind, id: number | null, params?: Record<string, string | number>): string {
  const base = `/children-content/items/${kind}/${id === null ? 'new' : id}`;
  const qs = params ? new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)])).toString() : '';
  return qs ? `${base}?${qs}` : base;
}
