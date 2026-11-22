'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useState } from 'react';

import {
  type ShopCart,
  type ShopCartLine,
  shopCodeKey,
  shopErrorKey,
  useApplyDiscount,
  useRemoveCartItem,
  useRemoveDiscount,
  useSetCartQuantity,
  useShopCart,
  useShopPrice,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { APP_SHOP, STORE_HOME, storeProductHref } from '../model/store';
import { isTeenRefusal, WebShopGate, WebShopTeen } from './WebShopGate';
import '@/shared/ui/styles/shp.css';
import './wsh.css';
import './wsc.css';

const MAX_QTY = 10;

function Line({ line }: { line: ShopCartLine }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const locale = useLocale() as Locale;
  const price = useShopPrice();
  const setQty = useSetCartQuantity();
  const remove = useRemoveCartItem();
  const busy = setQty.isPending || remove.isPending;
  const failed = setQty.isError ? shopErrorKey(setQty.error) : null;
  const out = line.status !== 'ok';
  return (
    <li className={`wsc-line${out ? ' is-out' : ''}`}>
      <Link href={storeProductHref(line.product_id)} className="wsc-art" aria-hidden tabIndex={-1}>
        {line.image_url ? (
          // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
          <img src={line.image_url} alt="" loading="lazy" />
        ) : (
          <Icon name="bag" size={28} strokeWidth={1.4} />
        )}
      </Link>
      <div className="wsc-line-main">
        <Link href={storeProductHref(line.product_id)} className="wsc-line-title">
          {line.title}
        </Link>
        {line.variant_label ? <span className="wsc-line-sub">{line.variant_label}</span> : null}
        <button type="button" className="wsc-remove" disabled={busy} onClick={() => remove.mutate(line.variant_id)}>
          <Icon name="trash" size={14} />
          {w('cart.remove')}
        </button>
        {out ? <span className="wsc-error">{t('cart.outOfStock')}</span> : null}
        {failed ? (
          <span className="wsc-error" role="alert">
            {t(`errors.${failed}`)}
          </span>
        ) : null}
      </div>
      <div className="wsc-qty" role="group" aria-label={t('cart.qty', { title: line.title })}>
        <button
          type="button"
          className="wsc-step"
          disabled={busy || line.quantity <= 1}
          aria-label={t('cart.dec')}
          onClick={() => setQty.mutate({ variantId: line.variant_id, quantity: line.quantity - 1 })}
        >
          <Icon name="minus" size={16} />
        </button>
        <span className="wsc-n" aria-live="polite">
          {formatNumber(line.quantity, locale)}
        </span>
        <button
          type="button"
          className="wsc-step"
          disabled={busy || line.quantity >= MAX_QTY || out}
          aria-label={t('cart.inc')}
          onClick={() => setQty.mutate({ variantId: line.variant_id, quantity: line.quantity + 1 })}
        >
          <Icon name="plus" size={16} />
        </button>
      </div>
      <span className="wsc-line-price shp-price">{out ? '' : price(line.line_total)}</span>
    </li>
  );
}

function DiscountBox({ code, error }: { code: string | null; error: string | null }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const [value, setValue] = useState('');
  const apply = useApplyDiscount();
  const remove = useRemoveDiscount();
  // 429 = the hourly limit on code attempts; every other refusal is the generic «not valid» (never says why).
  const failed = apply.isError ? (getApiErrorStatus(apply.error) === 429 ? 'limit' : shopErrorKey(apply.error)) : null;
  if (code) {
    return (
      <Card className="wsc-disc">
        <p className="wsc-disc-on">{t('cart.discountApplied', { code })}</p>
        {error ? (
          <p className="wsc-error" role="alert">
            {t(`errors.${shopCodeKey(error)}`)}
          </p>
        ) : null}
        <SecondaryButton variant="text" loading={remove.isPending} onClick={() => remove.mutate()}>
          {t('cart.discountRemove')}
        </SecondaryButton>
      </Card>
    );
  }
  return (
    <Card className="wsc-disc">
      <label className="lbl" htmlFor="wsc-code">
        {t('cart.discountPrompt')}
      </label>
      <div className="wsc-disc-row">
        <div className="field">
          <input id="wsc-code" dir="ltr" value={value} maxLength={32} autoComplete="off" placeholder={t('cart.discountPlaceholder')} onChange={(e) => setValue(e.target.value)} />
        </div>
        <SecondaryButton loading={apply.isPending} disabled={!value.trim()} onClick={() => apply.mutate(value.trim(), { onSuccess: () => setValue('') })}>
          {t('cart.discountApply')}
        </SecondaryButton>
      </div>
      {failed ? (
        <p className="wsc-error" role="alert">
          {failed === 'limit' ? w('cart.discountLimit') : t(`errors.${failed}`)}
        </p>
      ) : null}
    </Card>
  );
}

/** Order-summary card shared by the cart and the checkout. */
export function CartSummary({ children, footer }: { children: ReactNode; footer: ReactNode }) {
  const w = useTranslations('webShop');
  return (
    <aside className="wsc-aside" aria-label={w('cart.summary')}>
      <Card className="wsc-summary">
        <h2 className="wsh-h3">{w('cart.summary')}</h2>
        <dl className="wsc-rows">{children}</dl>
        {footer}
      </Card>
      <p className="wsc-privacy">
        <Icon name="lock" size={14} />
        <span>{w('cart.privacy')}</span>
      </p>
    </aside>
  );
}

function FreeShip({ group }: { group: ShopCart['groups'][number] }) {
  const t = useTranslations('shop');
  const price = useShopPrice();
  const over = group.seller.free_shipping_over;
  if (!over) return null;
  const pct = Math.min(100, Math.round((group.items_total / over) * 100));
  return (
    <div className="wsc-free">
      <p className={`wsc-free-text${group.shipping_fee === 0 ? ' is-done' : ''}`}>
        {group.free_shipping_remaining ? t('cart.freeShip', { amount: price(group.free_shipping_remaining) }) : t('cart.freeShipDone')}
      </p>
      <div className="wsc-bar" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct} aria-label={t('cart.freeShipDone')}>
        <span className="wsc-bar-fill" style={{ inlineSize: `${pct}%` }} />
      </div>
    </div>
  );
}

