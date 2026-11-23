'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useShopOrder, useShopPrice, useSlotText } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, PrimaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { APP_SHOP, STORE_HOME } from '../model/store';
import { isTeenRefusal, WebShopGate, WebShopTeen } from './WebShopGate';
import './wsh.css';
import './wsc.css';

const STATUS_KEYS = ['placed', 'confirmed', 'shipped', 'delivered', 'cancelled', 'awaiting_seller'] as const;

function DoneBody({ id }: { id: number }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const locale = useLocale() as Locale;
  const price = useShopPrice();
  const slotText = useSlotText();
  const q = useShopOrder(id);

  if (q.isPending) {
    return (
      <SkeletonGroup label={t('loading')}>
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  }
  if (q.isError) {
    if (isTeenRefusal(q.error)) return <WebShopTeen />;
    const missing = getApiErrorStatus(q.error) === 404;
    return (
      <Card>
        <EmptyState
          icon={missing ? 'bag' : 'warning'}
          title={missing ? t('order.notFoundTitle') : t('loadError')}
          body={missing ? t('order.notFoundBody') : undefined}
          action={
            missing ? (
              <Link href={STORE_HOME} className="wsh-go">
                {t('cart.browse')}
              </Link>
            ) : (
              <PrimaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
                {t('retry')}
              </PrimaryButton>
            )
          }
        />
      </Card>
    );
  }
  const order = q.data;
  return (
    <div className="wsc-done">
      <span className="wsc-done-mark" aria-hidden>
        <Icon name="check" size={40} strokeWidth={2.4} />
      </span>
      <h1 className="wsh-h1">{t('order.placedTitle')}</h1>
      <p className="wsh-lead wsc-done-lead">
        {t('order.codeOnly', { code: formatNumber(order.code, locale) })} · {w('done.body', { amount: price(order.payable_total) })}
      </p>
      <ul className="wsc-done-ships">
        {order.shipments.map((s) => {
          const key = (STATUS_KEYS as readonly string[]).includes(s.status) ? (s.status as (typeof STATUS_KEYS)[number]) : 'placed';
          return (
            <li key={s.id}>
              <Card className="wsc-card">
                <div className="wsc-seller">
                  <h2 className="wsc-seller-name">
                    <Icon name="bag" size={16} />
                    {s.seller?.name ?? t('order.title')}
                  </h2>
                  <span className="wsc-chip">{t(`order.status.${key}`)}</span>
                </div>
                <ul className="wsc-done-items">
                  {s.items.map((it) => (
                    <li key={it.id} className="wsc-done-item">
                      <span className="wsc-thumb" aria-hidden>
                        {it.image_url ? (
                          // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
                          <img src={it.image_url} alt="" loading="lazy" />
                        ) : (
                          <Icon name="bag" size={20} strokeWidth={1.4} />
                        )}
                      </span>
                      <span className="wsc-line-title">{it.title}</span>
                      <span className="wsc-line-sub">{t('order.qty', { n: formatNumber(it.quantity, locale) })}</span>
                    </li>
                  ))}
                </ul>
                <p className="wsc-line-sub">{s.slot ? slotText(s.slot) : t('order.slotNone')}</p>
                <p className="wsh-hint">
                  {t('order.waiting')} · {t('order.waitingHint')}
                </p>
              </Card>
            </li>
          );
        })}
      </ul>
      <div className="wsc-actions">
        <Link href={`${APP_SHOP}/order/${order.id}`} className="wsh-go">
          <Icon name="bag" size={18} />
          {w('done.track')}
        </Link>
        <Link href={STORE_HOME} className="wsc-ghost">
          {w('done.continue')}
        </Link>
      </div>
      <p className="wsh-hint wsc-done-note">{w('done.returns')}</p>
    </div>
  );
}

/** `/store/order/[id]` — «سفارشت ثبت شد» on desktop (CB-SHOP-11, W_Shop_Done): per-seller parcels and the next steps. */
export function WebShopDone({ id }: { id: number }) {
  return (
    <WebShopGate appHref={`${APP_SHOP}/order/${id}`}>
      <DoneBody id={id} />
    </WebShopGate>
  );
}

