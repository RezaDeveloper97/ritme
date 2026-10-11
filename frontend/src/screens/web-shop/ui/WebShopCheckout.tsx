'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  checkoutBlockerKey,
  maskPhone,
  type ShopAddress,
  type ShopCheckout,
  shopErrorKey,
  useDeleteAddress,
  usePlaceOrder,
  useSaveAddress,
  useIdempotencyKey,
  useShopCheckout,
  useShopPrice,
  useSlotText,
} from '@/entities/shop';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, Checkbox, EmptyState, Icon, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup, Switch } from '@/shared/ui';

import { APP_SHOP, STORE_HOME } from '../model/store';
import { CartSummary } from './WebShopCart';
import { isTeenRefusal, WebShopGate, WebShopTeen } from './WebShopGate';

function Steps() {
  const t = useTranslations('shop.checkout');
  const steps = ['stepCart', 'stepShip', 'stepConfirm'] as const;
  return (
    <ol className="wsc-flow" aria-label={t('steps')}>
      {steps.map((s, i) => (
        <li key={s} className={`wsc-flow-item${i === 1 ? ' is-current' : i === 0 ? ' is-done' : ''}`} aria-current={i === 1 ? 'step' : undefined}>
          <span className="wsc-flow-bar" aria-hidden />
          {t(s)}
        </li>
      ))}
    </ol>
  );
}

function Field({ id, label, value, onChange, max, ltr, inputMode, autoComplete }: { id: string; label: string; value: string; onChange: (v: string) => void; max: number; ltr?: boolean; inputMode?: 'tel' | 'numeric'; autoComplete?: string }) {
  return (
    <div className="wsc-field">
      <label className="lbl" htmlFor={id}>
        {label}
      </label>
      <div className="field">
        <input id={id} value={value} maxLength={max} dir={ltr ? 'ltr' : undefined} inputMode={inputMode} autoComplete={autoComplete} onChange={(e) => onChange(e.target.value)} />
      </div>
    </div>
  );
}

/** Add / edit one address inline. Address and phone are PII: held in state only, never logged. */
function AddressForm({ address, first, onDone, onCancel }: { address: ShopAddress | null; first: boolean; onDone: (id: number) => void; onCancel: () => void }) {
  const t = useTranslations('shop.address');
  const save = useSaveAddress();
  const del = useDeleteAddress();
  const [f, setF] = useState({
    label: address?.label ?? '',
    name: address?.recipient_name ?? '',
    phone: address?.recipient_phone ?? '',
    province: address?.province ?? '',
    city: address?.city ?? '',
    line: address?.address_line ?? '',
    postal: address?.postal_code ?? '',
  });
  const [isDefault, setIsDefault] = useState(address?.is_default ?? first);
  const set = (k: keyof typeof f) => (v: string) => setF((cur) => ({ ...cur, [k]: v }));
  const ok = f.name.trim() && f.phone.trim() && f.city.trim() && f.line.trim();
  const submit = () =>
    save.mutate(
      {
        id: address?.id,
        input: {
          label: f.label.trim() || null,
          recipient_name: f.name.trim(),
          recipient_phone: f.phone.trim(),
          province: f.province.trim() || null,
          city: f.city.trim(),
          address_line: f.line.trim(),
          postal_code: f.postal.trim() || null,
          is_default: isDefault,
        },
      },
      { onSuccess: (a) => onDone(a.id) },
    );
  return (
    <div className="wsc-form">
      <Field id="wsc-label" label={t('label')} value={f.label} onChange={set('label')} max={40} />
      <Field id="wsc-name" label={t('name')} value={f.name} onChange={set('name')} max={120} autoComplete="name" />
      <Field id="wsc-phone" label={t('phone')} value={f.phone} onChange={set('phone')} max={20} ltr inputMode="tel" autoComplete="tel" />
      <Field id="wsc-province" label={t('province')} value={f.province} onChange={set('province')} max={60} />
      <Field id="wsc-city" label={t('city')} value={f.city} onChange={set('city')} max={60} />
      <Field id="wsc-postal" label={t('postal')} value={f.postal} onChange={set('postal')} max={12} ltr inputMode="numeric" />
      <div className="wsc-field wsc-wide">
        <Field id="wsc-line" label={t('line')} value={f.line} onChange={set('line')} max={500} autoComplete="street-address" />
      </div>
      <div className="wsc-wide">
        <Checkbox checked={isDefault} onCheckedChange={setIsDefault} label={t('default')} />
      </div>
      {save.isError ? (
        <p className="wsc-error wsc-wide" role="alert">
          {t('saveFailed')}
        </p>
      ) : null}
      <p className="wsh-hint wsc-wide">{t('note')}</p>
      <div className="wsc-actions wsc-wide">
        <PrimaryButton loading={save.isPending} disabled={!ok} onClick={submit}>
          {t('save')}
        </PrimaryButton>
        <SecondaryButton variant="text" onClick={onCancel}>
          {t('cancel')}
        </SecondaryButton>
        {address ? (
          <SecondaryButton variant="text" disabled={save.isPending || del.isPending} onClick={() => del.mutate(address.id, { onSuccess: onCancel })}>
            {t('delete')}
          </SecondaryButton>
        ) : null}
      </div>
    </div>
  );
}

