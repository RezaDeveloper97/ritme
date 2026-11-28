'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
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
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/shp.css';
import '@/shared/ui/styles/shcart.css';

const MAX_QTY = 10;

function Line({ line, price }: { line: ShopCartLine; price: (n: number) => string }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const setQty = useSetCartQuantity();
  const remove = useRemoveCartItem();
  const busy = setQty.isPending || remove.isPending;
  const failed = setQty.isError ? shopErrorKey(setQty.error) : null;
  const out = line.status !== 'ok';
  return (
    <li className={`shcart-line${out ? ' is-out' : ''}`}>
      <Link href={`/shop/product/${line.product_id}`} className="shcart-art" aria-hidden tabIndex={-1}>
        {line.image_url ? (
          // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
          <img src={line.image_url} alt="" loading="lazy" />
        ) : (
          <Icon name="bag" size={24} strokeWidth={1.4} />
        )}
      </Link>
      <div className="shcart-main">
        <Link href={`/shop/product/${line.product_id}`} className="shcart-title" {...pilot(`shop-cart.product:${line.variant_id}`)}>
          {line.title}
        </Link>
        {line.variant_label ? <span className="shcart-variant">{line.variant_label}</span> : null}
        {out ? <span className="shcart-out">{t('cart.outOfStock')}</span> : <span className="shp-price">{price(line.line_total)}</span>}
        {failed ? (
          <span className="shcart-error" role="alert">
            {t(`errors.${failed}`)}
          </span>
        ) : null}
      </div>
      <div className="shcart-qty" role="group" aria-label={t('cart.qty', { title: line.title })}>
        <button
          type="button"
          className="shcart-step"
          disabled={busy}
          aria-label={line.quantity <= 1 ? t('cart.remove', { title: line.title }) : t('cart.dec')}
          {...(line.quantity > 1 ? pilot(`shop-cart.dec:${line.variant_id}`) : {})}
          onClick={() => (line.quantity <= 1 ? remove.mutate(line.variant_id) : setQty.mutate({ variantId: line.variant_id, quantity: line.quantity - 1 }))}
        >
          <Icon name={line.quantity <= 1 ? 'trash' : 'minus'} size={16} />
        </button>
        <span className="shcart-n" aria-live="polite">
          {formatNumber(line.quantity, locale)}
        </span>
        <button
          type="button"
          className="shcart-step"
          disabled={busy || line.quantity >= MAX_QTY || out}
          aria-label={t('cart.inc')}
          {...pilot(`shop-cart.inc:${line.variant_id}`)}
          onClick={() => setQty.mutate({ variantId: line.variant_id, quantity: line.quantity + 1 })}
        >
          <Icon name="plus" size={16} />
        </button>
      </div>
    </li>
  );
}

function DiscountBox({ code, error }: { code: string | null; error: string | null }) {
  const t = useTranslations('shop');
  const [value, setValue] = useState('');
  const apply = useApplyDiscount();
  const remove = useRemoveDiscount();
  const failed = apply.isError ? shopErrorKey(apply.error) : null;
  if (code) {
    return (
      <Card className="shcart-disc">
        <p className="shcart-disc-on">{t('cart.discountApplied', { code })}</p>
        {error ? (
          <p className="shcart-error" role="alert">
            {t(`errors.${shopCodeKey(error)}`)}
          </p>
        ) : null}
        <SecondaryButton variant="text" loading={remove.isPending} onClick={() => remove.mutate()} {...pilot('shop-cart.discount-remove')}>
          {t('cart.discountRemove')}
        </SecondaryButton>
      </Card>
    );
  }
  return (
    <Card className="shcart-disc">
      <label className="lbl" htmlFor="shcart-code">
        {t('cart.discountPrompt')}
      </label>
      <div className="shcart-disc-row">
        <div className="field">
          <input
            id="shcart-code"
            dir="ltr"
            value={value}
            maxLength={32}
            autoComplete="off"
            placeholder={t('cart.discountPlaceholder')}
            onChange={(e) => setValue(e.target.value)}
            {...pilot('shop-cart.discount-code', 'text')}
          />
        </div>
        <SecondaryButton loading={apply.isPending} disabled={!value.trim()} {...pilot('shop-cart.discount-apply')} onClick={() => apply.mutate(value.trim(), { onSuccess: () => setValue('') })}>
          {t('cart.discountApply')}
        </SecondaryButton>
      </div>
      {failed ? (
        <p className="shcart-error" role="alert">
          {t(`errors.${failed}`)}
        </p>
      ) : null}
    </Card>
  );
}

