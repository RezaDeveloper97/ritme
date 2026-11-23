'use client';

import { useSearchParams } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  RETURN_REASONS,
  type ReturnReason,
  type ShopOrder,
  type ShopOrderItem,
  shopErrorKey,
  useCancelOrder,
  useCreateReview,
  useRequestReturn,
  useShopOrder,
  useShopPrice,
  useSlotText,
} from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatTimeOfDay } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  StepTimeline,
  type TimelineStep,
} from '@/shared/ui';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/shp.css';
import './shord.css';
import '@/shared/ui/styles/shco.css';
import '@/shared/ui/styles/shcart.css';
import './shrev.css';

type Shipment = ShopOrder['shipments'][number];

const STATUS_KEYS = ['placed', 'confirmed', 'shipped', 'delivered', 'cancelled', 'awaiting_seller'] as const;
const RETURN_STATUS_KEYS = ['requested', 'approved', 'rejected', 'received', 'refunded'] as const;

function useEventTime() {
  const locale = useLocale() as Locale;
  return (at: string | null) => {
    if (!at) return undefined;
    const d = new Date(at);
    return Number.isNaN(d.getTime()) ? undefined : `${formatDayMonth(d, locale)} ${formatTimeOfDay(d, locale)}`;
  };
}

/** Seller confirms → shipped → delivered, from the shipment status (the board's per-seller timeline). */
function ShipmentCard({ s, onReturn, onReview }: { s: Shipment; onReturn: (item: ShopOrderItem) => void; onReview: (item: ShopOrderItem) => void }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const slotText = useSlotText();
  const when = useEventTime();
  const at = (status: string) => when(s.timeline.find((e) => e.status === status)?.at ?? null);

  const rank = { awaiting_seller: 0, confirmed: 1, shipped: 2, delivered: 3 }[s.status as 'awaiting_seller'] ?? 0;
  const cancelled = s.status === 'cancelled';
  const state = (i: number): TimelineStep['state'] => (rank > i ? 'done' : rank === i ? 'current' : 'todo');
  const steps: TimelineStep[] = [
    { id: 'confirm', title: t('order.stepConfirm'), state: state(0), meta: at('confirmed') },
    { id: 'ship', title: t('order.stepShip'), state: state(1), meta: at('shipped') ?? (s.tracking_code ? t('order.tracking', { code: s.tracking_code }) : undefined) },
    {
      id: 'deliver',
      title: t('order.stepDeliver'),
      state: rank === 3 ? 'done' : state(2),
      meta: rank === 3 ? at('delivered') : s.slot ? slotText(s.slot) : t('order.slotNone'),
    },
  ];
  const statusKey = (STATUS_KEYS as readonly string[]).includes(s.status) ? (s.status as (typeof STATUS_KEYS)[number]) : 'placed';
  const returnable = s.status === 'delivered' && s.return_until && new Date(s.return_until).getTime() > Date.now();

  return (
    <Card className="shord-ship">
      <div className="shord-ship-head">
        <h2 className="shord-ship-name">{s.seller?.name ?? t('title')}</h2>
        <StatusPill tone={cancelled ? 'neutral' : s.status === 'delivered' ? 'success' : 'brand'}>{t(`order.status.${statusKey}`)}</StatusPill>
      </div>
      {cancelled ? null : (
        <StepTimeline steps={steps} label={t('order.timeline')} stateLabels={{ done: t('order.stateLabels.done'), current: t('order.stateLabels.current'), todo: t('order.stateLabels.todo') }} marker="dot" locale={locale} />
      )}
      {rank === 0 && !cancelled ? (
        <p className="shord-hint">
          {t('order.waiting')} · {t('order.waitingHint')}
        </p>
      ) : null}
      <ul className="shord-items">
        {s.items.map((it) => (
          <li key={it.id} className="shord-item">
            <span className="shord-item-main">
              <span className="shord-item-title">{it.title}</span>
              {it.variant_label ? <span className="shord-item-variant">{it.variant_label}</span> : null}
            </span>
            <span className="shord-item-qty">{t('order.qty', { n: formatNumber(it.quantity, locale) })}</span>
            {returnable && it.returnable ? (
              <button type="button" className="shord-link" {...pilot(`shop-order.return:${it.id}`)} onClick={() => onReturn(it)}>
                {t('order.returnRequest')}
              </button>
            ) : null}
            {/* CB-SHOP-12 F-5: «ثبت نظر» on a delivered item, then its moderation state. */}
            {it.reviewable ? (
              <button type="button" className="shord-link" {...pilot(`shop-order.review:${it.id}`)} onClick={() => onReview(it)}>
                <Icon name="star" size={16} />
                {t('order.review')}
              </button>
            ) : it.review_status ? (
              <StatusPill tone={it.review_status === 'published' ? 'success' : 'neutral'}>{t(`order.reviewStatus.${it.review_status}`)}</StatusPill>
            ) : null}
          </li>
        ))}
      </ul>
      {s.status === 'delivered' && s.return_until ? (
        <p className="shord-hint">{t('order.returnWindow', { date: formatDayMonth(new Date(s.return_until), locale) })}</p>
      ) : null}
    </Card>
  );
}

