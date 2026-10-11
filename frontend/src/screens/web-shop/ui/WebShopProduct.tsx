'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import {
  shopErrorKey,
  type ShopCatalogProduct,
  type ShopCatalogVariant,
  useAddToCart,
  useShopCatalogProduct,
  useShopPrice,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale } from '@/shared/i18n';
import { formatMonthLabel, formatNumber, fromApiDate, toParts } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { isAuthenticated } from '@/shared/session';
import { Card, EmptyState, Icon, InfoNote, Skeleton, SkeletonGroup } from '@/shared/ui';

import { APP_SHOP, STORE_HOME, storeCategoryHref } from '../model/store';
import { ShopWebShell, useRating, WebTile } from './ShopWebShell';

type Dim = 'size' | 'color' | 'volume';
type Selection = Record<Dim, string | null>;
const DIMS: Dim[] = ['color', 'size', 'volume'];
const ATTR_ROWS = ['material', 'wash'];
const ATTR_OWN_SPOT = ['size_guide', 'ingredients'];
const SKIN_KEYS = ['oily', 'combination', 'normal', 'dry', 'sensitive'];

const selectionOf = (v: ShopCatalogVariant | undefined): Selection => ({ size: v?.size ?? null, color: v?.color ?? null, volume: v?.volume ?? null });

function Stars({ value }: { value: number }) {
  return (
    <span className="shp-stars" aria-hidden>
      {[1, 2, 3, 4, 5].map((n) => (
        <Icon key={n} name="star" size={12} className={n <= Math.round(value) ? 'is-on' : 'is-off'} />
      ))}
    </span>
  );
}

