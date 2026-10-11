'use client';

import { useSearchParams } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { parseForUserId } from '@/entities/companion';
import {
  CHECKLIST_GROUPS,
  type ChecklistGroupKey,
  type ShopChecklistGroup,
  type ShopChecklistItem,
  useSetChecklistItemState,
  useShopChecklist,
  useUpdateCustomItem,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import {
  Card,
  Checkbox,
  ChipGroup,
  EmptyState,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { ItemSheet } from './ItemSheet';

type Filter = 'all' | 'essential' | 'first' | ChecklistGroupKey;
type T = ReturnType<typeof useTranslations>;

function keepItem(item: ShopChecklistItem, filter: Filter, groupKey: string): boolean {
  if (filter === 'all') return true;
  if (filter === 'essential') return item.is_essential;
  if (filter === 'first') return item.is_first_days;
  return groupKey === filter;
}

/** «۶ عدد دارم» / «در سبد» / «لازم ندارم» — the row's state line (nothing for an untouched row). */
function stateLine(item: ShopChecklistItem, t: T, locale: Locale): string | null {
  if (item.state === 'todo') return null;
  if (item.quantity && item.quantity > 0 && ['have', 'in_cart', 'on_the_way'].includes(item.state)) {
    return t(`qty.${item.state}`, { n: formatNumber(item.quantity, locale) });
  }
  return t(`state.${item.state}`);
}

function Row({
  item,
  readOnly,
  onEdit,
}: {
  item: ShopChecklistItem;
  readOnly: boolean;
  onEdit: (item: ShopChecklistItem) => void;
}) {
  const t = useTranslations('shop.checklist');
  const locale = useLocale() as Locale;
  const setState = useSetChecklistItemState();
  const update = useUpdateCustomItem();
  const have = item.state === 'have';
  const line = stateLine(item, t, locale);
  const toggle = (next: boolean) => {
    const state = next ? 'have' : 'todo';
    if (item.is_custom && item.id !== null) update.mutate({ id: item.id, state });
    else if (item.layette_item_id !== null) setState.mutate({ layetteItemId: item.layette_item_id, state });
  };
  const description = [line, item.note].filter(Boolean).join(' · ');
  return (
    <li className={`shc-row${item.state === 'not_needed' ? ' is-skipped' : ''}`}>
      <Checkbox
        className="shc-check"
        checked={have}
        disabled={readOnly || setState.isPending || update.isPending}
        onCheckedChange={toggle}
        label={item.title}
        description={description || undefined}
      />
      {item.category_slug && !have && item.state !== 'not_needed' ? (
        <Link href={`/shop/${item.category_slug}`} className="shc-see">
          {t('see')}
        </Link>
      ) : null}
      {readOnly ? null : (
        <button type="button" className="shc-edit" aria-label={t('edit', { name: item.title })} onClick={() => onEdit(item)}>
          <Icon name="pen" size={16} />
        </button>
      )}
    </li>
  );
}

function GroupSection({
  group,
  items,
  readOnly,
  onEdit,
}: {
  group: ShopChecklistGroup;
  items: ShopChecklistItem[];
  readOnly: boolean;
  onEdit: (item: ShopChecklistItem) => void;
}) {
  const t = useTranslations('shop.checklist');
  const locale = useLocale() as Locale;
  return (
    <section className="shc-group" aria-labelledby={`shc-g-${group.key}`}>
      <div className="shc-group-head">
        <h2 id={`shc-g-${group.key}`} className="shc-group-title">
          {group.title}
        </h2>
        <span className="shc-group-count">
          {t('groupCount', { ready: formatNumber(group.ready, locale), total: formatNumber(group.total, locale) })}
        </span>
      </div>
      <Card padding="none">
        <ul className="shc-rows">
          {items.map((item) => (
            <Row key={`${item.is_custom ? 'c' : 't'}${item.id ?? item.layette_item_id}`} item={item} readOnly={readOnly} onEdit={onEdit} />
          ))}
        </ul>
      </Card>
    </section>
  );
}

/**
 * `/shop/checklist` — «لیست سیسمونی» (CB-SHOP-09, nbd_Shop_Checklist): progress ring + due date, family share, add
 * item, filter chips, per-group rows with states and «see» links into the shop category, and the "a general list,
 * not a duty; more stuff ≠ more ready" note. Her list only: `?for=<ownerId>` is the family (companion) view, read
 * only. A teen-mode account (404) sees an unavailable state.
 */
export function ShopChecklistPage() {
  const t = useTranslations('shop.checklist');
  const ts = useTranslations('shop');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const forUserId = parseForUserId(useSearchParams().get('for'));
  const q = useShopChecklist(forUserId);
  const [filter, setFilter] = useState<Filter>('all');
  const [sheet, setSheet] = useState<{ item: ShopChecklistItem | null } | null>(null);
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  let subtitle: string | undefined;
  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={ts('loading')} className="shc-body">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (status === 404 || status === 403) {
    body = (
      <Card>
        <EmptyState icon="shield" title={t('unavailableTitle')} body={t('unavailableBody')} />
      </Card>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={ts('loadError')}
          action={
            <PrimaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
              {ts('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else {
    const d = q.data;
    const readOnly = d.read_only;
    if (d.due_date) {
      const date = formatLongDate(fromApiDate(d.due_date), locale);
      subtitle =
        d.weeks_left !== null && d.weeks_left >= 0
          ? t('due', { date, weeks: formatNumber(d.weeks_left, locale) })
          : t('dueDate', { date });
    }
    const groupKeys = d.groups.map((g) => g.key);
    const chipGroups = [...CHECKLIST_GROUPS.filter((k) => groupKeys.includes(k))];
    const visible = d.groups
      .map((g) => ({ group: g, items: g.items.filter((i) => keepItem(i, filter, g.key)) }))
      .filter((g) => g.items.length > 0);
    const defaultGroup = (CHECKLIST_GROUPS as readonly string[]).includes(filter) ? (filter as ChecklistGroupKey) : 'clothing';
    body = (
      <>
        <Card variant="hero" className="shc-sum">
          <ProgressRing
            value={d.summary.percent / 100}
            size={96}
            thickness={10}
            label={t('ringLabel')}
            valueText={t('ringText', { ready: d.summary.ready, total: d.summary.total })}
          >
            <span className="shc-pct">{formatNumber(d.summary.percent, locale)}{locale === 'fa' ? '٪' : '%'}</span>
          </ProgressRing>
          <div className="shc-sum-text">
            <p className="shc-sum-main">
              {t('summary', { ready: formatNumber(d.summary.ready, locale), total: formatNumber(d.summary.total, locale) })}
            </p>
            {d.summary.not_needed > 0 ? (
              <p className="shc-sum-sub">{t('notNeeded', { n: d.summary.not_needed, num: formatNumber(d.summary.not_needed, locale) })}</p>
            ) : null}
            <p className="shc-sum-sub">{t('yours')}</p>
          </div>
        </Card>
        {readOnly ? (
          <InfoNote icon="eye">{t('readOnly')}</InfoNote>
        ) : (
          <div className="shc-actions">
            <SecondaryButton variant="outline" block={false} icon="users" onClick={() => router.push('/companions')}>
              {t('share')}
            </SecondaryButton>
            <SecondaryButton variant="outline" block={false} icon="plus" onClick={() => setSheet({ item: null })}>
              {t('addItem')}
            </SecondaryButton>
          </div>
        )}
        <ChipGroup label={t('filters')} className="shc-chips">
          <PillChip pressed={filter === 'all'} onPressedChange={() => setFilter('all')}>
            {t('filter.all')}
          </PillChip>
          <PillChip pressed={filter === 'essential'} onPressedChange={() => setFilter('essential')}>
            {t('filter.essential')}
          </PillChip>
          <PillChip pressed={filter === 'first'} onPressedChange={() => setFilter('first')}>
            {t('filter.first')}
          </PillChip>
          {chipGroups.map((k) => (
            <PillChip key={k} pressed={filter === k} onPressedChange={() => setFilter(k)}>
              {t(`groups.${k}`)}
            </PillChip>
          ))}
        </ChipGroup>
        {visible.length === 0 ? (
          <Card>
            <EmptyState icon="bag" title={t('emptyTitle')} body={t('emptyBody')} />
          </Card>
        ) : (
          visible.map(({ group, items }) => <GroupSection key={group.key} group={group} items={items} readOnly={readOnly} onEdit={(item) => setSheet({ item })} />)
        )}
        <InfoNote icon="info">{t('note')}</InfoNote>
        {sheet ? <ItemSheet item={sheet.item} defaultGroup={defaultGroup} onClose={() => setSheet(null)} /> : null}
      </>
    );
  }

  return (
    <div className="view shc-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('title')} subtitle={subtitle} onBack={() => router.push('/shop')} backLabel={ts('back')} />
        <div className="shc-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
