'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  checkoutBlockerKey,
  maskPhone,
  type ShopCheckout,
  shopErrorKey,
  useIdempotencyKey,
  usePlaceOrder,
  useShopCheckout,
  useShopPrice,
  useSlotText,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import { AddressSheet } from './AddressSheet';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/shp.css';
import '@/shared/ui/styles/shco.css';
import '@/shared/ui/styles/shcart.css';

function Stepper() {
  const t = useTranslations('shop.checkout');
  const steps = ['stepCart', 'stepShip', 'stepConfirm'] as const;
  return (
    <ol className="shco-steps" aria-label={t('steps')}>
      {steps.map((s, i) => (
        <li key={s} className={`shco-step${i === 1 ? ' is-current' : i === 0 ? ' is-done' : ''}`} aria-current={i === 1 ? 'step' : undefined}>
          <span className="shco-step-dot" aria-hidden>
            {i === 0 ? <Icon name="check" size={12} strokeWidth={3} /> : null}
          </span>
          {t(s)}
        </li>
      ))}
    </ol>
  );
}

function Review({ data }: { data: ShopCheckout }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const price = useShopPrice();
  const slotText = useSlotText();
  const place = usePlaceOrder();
  const nextKey = useIdempotencyKey();
  const [addressId, setAddressId] = useState<number | null>(data.default_address_id ?? data.addresses[0]?.id ?? null);
  const [sheet, setSheet] = useState(false);
  const [discreet, setDiscreet] = useState(false);
  const [slots, setSlots] = useState<Record<number, number>>({});

  const address = data.addresses.find((a) => a.id === addressId) ?? null;
  const blockers = data.blockers.filter((b) => b !== 'address_invalid' || !address);
  const canPlace = data.can_place && address !== null && !place.isPending;
  const codBlockers = data.shipments.filter((s) => !s.cod_available);

  const submit = () => {
    if (!address) return;
    place.mutate(
      {
        idempotencyKey: nextKey(),
        addressId: address.id,
        discreet: data.discreet_packaging_available && discreet,
        slots: Object.entries(slots).map(([seller, slot]) => ({ seller_id: Number(seller), slot_id: slot })),
      },
      { onSuccess: (order) => router.replace(`/shop/order/${order.id}?placed=1`) },
    );
  };

  return (
    <>
      <Stepper />

      <section className="shco-sec" aria-labelledby="shco-addr">
        <div className="shp-sec-head">
          <h2 id="shco-addr" className="shp-sec-title">
            {t('checkout.address')}
          </h2>
          <button type="button" className="shp-sec-all" onClick={() => setSheet(true)}>
            {address ? t('checkout.change') : t('checkout.add')}
          </button>
        </div>
        <Card className="shco-card">
          {address ? (
            <>
              <p className="shco-line-main">{address.label || address.city}</p>
              <p className="shco-line-sub">
                {address.address_line} · {address.city}
              </p>
            </>
          ) : (
            <p className="shco-line-sub">{t('checkout.addFirst')}</p>
          )}
        </Card>
        {address ? (
          <>
            <h2 className="shp-sec-title">{t('checkout.recipient')}</h2>
            <Card className="shco-card">
              <p className="shco-line-main">
                {address.recipient_name} · <bdi dir="ltr">{maskPhone(address.recipient_phone, locale)}</bdi>
              </p>
            </Card>
          </>
        ) : null}
      </section>

      <section className="shco-sec" aria-labelledby="shco-slots">
        <h2 id="shco-slots" className="shp-sec-title">
          {t('checkout.slots')}
        </h2>
        {data.shipments.map((s) => (
          <Card key={s.seller.id} className="shco-card">
            <p className="shco-line-main">{s.seller.name}</p>
            {s.slots.length ? (
              <ChipGroup label={`${t('checkout.slots')} · ${s.seller.name}`}>
                {s.slots.slice(0, 9).map((slot) => (
                  <PillChip
                    key={slot.id}
                    pressed={slots[s.seller.id] === slot.id}
                    onPressedChange={(on) =>
                      setSlots((cur) => {
                        const next = { ...cur };
                        if (on) next[s.seller.id] = slot.id;
                        else delete next[s.seller.id];
                        return next;
                      })
                    }
                  >
                    {slotText(slot)}
                  </PillChip>
                ))}
              </ChipGroup>
            ) : (
              <p className="shco-line-sub">{t('checkout.slotNone')}</p>
            )}
          </Card>
        ))}
      </section>

      {data.discreet_packaging_available ? (
        <Card className="shco-card shco-row">
          <div className="shco-row-main">
            <p id="shco-plain" className="shco-line-main">
              {t('checkout.discreet')}
            </p>
            <p className="shco-line-sub">{t('checkout.discreetBody')}</p>
          </div>
          <Switch checked={discreet} onCheckedChange={setDiscreet} labelledBy="shco-plain" />
        </Card>
      ) : null}

      <section className="shco-sec" aria-labelledby="shco-pay">
        <h2 id="shco-pay" className="shp-sec-title">
          {t('checkout.payment')}
        </h2>
        {/* Pay on delivery is the only method (DECISIONS #13/#15): shown as the fixed choice, not a picker. */}
        <Card className="shco-card shco-cod">
          <Icon name="checkCircle" size={22} />
          <div>
            <p className="shco-line-main">{t('checkout.cod')}</p>
            <p className="shco-line-sub">{t('checkout.codBody')}</p>
          </div>
        </Card>
        {codBlockers.map((s) => (
          <p key={s.seller.id} className="shco-error" role="alert">
            {t('checkout.codLimit', { seller: s.seller.name, amount: price(s.seller.cod_max_total ?? 0) })}
          </p>
        ))}
      </section>

      <Card className="shcart-totals">
        <dl>
          <div className="shcart-row">
            <dt>{t('checkout.items')}</dt>
            <dd>{price(data.items_total)}</dd>
          </div>
          {data.discount_total > 0 ? (
            <div className="shcart-row">
              <dt>{t('checkout.discount')}</dt>
              <dd>− {price(data.discount_total)}</dd>
            </div>
          ) : null}
          <div className="shcart-row">
            <dt>{t('checkout.shipping', { n: formatNumber(data.shipments.length, locale) })}</dt>
            <dd>{data.shipping_total === 0 ? t('cart.freeShipDone') : price(data.shipping_total)}</dd>
          </div>
          <div className="shcart-row is-total">
            <dt>{t('checkout.payable')}</dt>
            <dd>{price(data.payable_total)}</dd>
          </div>
        </dl>
      </Card>

      {/* The COD ceiling is explained per seller above; every other blocker has its own line. */}
      {blockers
        .filter((b) => b !== 'cod_limit_exceeded')
        .map((b) => (
          <p key={b} className="shco-error" role="alert">
            {t(`checkout.blockers.${checkoutBlockerKey(b)}`)}
          </p>
        ))}
      {place.isError ? (
        <p className="shco-error" role="alert">
          {shopErrorKey(place.error) === 'generic' ? t('checkout.placeFailed') : t(`errors.${shopErrorKey(place.error)}`)}
        </p>
      ) : null}
      <p className="shp-note">{t('home.cod')}</p>

      <div className="shp-buybar">
        <div className="shp-buy-info">
          <span className="shp-buy-variant">{t('checkout.payOnDelivery', { amount: price(data.payable_total) })}</span>
        </div>
        <PrimaryButton disabled={!canPlace} loading={place.isPending} onClick={submit}>
          {place.isPending ? t('checkout.placing') : t('checkout.place')}
        </PrimaryButton>
      </div>

      {sheet ? (
        <AddressSheet
          addresses={data.addresses}
          selectedId={addressId}
          onSelect={setAddressId}
          onClose={() => setSheet(false)}
        />
      ) : null}
    </>
  );
}

/**
 * `/shop/checkout` — «تکمیل خرید» (CB-SHOP-08, nbd_Shop_Checkout): stepper, address + recipient, a delivery slot per
 * seller, plain packaging, pay on delivery (the only method — no bank-gateway row, DECISIONS #13) and totals.
 * Placing sends one idempotency key per attempt, reused on retry. Addresses and phones are never logged.
 */
export function ShopCheckoutPage() {
  const t = useTranslations('shop');
  const router = useRouter();
  const q = useShopCheckout();
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
            <SecondaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
              {t('retry')}
            </SecondaryButton>
          }
        />
      </Card>
    );
  } else if (q.data.cart.items_count === 0 && q.data.cart.unavailable_items.length === 0) {
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
    body = <Review data={q.data} />;
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('checkout.title')} onBack={() => router.back('/shop/cart')} backLabel={t('checkout.back')} />
        <div className="shp-body shco-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