function ProductBody({ product }: { product: ShopCatalogProduct }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const locale = useLocale() as Locale;
  const price = useShopPrice();
  const rating = useRating();
  const mounted = useMounted();
  const authed = mounted && isAuthenticated();
  const addCart = useAddToCart();

  const initial = product.variants.find((v) => v.id === product.default_variant_id) ?? product.variants[0];
  const [sel, setSel] = useState<Selection>(selectionOf(initial));
  const [shot, setShot] = useState(0);
  const options = useMemo(() => {
    const out = {} as Record<Dim, string[]>;
    for (const d of DIMS) out[d] = [...new Set(product.variants.map((v) => v[d]).filter((x): x is string => !!x))];
    return out;
  }, [product.variants]);
  const variant = product.variants.find((v) => DIMS.every((d) => v[d] === sel[d]));
  const pick = (dim: Dim, value: string) => {
    const cand = product.variants.filter((v) => v[dim] === value);
    const best = cand.find((v) => DIMS.every((d) => d === dim || v[d] === sel[d])) ?? cand.find((v) => v.in_stock) ?? cand[0];
    setSel(selectionOf(best));
  };
  const available = (dim: Dim, value: string) => product.variants.some((v) => v[dim] === value && v.in_stock);

  const isBeauty = !!product.irc_code || product.skin_types.length > 0;
  const shown = variant ?? initial;
  const shownPrice = shown?.price ?? product.price;
  const shownWas = shown?.compare_at_price ?? product.compare_at_price;
  const inStock = shown ? shown.in_stock : product.in_stock;
  const images = product.images.length ? product.images : product.image_url ? [product.image_url] : [];
  const s = product.seller;
  const attrs = product.attributes;
  const rows = [...ATTR_ROWS.filter((k) => attrs[k]), ...Object.keys(attrs).filter((k) => !ATTR_ROWS.includes(k) && !ATTR_OWN_SPOT.includes(k))];
  const attrLabel = (k: string) => (k === 'material' || k === 'wash' ? t(`product.attr.${k}`) : k.replace(/_/g, ' '));
  const expiry = product.expires_on
    ? (() => {
        const d = toParts(fromApiDate(product.expires_on), locale);
        return formatMonthLabel(d.year, d.month, locale);
      })()
    : null;
  const days = s && s.ship_min_days === s.ship_max_days ? s.ship_min_days : null;
  const skin = (k: string) => (SKIN_KEYS.includes(k) ? t(`skin.${k as 'oily'}`) : k);

  let buy;
  if (!inStock || !shown) {
    buy = <p className="wsh-out">{inStock ? t('product.outHint') : t('home.outOfStock')}</p>;
  } else if (!authed) {
    // Cart is a signed-in feature: logged out → a login link (never a silent no-op).
    buy = (
      <div className="wsh-login">
        <strong>{w('product.loginTitle')}</strong>
        <p>{w('product.loginBody')}</p>
        <Link href="/signup" className="wsh-go">
          {w('product.login')}
        </Link>
      </div>
    );
  } else if (addCart.isSuccess) {
    buy = (
      <Link href={`${APP_SHOP}/cart`} className="wsh-go">
        {w('product.openCart')}
        <Icon name="arrowR" size={16} />
      </Link>
    );
  } else {
    buy = (
      <button type="button" className="wsh-go" disabled={addCart.isPending} aria-describedby="wsh-buy-hint" onClick={() => addCart.mutate({ variantId: shown.id })}>
        <Icon name="bag" size={18} />
        {t('product.addToCart')}
      </button>
    );
  }
  const hint = addCart.isError
    ? t(`errors.${shopErrorKey(addCart.error)}`)
    : addCart.isSuccess
      ? t('product.addedToCart')
      : inStock && shown
        ? t('product.codHint')
        : null;

  return (
    <article className="wsh-article">
      <nav aria-label={w('product.crumbLabel')} className="wsh-crumbs">
        <Link href={STORE_HOME}>{w('product.crumbRoot')}</Link>
        <span aria-hidden>/</span>
        <Link href={storeCategoryHref(product.category)}>{w('product.crumbCategory')}</Link>
        <span aria-hidden>/</span>
        <span aria-current="page">{product.title}</span>
      </nav>

      <div className="wsh-pcols">
        <section aria-label={t('product.gallery')} className="wsh-gallery">
          <div className="wsh-hero-img">
            {images[shot] ? (
              // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
              <img src={images[shot]} alt={product.title} />
            ) : (
              <Icon name="bag" size={72} strokeWidth={1.2} />
            )}
            {product.badges[0] ? <span className="shp-tile-badge">{product.badges[0]}</span> : null}
          </div>
          {images.length > 1 ? (
            <ul className="wsh-thumbs">
              {images.map((src, i) => (
                <li key={src}>
                  <button
                    type="button"
                    className={clsx('wsh-thumb', i === shot && 'on')}
                    aria-pressed={i === shot}
                    aria-label={t('product.photo', { n: formatNumber(i + 1, locale) })}
                    onClick={() => setShot(i)}
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image */}
                    <img src={src} alt="" loading="lazy" />
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </section>

        <div className="wsh-info">
          {s ? (
            <p className="shp-pseller">
              {s.name}
              {s.is_verified ? (
                <span className="shp-verified">
                  <Icon name="shield" size={13} />
                  {t('product.verifiedSeller')}
                </span>
              ) : null}
            </p>
          ) : null}
          <h1 className="wsh-h1 wsh-ptitle">{product.title}</h1>
          {product.subtitle ? <p className="wsh-lead">{product.subtitle}</p> : null}
          <p className="shp-prating">
            {product.rating_avg !== null && product.rating_count > 0 ? (
              <>
                <Icon name="star" size={14} />
                <bdi>{rating(product.rating_avg)}</bdi>
                <span className="shp-tile-count">{t('product.ratingCount', { n: formatNumber(product.rating_count, locale) })}</span>
              </>
            ) : (
              <span className="shp-tile-count">{t('product.noReviews')}</span>
            )}
            {product.purchase_count > 0 ? <span className="shp-tile-count">· {t('product.purchases', { n: formatNumber(product.purchase_count, locale) })}</span> : null}
          </p>
          {isBeauty && product.skin_types.length > 0 ? <p className="shp-skin">{t('product.suits', { types: product.skin_types.map(skin).join(t('product.sep')) })}</p> : null}
          <p className="wsh-price">
            <span className="shp-price is-big">{price(shownPrice)}</span>
            {shownWas && shownWas > shownPrice ? <s className="shp-was">{price(shownWas)}</s> : null}
          </p>

          {DIMS.map((dim) =>
            options[dim].length > 0 ? (
              <div key={dim} className="wsh-opt" role="group" aria-label={t(`product.dim.${dim}`)}>
                <h2 className="wsh-opt-title">
                  {t(`product.dim.${dim}`)}
                  {dim === 'color' && sel.color ? <span className="shp-opt-val">: {sel.color}</span> : null}
                </h2>
                <div className="wsh-opt-row">
                  {options[dim].map((v) => (
                    <button
                      key={v}
                      type="button"
                      className={clsx('wsh-pill', sel[dim] === v && 'on')}
                      aria-pressed={sel[dim] === v}
                      disabled={!available(dim, v)}
                      aria-label={available(dim, v) ? undefined : t('product.optionOut', { name: v })}
                      onClick={() => pick(dim, v)}
                    >
                      {v}
                    </button>
                  ))}
                </div>
                {dim === 'size' && attrs.size_guide ? <p className="wsh-hint">{t('product.sizeGuide', { text: attrs.size_guide })}</p> : null}
              </div>
            ) : null,
          )}

          <div className="wsh-buy">
            {buy}
            {hint ? (
              <p id="wsh-buy-hint" className="wsh-hint" role="status">
                {hint}
              </p>
            ) : null}
          </div>

          {s ? (
            <Card className="wsh-facts">
              <dl className="shp-dl">
                <div className="shp-dl-row">
                  <dt>{t('product.shipping')}</dt>
                  <dd>
                    {days !== null
                      ? t('product.shipDays', { n: formatNumber(days, locale) })
                      : t('product.shipRange', { min: formatNumber(s.ship_min_days, locale), max: formatNumber(s.ship_max_days, locale) })}
                    {s.discreet_packaging || product.discreet_packaging ? ` · ${t('product.discreet')}` : ` · ${t('product.fromSeller')}`}
                    <span className="shp-dl-sub">
                      {s.shipping_fee === 0
                        ? t('product.shipFree')
                        : s.free_shipping_over
                          ? t('product.shipFeeOver', { fee: price(s.shipping_fee), over: price(s.free_shipping_over) })
                          : t('product.shipFee', { fee: price(s.shipping_fee) })}
                    </span>
                  </dd>
                </div>
                {s.return_days > 0 ? (
                  <div className="shp-dl-row">
                    <dt>{t('product.returns')}</dt>
                    <dd>{t(isBeauty ? 'product.returnSealed' : 'product.returnOpen', { n: formatNumber(s.return_days, locale) })}</dd>
                  </div>
                ) : null}
              </dl>
            </Card>
          ) : null}
          {isBeauty ? <InfoNote icon="info">{t('product.caution')}</InfoNote> : null}
        </div>
      </div>

      {rows.length > 0 || (isBeauty && (product.irc_code || attrs.ingredients)) || product.description ? (
        <section className="wsh-block" aria-labelledby="wsh-specs">
          <h2 id="wsh-specs" className="wsh-h2">
            {w('product.specs')}
          </h2>
          {product.description ? <p className="wsh-desc">{product.description}</p> : null}
          <Card>
            <dl className="shp-dl">
              {isBeauty && product.irc_code ? (
                <div className="shp-dl-row">
                  <dt>{t('product.irc')}</dt>
                  <dd>
                    <bdi>{product.irc_code}</bdi>
                    {expiry ? ` · ${t('product.expires', { date: expiry })}` : null}
                  </dd>
                </div>
              ) : null}
              {isBeauty && attrs.ingredients ? (
                <div className="shp-dl-row">
                  <dt>{t('product.ingredients')}</dt>
                  <dd>{attrs.ingredients}</dd>
                </div>
              ) : null}
              {rows.map((k) => (
                <div key={k} className="shp-dl-row">
                  <dt>{attrLabel(k)}</dt>
                  <dd>{attrs[k]}</dd>
                </div>
              ))}
            </dl>
          </Card>
        </section>
      ) : null}

      {product.reviews.items.length > 0 ? (
        <section className="wsh-block" aria-labelledby="wsh-reviews">
          <h2 id="wsh-reviews" className="wsh-h2">
            {t('product.reviews')}
          </h2>
          <ul className="wsh-reviews">
            {product.reviews.items.slice(0, 6).map((r) => (
              <li key={r.id} className="shp-review">
                <div className="shp-review-head">
                  <span className="shp-review-author">{r.author_name}</span>
                  <Stars value={r.rating} />
                </div>
                <p className="shp-review-meta">
                  {r.verified_purchase ? t('product.verified') : null}
                  {r.verified_purchase && r.variant_label ? ' · ' : null}
                  {r.variant_label}
                </p>
                {r.body ? <p className="shp-review-body">{r.body}</p> : null}
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {product.bought_together.length > 0 ? (
        <section className="wsh-block" aria-labelledby="wsh-bt">
          <h2 id="wsh-bt" className="wsh-h2">
            {t('product.boughtTogether')}
          </h2>
          <ul className="wsh-tiles">
            {product.bought_together.slice(0, 5).map((p) => (
              <WebTile key={p.id} p={p} />
            ))}
          </ul>
        </section>
      ) : null}
    </article>
  );
}

/**
 * CB-SHOP-10 — `W_Shop_Product`: breadcrumb, gallery, info + variant choosers + add-to-cart, specifications, reviews,
 * bought-together. Public and server-rendered (`initial`). Logged out → a login link; logged in → the shared cart hook,
 * then a link into the app cart. Pay on delivery only.
 */
export function WebShopProduct({ id, initial }: { id: number; initial?: ShopCatalogProduct | null }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const q = useShopCatalogProduct(id, { anon: true, initialData: initial ?? undefined });
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="wsh-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon={status === 403 ? 'shield' : status === 404 ? 'bag' : 'warning'}
          title={status === 403 ? t('home.teenTitle') : status === 404 ? t('product.notFoundTitle') : t('loadError')}
          body={status === 403 ? w('product.teen') : status === 404 ? t('product.notFoundBody') : undefined}
          action={
            <Link href={STORE_HOME} className="wsh-link-btn">
              {w('product.crumbRoot')}
            </Link>
          }
        />
      </Card>
    );
  } else {
    body = <ProductBody key={q.data.id} product={q.data} />;
  }

  return (
    <ShopWebShell appHref={`${APP_SHOP}/product/${id}`}>
      <div className="wsh">
        {body}
        <p className="shp-note wsh-note">{t('home.cod')}</p>
      </div>
    </ShopWebShell>
  );
}
