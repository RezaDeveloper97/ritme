'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import {
  shopErrorKey,
  useAddToCart,
  useShopCatalogProduct,
  useShopCatalogReviews,
  type ShopCatalogProduct,
  type ShopCatalogReview,
  type ShopCatalogVariant,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatMonthLabel, formatNumber, fromApiDate, toParts } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { CartButton } from './CartButton';
import { ProductTile, usePrice, useRatingText, useSkinLabel } from './ProductTile';

type Dim = 'size' | 'color' | 'volume';
type Selection = Record<Dim, string | null>;
const DIMS: Dim[] = ['color', 'size', 'volume'];

/** Attribute rows shown as «label — value»; the size guide and ingredients get their own spot. */
const ATTR_ROWS = ['material', 'wash'];
const ATTR_OWN_SPOT = ['size_guide', 'ingredients'];

const selectionOf = (v: ShopCatalogVariant | undefined): Selection => ({
  size: v?.size ?? null,
  color: v?.color ?? null,
  volume: v?.volume ?? null,
});

function Stars({ value }: { value: number }) {
  return (
    <span className="shp-stars" aria-hidden>
      {[1, 2, 3, 4, 5].map((n) => (
        <Icon key={n} name="star" size={12} className={n <= Math.round(value) ? 'is-on' : 'is-off'} />
      ))}
    </span>
  );
}

function ReviewItem({ r }: { r: ShopCatalogReview }) {
  const t = useTranslations('shop');
  return (
    <li className="shp-review">
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
  );
}

function Reviews({ product }: { product: ShopCatalogProduct }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const [all, setAll] = useState(false);
  const more = useShopCatalogReviews(product.id, all);
  const items = all && more.data ? more.data.pages.flatMap((p) => p.reviews) : product.reviews.items;
  if (product.reviews.rating_count === 0 && items.length === 0) return null;
  return (
    <section className="shp-sec" aria-labelledby="shp-reviews">
      <div className="shp-sec-head">
        <h2 id="shp-reviews" className="shp-sec-title">
          {t('product.reviews')}
        </h2>
        {!all && product.reviews.rating_count > items.length ? (
          <button type="button" className="shp-sec-all" onClick={() => setAll(true)}>
            {t('product.allReviews', { n: formatNumber(product.reviews.rating_count, locale) })}
          </button>
        ) : null}
      </div>
      <ul className="shp-reviews">
        {items.map((r) => (
          <ReviewItem key={r.id} r={r} />
        ))}
      </ul>
      {all && more.isPending ? (
        <SkeletonGroup label={t('loading')}>
          <Skeleton shape="block" />
        </SkeletonGroup>
      ) : null}
      {all && more.hasNextPage ? (
        <SecondaryButton variant="text" loading={more.isFetchingNextPage} onClick={() => void more.fetchNextPage()}>
          {t('product.moreReviews')}
        </SecondaryButton>
      ) : null}
    </section>
  );
}