function ReturnSheet({ orderId, item, onClose }: { orderId: number; item: ShopOrderItem; onClose: () => void }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const req = useRequestReturn(orderId);
  const [qty, setQty] = useState(1);
  const [reason, setReason] = useState<ReturnReason>('changed_mind');
  const [note, setNote] = useState('');
  return (
    <AppSheet
      open
      onClose={req.isPending ? () => undefined : onClose}
      size="full"
      title={t('return.title')}
      footer={
        <div className="shco-sheet-actions">
          <PrimaryButton
            {...pilot('shop-return.submit')}
            loading={req.isPending}
            onClick={() => req.mutate({ orderItemId: item.id, quantity: qty, reason, note: note.trim() }, { onSuccess: onClose })}
          >
            {t('return.submit')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="shco-sheet">
        <p className="shord-item-title">{item.title}</p>
        {item.quantity > 1 ? (
          <NumberStepper
            value={qty}
            onChange={setQty}
            pilot={pilot('shop-return.qty')}
            min={1}
            max={item.quantity}
            label={t('return.qty')}
            decrementLabel={t('cart.dec')}
            incrementLabel={t('cart.inc')}
            locale={locale}
          />
        ) : null}
        <ChipGroup label={t('return.reason')} pilot={pilot('shop-return.reason', 'choice')}>
          {RETURN_REASONS.map((r) => (
            <PillChip key={r} pressed={reason === r} onPressedChange={() => setReason(r)} {...pilotOption(r)}>
              {t(`return.reasons.${r}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <label className="lbl" htmlFor="shord-note">
          {t('return.note')}
        </label>
        <div className="field">
          <input id="shord-note" value={note} maxLength={1000} onChange={(e) => setNote(e.target.value)} {...pilot('shop-return.note', 'text')} />
        </div>
        {req.isError ? (
          <p className="shco-error" role="alert">
            {t('return.failed')}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

/** CB-SHOP-12 F-5: «ثبت نظر» — 1–5 stars and an optional text; it shows on the product after the admin's check. */
function ReviewSheet({ orderId, item, onClose }: { orderId: number; item: ShopOrderItem; onClose: () => void }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const create = useCreateReview(orderId);
  const [rating, setRating] = useState(0);
  const [body, setBody] = useState('');
  const [tried, setTried] = useState(false);
  const submit = () => {
    setTried(true);
    if (rating < 1) return;
    create.mutate({ orderItemId: item.id, rating, body: body.trim() }, { onSuccess: onClose });
  };
  const errorKey = create.isError ? shopErrorKey(create.error) : null;
  return (
    <AppSheet
      open
      onClose={create.isPending ? () => undefined : onClose}
      size="full"
      title={t('review.title')}
      footer={
        <div className="shco-sheet-actions">
          <PrimaryButton loading={create.isPending} onClick={submit} {...pilot('shop-review.submit')}>
            {t('review.submit')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="shco-sheet">
        <p className="shord-item-title">{item.title}</p>
        {item.variant_label ? <p className="shord-item-variant">{item.variant_label}</p> : null}
        <div className="shrev-stars" role="radiogroup" aria-label={t('review.rating')} {...pilot('shop-review.rating', 'choice')}>
          <span className="lbl" aria-hidden>{t('review.rating')}</span>
          <div className="shrev-stars-row">
            {[1, 2, 3, 4, 5].map((n) => (
              <button
                key={n}
                type="button"
                role="radio"
                aria-checked={rating === n}
                {...pilotOption(String(n))}
                aria-label={t('review.star', { n: formatNumber(n, locale) })}
                className={rating >= n ? 'shrev-star is-on' : 'shrev-star'}
                onClick={() => setRating(n)}
              >
                <Icon name="star" size={28} />
              </button>
            ))}
          </div>
        </div>
        {tried && rating < 1 ? (
          <p className="shco-error" role="alert">
            {t('review.pickRating')}
          </p>
        ) : null}
        <label className="lbl" htmlFor="shrev-body">
          {t('review.body')}
        </label>
        <div className="field shrev-field">
          <textarea id="shrev-body" className="shrev-body" rows={4} value={body} maxLength={2000} onChange={(e) => setBody(e.target.value)} {...pilot('shop-review.body', 'text')} />
        </div>
        <p className="shco-note">{t('review.hint')}</p>
        {errorKey ? (
          <p className="shco-error" role="alert">
            {errorKey === 'generic' ? t('review.failed') : t(`errors.${errorKey}`)}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

function CancelSheet({ orderId, onClose }: { orderId: number; onClose: () => void }) {
  const t = useTranslations('shop');
  const cancel = useCancelOrder(orderId);
  return (
    <AppSheet
      open
      onClose={cancel.isPending ? () => undefined : onClose}
      size="half"
      title={t('order.cancelTitle')}
      footer={
        <div className="shco-sheet-actions">
          <PrimaryButton loading={cancel.isPending} onClick={() => cancel.mutate(undefined, { onSuccess: onClose })}>
            {t('order.cancel')}
          </PrimaryButton>
          <SecondaryButton variant="text" disabled={cancel.isPending} onClick={onClose}>
            {t('order.cancelKeep')}
          </SecondaryButton>
        </div>
      }
    >
      <div className="shco-sheet">
        <p className="shco-note">{t('order.cancelBody')}</p>
        {cancel.isError ? (
          <p className="shco-error" role="alert">
            {shopErrorKey(cancel.error) === 'generic' ? t('order.cancelFailed') : t(`errors.${shopErrorKey(cancel.error)}`)}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

function OrderBody({ order, placed }: { order: ShopOrder; placed: boolean }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const price = useShopPrice();
  const [returning, setReturning] = useState<ShopOrderItem | null>(null);
  const [reviewing, setReviewing] = useState<ShopOrderItem | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const statusKey = (STATUS_KEYS as readonly string[]).includes(order.status) ? (order.status as (typeof STATUS_KEYS)[number]) : 'placed';

  return (
    <>
      <div className="shord-hero">
        <span className="shord-hero-icon" aria-hidden>
          <Icon name={order.status === 'cancelled' ? 'x' : 'checkCircle'} size={26} />
        </span>
        <h2 className="shord-hero-title">{placed && order.status !== 'cancelled' ? t('order.placedTitle') : t(`order.status.${statusKey}`)}</h2>
        <p className="shord-hero-sub">{placed ? t('order.code', { code: formatNumber(order.code, locale) }) : t('order.codeOnly', { code: formatNumber(order.code, locale) })}</p>
      </div>

      {order.shipments.map((s) => (
        <ShipmentCard key={s.id} s={s} onReturn={setReturning} onReview={setReviewing} />
      ))}

      {order.checklist_on_the_way > 0 ? (
        <Card className="shord-check">
          <Icon name="todo" size={22} />
          <div className="shord-check-main">
            <p className="shord-item-title">{t('order.checklist', { n: formatNumber(order.checklist_on_the_way, locale) })}</p>
            <p className="shord-hint">{t('order.checklistBody')}</p>
          </div>
          <Link href="/shop/checklist" className="shord-link">
            {t('order.checklistOpen')}
          </Link>
        </Card>
      ) : null}

      <Card className="shcart-totals">
        <h2 className="shp-sec-title">{t('order.details')}</h2>
        <dl>
          <div className="shcart-row">
            <dt>{t('order.items')}</dt>
            <dd>{price(order.items_total)}</dd>
          </div>
          {order.discount_total > 0 ? (
            <div className="shcart-row">
              <dt>{t('order.discount')}</dt>
              <dd>− {price(order.discount_total)}</dd>
            </div>
          ) : null}
          <div className="shcart-row">
            <dt>{t('order.shipping')}</dt>
            <dd>{order.shipping_total === 0 ? t('cart.freeShipDone') : price(order.shipping_total)}</dd>
          </div>
          <div className="shcart-row is-total">
            <dt>{t('order.payable')}</dt>
            <dd>{price(order.payable_total)}</dd>
          </div>
        </dl>
        <p className="shord-hint">
          {t('order.cod')}
          {order.discreet ? ` · ${t('order.discreetOn')}` : ''}
        </p>
      </Card>

      {order.returns.length ? (
        <ul className="shord-returns">
          {order.returns.map((r) => {
            const k = (RETURN_STATUS_KEYS as readonly string[]).includes(r.status) ? (r.status as (typeof RETURN_STATUS_KEYS)[number]) : 'requested';
            return (
              <li key={r.id}>
                <StatusPill tone="neutral">{t(`return.statuses.${k}`)}</StatusPill>
              </li>
            );
          })}
        </ul>
      ) : null}

      <div className="shord-actions">
        {order.can_cancel ? (
          <SecondaryButton onClick={() => setCancelling(true)}>{t('order.cancel')}</SecondaryButton>
        ) : null}
        {placed ? (
          <>
            <PrimaryButton onClick={() => router.push('/profile/orders')} {...pilot('shop-order.my-orders')}>{t('order.myOrders')}</PrimaryButton>
            <SecondaryButton variant="text" onClick={() => router.push('/shop')} {...pilot('shop-order.shop')}>
              {t('order.continueShopping')}
            </SecondaryButton>
          </>
        ) : null}
      </div>

      {returning ? <ReturnSheet orderId={order.id} item={returning} onClose={() => setReturning(null)} /> : null}
      {reviewing ? <ReviewSheet orderId={order.id} item={reviewing} onClose={() => setReviewing(null)} /> : null}
      {cancelling ? <CancelSheet orderId={order.id} onClose={() => setCancelling(false)} /> : null}
    </>
  );
}

/**
 * `/shop/order/[id]` — «سفارشت ثبت شد» right after checkout (`?placed=1`) and the order detail later
 * (CB-SHOP-08, nbd_Shop_Order): a timeline per seller, totals, the layette-checklist link, returns and cancel.
 */
export function ShopOrderPage({ id }: { id: number }) {
  const t = useTranslations('shop');
  const router = useRouter();
  const placed = useSearchParams().get('placed') === '1';
  const q = useShopOrder(id);
  const status = q.isError ? getApiErrorStatus(q.error) : undefined;

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')}>
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (status === 404) {
    body = (
      <Card>
        <EmptyState icon="box" title={t('order.notFoundTitle')} body={t('order.notFoundBody')} action={<PrimaryButton onClick={() => router.push('/profile/orders')}>{t('order.myOrders')}</PrimaryButton>} />
      </Card>
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
  } else {
    body = <OrderBody order={q.data} placed={placed} />;
  }

  return (
    <div className="view shp-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={placed ? t('order.placedTitle') : t('order.title')}
          onBack={() => router.push(placed ? '/shop' : '/profile/orders')}
          backLabel={t('order.back')}
        />
        <div className="shp-body shord-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
