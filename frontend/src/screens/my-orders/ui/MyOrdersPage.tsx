'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type ShopOrderSummary, useShopOrders, useShopPrice } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer, StatusPill } from '@/shared/ui';

const STATUS_KEYS = ['placed', 'confirmed', 'shipped', 'delivered', 'cancelled'] as const;

function OrderCard({ o }: { o: ShopOrderSummary }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const price = useShopPrice();
  const key = (STATUS_KEYS as readonly string[]).includes(o.status) ? (o.status as (typeof STATUS_KEYS)[number]) : 'placed';
  const sellers = o.shipments.map((s) => s.seller?.name).filter(Boolean).join(' · ');
  return (
    <li>
      <Link href={`/shop/order/${o.id}`} className="shmy-card">
        <div className="shmy-head">
          <span className="shmy-code">{t('order.codeOnly', { code: formatNumber(o.code, locale) })}</span>
          <StatusPill tone={o.status === 'cancelled' ? 'neutral' : o.status === 'delivered' ? 'success' : 'brand'}>{t(`order.status.${key}`)}</StatusPill>
        </div>
        {sellers ? <span className="shmy-sellers">{sellers}</span> : null}
        <span className="shmy-meta">
          {t('orders.count', { n: formatNumber(o.items_count, locale) })}
          {o.placed_at ? ` · ${formatDayMonth(new Date(o.placed_at), locale)}` : ''}
        </span>
        <span className="shmy-foot">
          <span className="shp-price">{price(o.payable_total)}</span>
          <Icon name={locale === 'fa' ? 'chevronLeft' : 'chevronRight'} size={18} />
        </span>
      </Link>
    </li>
  );
}

/**
 * `/profile/orders` — «سفارش‌ها» from Me (CB-SHOP-08; replaces bloom B-N10-03's partner-orders row, DECISIONS #15).
 * Newest first; each card opens the order detail. Pay on delivery only.
 */
export function MyOrdersPage() {
  const t = useTranslations('shop');
  const router = useRouter();
  const q = useShopOrders();
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')}>
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (status === 403) {
    body = (
      <Card>
        <EmptyState icon="shield" title={t('orders.teenTitle')} />
      </Card>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <SecondaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
              {t('retry')}
            </SecondaryButton>
          }
        />
      </Card>
    );
  } else if (q.data.length === 0) {
    body = (
      <Card>
        <EmptyState
          icon="box"
          title={t('orders.emptyTitle')}
          body={t('orders.emptyBody')}
          action={<PrimaryButton onClick={() => router.push('/shop')}>{t('orders.browse')}</PrimaryButton>}
        />
      </Card>
    );
  } else {
    body = (
      <ul className="shmy-list">
        {q.data.map((o) => (
          <OrderCard key={o.id} o={o} />
        ))}
      </ul>
    );
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('orders.title')} onBack={() => router.push('/profile')} backLabel={t('orders.back')} />
        <div className="shp-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