function ProductBody({ product }: { product: ShopCatalogProduct }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const addCart = useAddToCart();
  const price = usePrice();
  const ratingText = useRatingText();
  const skinLabel = useSkinLabel();
  const initial = product.variants.find((v) => v.id === product.default_variant_id) ?? product.variants[0];
  const [sel, setSel] = useState<Selection>(selectionOf(initial));
  const [shot, setShot] = useState(0);

  const options = useMemo(() => {
    const out = {} as Record<Dim, string[]>;
    for (const d of DIMS) out[d] = [...new Set(product.variants.map((v) => v[d]).filter((x): x is string => !!x))];
    return out;
  }, [product.variants]);

  // The variant matching every chosen dimension (a dimension the product doesn't vary stays null on both sides).
  const variant = product.variants.find((v) => DIMS.every((d) => v[d] === sel[d]));

  const pick = (dim: Dim, value: string) => {
    const cand = product.variants.filter((v) => v[dim] === value);
    // Keep the other choices when that combination exists, else take the first in-stock variant with this value.
    const best =
      cand.find((v) => DIMS.every((d) => d === dim || v[d] === sel[d])) ?? cand.find((v) => v.in_stock) ?? cand[0];
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
  const days = s ? (s.ship_min_days === s.ship_max_days ? s.ship_min_days : null) : null;

  return (
    <>
      <div className="shp-gallery">
        <div className="shp-hero-art">
          {images[shot] ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={images[shot]} alt={product.title} />
          ) : (
            <Icon name="bag" size={56} strokeWidth={1.2} />
          )}
          {product.badges[0] ? <span className="shp-tile-badge">{product.badges[0]}</span> : null}
        </div>
        {images.length > 1 ? (
          <ul className="shp-thumbs" aria-label={t('product.gallery')}>
            {images.map((src, i) => (
              <li key={src}>
                <button
                  type="button"
                  className="shp-thumb-btn"
                  aria-pressed={i === shot}
                  aria-label={t('product.photo', { n: formatNumber(i + 1, locale) })}
                  onClick={() => setShot(i)}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS */}
                  <img src={src} alt="" loading="lazy" />
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>

      <div className="shp-pinfo">
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
        <h2 className="shp-ptitle">{product.title}</h2>
        {product.subtitle ? <p className="shp-psub">{product.subtitle}</p> : null}
        <p className="shp-prating">
          {product.rating_avg !== null && product.rating_count > 0 ? (
            <>
              <Icon name="star" size={14} />
              <bdi>{ratingText(product.rating_avg)}</bdi>
              <span className="shp-tile-count">{t('product.ratingCount', { n: formatNumber(product.rating_count, locale) })}</span>
            </>
          ) : (
            <span className="shp-tile-count">{t('product.noReviews')}</span>
          )}
          {product.purchase_count > 0 ? (
            <span className="shp-tile-count">· {t('product.purchases', { n: formatNumber(product.purchase_count, locale) })}</span>
          ) : null}
        </p>
        {isBeauty && product.skin_types.length ? (
          <p className="shp-skin">{t('product.suits', { types: product.skin_types.map((k) => skinLabel(k, k)).join(t('product.sep')) })}</p>
        ) : null}
        <p className="shp-pprice">
          <span className="shp-price is-big">{price(shownPrice)}</span>
          {shownWas && shownWas > shownPrice ? <s className="shp-was">{price(shownWas)}</s> : null}
        </p>
      </div>

      {DIMS.map((dim) =>
        options[dim].length ? (
          <section key={dim} className="shp-opt">
            <h3 className="shp-opt-title">
              {t(`product.dim.${dim}`)}
              {dim === 'color' && sel.color ? <span className="shp-opt-val">: {sel.color}</span> : null}
            </h3>
            <ChipGroup label={t(`product.dim.${dim}`)}>
              {options[dim].map((v) => (
                <PillChip
                  key={v}
                  pressed={sel[dim] === v}
                  disabled={!available(dim, v)}
                  onPressedChange={() => pick(dim, v)}
                  aria-label={available(dim, v) ? undefined : t('product.optionOut', { name: v })}
                >
                  {v}
                </PillChip>
              ))}
            </ChipGroup>
            {dim === 'size' && attrs.size_guide ? <p className="shp-guide">{t('product.sizeGuide', { text: attrs.size_guide })}</p> : null}
          </section>
        ) : null,
      )}

      <Card className="shp-facts">
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
          {s ? (
            <>
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
            </>
          ) : null}
        </dl>
      </Card>

      {isBeauty ? <InfoNote icon="info">{t('product.caution')}</InfoNote> : null}

      <Reviews product={product} />

      {product.bought_together.length ? (
        <section className="shp-sec" aria-labelledby="shp-bt">
          <h2 id="shp-bt" className="shp-sec-title">
            {t('product.boughtTogether')}
          </h2>
          <ul className="shp-tiles shp-rail">
            {product.bought_together.map((p) => (
              <ProductTile key={p.id} p={p} />
            ))}
          </ul>
        </section>
      ) : null}

      <p className="shp-note">{t('home.cod')}</p>

      {/* Sticky add-to-cart bar (CB-SHOP-08): POST /shop/cart/items with the chosen variant; then «دیدن سبد». */}
      <div className="shp-buybar">
        <div className="shp-buy-info">
          <span className="shp-buy-variant">{[shown?.size, shown?.color, shown?.volume].filter(Boolean).join(' · ') || product.title}</span>
          <span className="shp-price">{price(shownPrice)}</span>
          <span id="shp-buy-hint" className="shp-buy-hint" role="status">
            {addCart.isError
              ? t(`errors.${shopErrorKey(addCart.error)}`)
              : addCart.isSuccess
                ? t('product.addedToCart')
                : inStock
                  ? t('product.codHint')
                  : t('product.outHint')}
          </span>
        </div>
        {addCart.isSuccess ? (
          <PrimaryButton onClick={() => router.push('/shop/cart')}>{t('product.viewCart')}</PrimaryButton>
        ) : (
          <PrimaryButton
            disabled={!inStock || !shown}
            loading={addCart.isPending}
            aria-describedby="shp-buy-hint"
            onClick={() => shown && addCart.mutate({ variantId: shown.id })}
          >
            {inStock ? t('product.addToCart') : t('home.outOfStock')}
          </PrimaryButton>
        )}
      </div>
    </>
  );
}

/**
 * `/shop/product/[id]` — product page of the internal catalog (CB-SHOP-07, nbd_Shop_Product + nbd_Shop_ProductBeauty).
 * Baby products: gallery, seller, rating, color/size choosers with the size guide, material/wash/shipping/return,
 * reviews, bought-together. Beauty products add volume, skin types, IRC + expiry, ingredients and the pregnancy /
 * breastfeeding caution. Pay on delivery only. Teen mode (403) and unknown products (404) get a plain state.
 */
export function ShopProductPage({ id }: { id: number }) {
  const t = useTranslations('shop');
  const router = useRouter();
  const q = useShopCatalogProduct(id);
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
        <EmptyState icon="shield" title={t('home.teenTitle')} body={t('home.teenBody')} />
      </Card>
    );
  } else if (status === 404) {
    body = (
      <Card>
        <EmptyState icon="bag" title={t('product.notFoundTitle')} body={t('product.notFoundBody')} />
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
    // `key` resets the chosen variant when the user moves to another product from «bought together».
    body = <ProductBody key={q.data.id} product={q.data} />;
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('product.title')}
          onBack={() => router.push(q.data ? `/shop/${q.data.category}` : '/shop')}
          backLabel={t('back')}
          action={<CartButton />}
        />
        <div className="shp-body shp-pbody">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