/**
 * `/shop/cart` — «سبد خرید» (CB-SHOP-08, nbd_Shop_Cart): lines grouped per seller (one shipment each), quantity,
 * free-shipping hint, discount code, totals and «ادامه خرید». Shipping is priced at checkout. COD only.
 */
export function ShopCartPage() {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const price = useShopPrice();
  const q = useShopCart();
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  let body;
  let footer = null;
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
        <EmptyState icon="shield" title={t('cart.teenTitle')} body={t('cart.teenBody')} />
      </Card>
    );
  } else if (q.isError) {
    body = (
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
  } else {
    const cart = q.data;
    const empty = cart.groups.length === 0 && cart.unavailable_items.length === 0;
    if (empty) {
      body = (
        <Card>
          <EmptyState
            icon="bag"
            title={t('cart.emptyTitle')}
            body={t('cart.emptyBody')}
            action={<PrimaryButton onClick={() => router.push('/shop')}>{t('cart.browse')}</PrimaryButton>}
          />
        </Card>
      );
    } else {
      const canContinue = cart.items_count > 0 && cart.unavailable_items.length === 0 && !cart.discount?.error_code;
      body = (
        <>
          <p className="shcart-sum">
            {t('cart.summary', { items: formatNumber(cart.items_count, locale), sellers: formatNumber(cart.sellers_count, locale) })}
          </p>
          {cart.groups.map((g) => (
            <section key={g.seller.id} className="shcart-group" aria-label={g.seller.name}>
              <div className="shcart-seller">
                <h2 className="shcart-seller-name">{g.seller.name}</h2>
                <span className="shcart-seller-meta">
                  {t('cart.shipDays', { min: formatNumber(g.seller.ship_min_days, locale), max: formatNumber(g.seller.ship_max_days, locale) })}
                  {g.items.some((l) => l.discreet_packaging) ? ` · ${t('cart.plain')}` : ''}
                </span>
              </div>
              {g.free_shipping_remaining ? (
                <p className="shcart-free">{t('cart.freeShip', { amount: price(g.free_shipping_remaining) })}</p>
              ) : g.seller.free_shipping_over && g.shipping_fee === 0 ? (
                <p className="shcart-free is-done">{t('cart.freeShipDone')}</p>
              ) : null}
              <Card className="shcart-lines">
                <ul className="shcart-list">
                  {g.items.map((l) => (
                    <Line key={l.variant_id} line={l} price={price} />
                  ))}
                </ul>
              </Card>
            </section>
          ))}
          {cart.unavailable_items.length ? (
            <section className="shcart-group" aria-label={t('cart.unavailableTitle')}>
              <h2 className="shcart-seller-name">{t('cart.unavailableTitle')}</h2>
              <p className="shcart-free">{t('cart.unavailableBody')}</p>
              <Card className="shcart-lines">
                <ul className="shcart-list">
                  {cart.unavailable_items.map((l) => (
                    <Line key={l.variant_id} line={l} price={price} />
                  ))}
                </ul>
              </Card>
            </section>
          ) : null}
          <DiscountBox code={cart.discount?.code ?? null} error={cart.discount?.error_code ?? null} />
          <Card className="shcart-totals">
            <dl>
              <div className="shcart-row">
                <dt>{t('cart.items')}</dt>
                <dd>{price(cart.items_total)}</dd>
              </div>
              {cart.discount_total > 0 ? (
                <div className="shcart-row">
                  <dt>{t('cart.discount')}</dt>
                  <dd>− {price(cart.discount_total)}</dd>
                </div>
              ) : null}
              <div className="shcart-row">
                <dt>{t('cart.shipping')}</dt>
                <dd className="shcart-muted">{t('cart.shippingNext')}</dd>
              </div>
              <div className="shcart-row is-total">
                <dt>{t('cart.payable')}</dt>
                <dd>{price(cart.payable_total)}</dd>
              </div>
            </dl>
          </Card>
          <p className="shp-note">{t('cart.separate')}</p>
          <p className="shp-note">{t('home.cod')}</p>
        </>
      );
      footer = (
        <div className="shp-buybar">
          <div className="shp-buy-info">
            <span className="shp-buy-variant">{t('cart.total', { amount: price(cart.payable_total) })}</span>
          </div>
          <PrimaryButton disabled={!canContinue} onClick={() => router.push('/shop/checkout')} {...pilot('shop-cart.continue')}>
            {t('cart.continue')}
          </PrimaryButton>
        </div>
      );
    }
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('cart.title')} onBack={() => router.back('/shop')} backLabel={t('cart.back')} />
        <div className="shp-body shcart-body">
          {body}
          {footer}
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