function CartBody() {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const locale = useLocale() as Locale;
  const price = useShopPrice();
  const q = useShopCart();

  if (q.isPending) {
    return (
      <SkeletonGroup label={t('loading')}>
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  }
  if (q.isError) {
    if (isTeenRefusal(q.error)) return <WebShopTeen />;
    return (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <PrimaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  }
  const cart = q.data;
  if (cart.groups.length === 0 && cart.unavailable_items.length === 0) {
    return (
      <Card>
        <EmptyState
          icon="bag"
          title={t('cart.emptyTitle')}
          body={t('cart.emptyBody')}
          action={
            <Link href={STORE_HOME} className="wsh-go">
              {t('cart.browse')}
            </Link>
          }
        />
      </Card>
    );
  }
  const canContinue = cart.items_count > 0 && cart.unavailable_items.length === 0 && !cart.discount?.error_code;
  return (
    <div className="wsc-cols">
      <div className="wsc-main">
        {cart.groups.map((g) => (
          <section key={g.seller.id} className="wsc-group" aria-label={g.seller.name}>
            <Card className="wsc-card">
              <div className="wsc-seller">
                <h2 className="wsc-seller-name">
                  <Icon name="bag" size={16} />
                  {g.seller.name}
                </h2>
                <span className="wsc-seller-meta">
                  {t('cart.shipDays', { min: formatNumber(g.seller.ship_min_days, locale), max: formatNumber(g.seller.ship_max_days, locale) })}
                  {g.items.some((l) => l.discreet_packaging) ? ` · ${t('cart.plain')}` : ''}
                </span>
              </div>
              <FreeShip group={g} />
              <ul className="wsc-list">
                {g.items.map((l) => (
                  <Line key={l.variant_id} line={l} />
                ))}
              </ul>
            </Card>
          </section>
        ))}
        {cart.unavailable_items.length ? (
          <section className="wsc-group" aria-label={t('cart.unavailableTitle')}>
            <Card className="wsc-card">
              <h2 className="wsc-seller-name">{t('cart.unavailableTitle')}</h2>
              <p className="wsh-hint">{t('cart.unavailableBody')}</p>
              <ul className="wsc-list">
                {cart.unavailable_items.map((l) => (
                  <Line key={l.variant_id} line={l} />
                ))}
              </ul>
            </Card>
          </section>
        ) : null}
        <DiscountBox code={cart.discount?.code ?? null} error={cart.discount?.error_code ?? null} />
      </div>
      <CartSummary
        footer={
          canContinue ? (
            <Link href="/store/checkout" className="wsh-go wsc-cta">
              {t('cart.continue')}
            </Link>
          ) : (
            <button type="button" className="wsh-go wsc-cta" disabled>
              {t('cart.continue')}
            </button>
          )
        }
      >
        <div className="wsc-row">
          <dt>{w('cart.itemsCount', { n: formatNumber(cart.items_count, locale) })}</dt>
          <dd>{price(cart.items_total)}</dd>
        </div>
        {cart.discount_total > 0 ? (
          <div className="wsc-row">
            <dt>{t('cart.discount')}</dt>
            <dd>− {price(cart.discount_total)}</dd>
          </div>
        ) : null}
        <div className="wsc-row">
          <dt>{t('cart.shipping')}</dt>
          <dd className="wsc-muted">{t('cart.shippingNext')}</dd>
        </div>
        <div className="wsc-row is-total">
          <dt>{t('cart.payable')}</dt>
          <dd>{price(cart.payable_total)}</dd>
        </div>
      </CartSummary>
    </div>
  );
}

/** `/store/cart` — «سبد خرید» on desktop (CB-SHOP-11, W_Shop_Cart): lines per seller, free-shipping bar, discount, summary. */
export function WebShopCart() {
  const t = useTranslations('shop');
  return (
    <WebShopGate appHref={`${APP_SHOP}/cart`}>
      <h1 className="wsh-h1">{t('cart.title')}</h1>
      <CartBody />
    </WebShopGate>
  );
}