type Mode = { kind: 'view' } | { kind: 'pick' } | { kind: 'form'; address: ShopAddress | null };

function Review({ data }: { data: ShopCheckout }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const price = useShopPrice();
  const slotText = useSlotText();
  const place = usePlaceOrder();
  const nextKey = useIdempotencyKey();
  const [addressId, setAddressId] = useState<number | null>(data.default_address_id ?? data.addresses[0]?.id ?? null);
  const [mode, setMode] = useState<Mode>(data.addresses.length ? { kind: 'view' } : { kind: 'form', address: null });
  const [discreet, setDiscreet] = useState(false);
  const [slots, setSlots] = useState<Record<number, number>>({});

  const address = data.addresses.find((a) => a.id === addressId) ?? null;
  const blockers = data.blockers.filter((b) => b !== 'address_invalid' || !address).filter((b) => b !== 'cod_limit_exceeded');
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
      { onSuccess: (order) => router.replace(`${STORE_HOME}/order/${order.id}`) },
    );
  };

  return (
    <div className="wsc-cols">
      <div className="wsc-main">
        <Card className="wsc-card" aria-labelledby="wsc-h-addr">
          <div className="wsc-sec-head">
            <span className="wsc-num" aria-hidden>
              {formatNumber(1, locale)}
            </span>
            <h2 id="wsc-h-addr" className="wsh-h3">
              {t('checkout.address')}
            </h2>
            {mode.kind === 'view' && address ? (
              <button type="button" className="wsh-link-btn wsc-end" onClick={() => setMode({ kind: 'pick' })}>
                {t('checkout.change')}
              </button>
            ) : null}
          </div>
          {mode.kind === 'form' ? (
            <AddressForm
              address={mode.address}
              first={data.addresses.length === 0}
              onDone={(id) => {
                setAddressId(id);
                setMode({ kind: 'view' });
              }}
              onCancel={() => setMode(data.addresses.length ? { kind: 'view' } : { kind: 'form', address: null })}
            />
          ) : mode.kind === 'pick' ? (
            <div className="wsc-pick">
              <ul className="wsc-addr-list">
                {data.addresses.map((a) => (
                  <li key={a.id} className={`wsc-addr${a.id === addressId ? ' is-on' : ''}`}>
                    <button
                      type="button"
                      className="wsc-addr-main"
                      aria-pressed={a.id === addressId}
                      onClick={() => {
                        setAddressId(a.id);
                        setMode({ kind: 'view' });
                      }}
                    >
                      <span className="wsc-line-title">{a.label || a.city}</span>
                      <span className="wsc-line-sub">{a.address_line}</span>
                    </button>
                    <button type="button" className="wsh-link-btn" onClick={() => setMode({ kind: 'form', address: a })}>
                      {t('address.edit')}
                    </button>
                  </li>
                ))}
              </ul>
              <SecondaryButton onClick={() => setMode({ kind: 'form', address: null })}>{t('address.add')}</SecondaryButton>
            </div>
          ) : address ? (
            <div className="wsc-addr-view">
              <p className="wsc-line-title">{address.label || address.city}</p>
              <p className="wsc-line-sub">
                {address.city} · {address.address_line}
              </p>
              <p className="wsc-line-sub">
                {t('checkout.recipient')}: {address.recipient_name} · <bdi dir="ltr">{maskPhone(address.recipient_phone, locale)}</bdi>
              </p>
            </div>
          ) : null}
        </Card>

        <Card className="wsc-card" aria-labelledby="wsc-h-slots">
          <div className="wsc-sec-head">
            <span className="wsc-num" aria-hidden>
              {formatNumber(2, locale)}
            </span>
            <h2 id="wsc-h-slots" className="wsh-h3">
              {t('checkout.slots')}
            </h2>
          </div>
          {data.shipments.map((s) => (
            <div key={s.seller.id} className="wsc-ship">
              <p className="wsc-seller-name">
                <Icon name="bag" size={16} />
                {s.seller.name} · {w('checkout.items', { n: formatNumber(s.items_count, locale) })}
              </p>
              {s.slots.length ? (
                <div className="wsc-slots" role="group" aria-label={`${t('checkout.slots')} · ${s.seller.name}`}>
                  {s.slots.slice(0, 9).map((slot) => {
                    const on = slots[s.seller.id] === slot.id;
                    return (
                      <button
                        key={slot.id}
                        type="button"
                        className={`wsc-slot${on ? ' on' : ''}`}
                        aria-pressed={on}
                        onClick={() =>
                          setSlots((cur) => {
                            const next = { ...cur };
                            if (on) delete next[s.seller.id];
                            else next[s.seller.id] = slot.id;
                            return next;
                          })
                        }
                      >
                        {slotText(slot)}
                      </button>
                    );
                  })}
                </div>
              ) : (
                <p className="wsh-hint">{t('checkout.slotNone')}</p>
              )}
            </div>
          ))}
        </Card>

        {data.discreet_packaging_available ? (
          <Card className="wsc-card wsc-toggle">
            <span className="wsc-toggle-icon" aria-hidden>
              <Icon name="bag" size={20} />
            </span>
            <div className="wsc-toggle-main">
              <p id="wsc-plain" className="wsc-line-title">
                {t('checkout.discreet')}
              </p>
              <p className="wsc-line-sub">{t('checkout.discreetBody')}</p>
            </div>
            <Switch checked={discreet} onCheckedChange={setDiscreet} labelledBy="wsc-plain" />
          </Card>
        ) : null}

        <Card className="wsc-card" aria-labelledby="wsc-h-pay">
          <div className="wsc-sec-head">
            <span className="wsc-num" aria-hidden>
              {formatNumber(3, locale)}
            </span>
            <h2 id="wsc-h-pay" className="wsh-h3">
              {t('checkout.payment')}
            </h2>
          </div>
          {/* Pay on delivery is the only method: a fixed card, not a picker (DECISIONS #13/#15). */}
          <div className="wsc-cod">
            <Icon name="checkCircle" size={22} />
            <div>
              <p className="wsc-line-title">{t('checkout.cod')}</p>
              <p className="wsc-line-sub">{t('checkout.codBody')}</p>
            </div>
          </div>
          {codBlockers.map((s) => (
            <p key={s.seller.id} className="wsc-error" role="alert">
              {t('checkout.codLimit', { seller: s.seller.name, amount: price(s.seller.cod_max_total ?? 0) })}
            </p>
          ))}
        </Card>
      </div>

      <CartSummary
        footer={
          <>
            {blockers.map((b) => (
              <p key={b} className="wsc-error" role="alert">
                {t(`checkout.blockers.${checkoutBlockerKey(b)}`)}
              </p>
            ))}
            {place.isError ? (
              <p className="wsc-error" role="alert">
                {shopErrorKey(place.error) === 'generic' ? t('checkout.placeFailed') : t(`errors.${shopErrorKey(place.error)}`)}
              </p>
            ) : null}
            <button type="button" className="wsh-go wsc-cta" disabled={!canPlace} onClick={submit}>
              {place.isPending ? t('checkout.placing') : t('checkout.payOnDelivery', { amount: price(data.payable_total) })}
            </button>
            <p className="wsh-hint">{t('home.cod')}</p>
          </>
        }
      >
        <div className="wsc-row">
          <dt>{w('cart.itemsCount', { n: formatNumber(data.cart.items_count, locale) })}</dt>
          <dd>{price(data.items_total)}</dd>
        </div>
        {data.discount_total > 0 ? (
          <div className="wsc-row">
            <dt>{t('checkout.discount')}</dt>
            <dd>− {price(data.discount_total)}</dd>
          </div>
        ) : null}
        <div className="wsc-row">
          <dt>{t('checkout.shipping', { n: formatNumber(data.shipments.length, locale) })}</dt>
          <dd>{data.shipping_total === 0 ? t('cart.freeShipDone') : price(data.shipping_total)}</dd>
        </div>
        <div className="wsc-row is-total">
          <dt>{t('checkout.payable')}</dt>
          <dd>{price(data.payable_total)}</dd>
        </div>
      </CartSummary>
    </div>
  );
}

function CheckoutBody() {
  const t = useTranslations('shop');
  const q = useShopCheckout();
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
            <SecondaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
              {t('retry')}
            </SecondaryButton>
          }
        />
      </Card>
    );
  }
  if (q.data.cart.items_count === 0 && q.data.cart.unavailable_items.length === 0) {
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
  return (
    <>
      <Steps />
      <Review data={q.data} />
    </>
  );
}

/**
 * `/store/checkout` — «تکمیل خرید» on desktop (CB-SHOP-11, W_Shop_Checkout): address, a delivery slot per seller,
 * plain packaging, pay on delivery as the single fixed method (the board's gateway option is out of scope for the MVP).
 * One idempotency key per page visit, reused on retry. Addresses and phones are never logged.
 */
export function WebShopCheckout() {
  const t = useTranslations('shop');
  return (
    <WebShopGate appHref={`${APP_SHOP}/checkout`}>
      <h1 className="wsh-h1">{t('checkout.title')}</h1>
      <CheckoutBody />
    </WebShopGate>
  );
}
