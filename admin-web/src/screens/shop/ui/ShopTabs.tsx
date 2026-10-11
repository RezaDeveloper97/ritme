'use client';

import { useLocale, useTranslations } from 'next-intl';
import { usePathname } from 'next/navigation';

import { formatNumber } from '@/shared/lib';
import { Badge, LinkTabs, type BadgeTone } from '@/shared/ui';

/**
 * CB-SHOP-05: the shop section's tabs. bloom's products / categories (B-N10-03) stay first; the internal-shop
 * screens are added after them — one shop section, no second one. The order tabs need the `support` capability
 * (the API answers 403 otherwise).
 */
const CATALOG = [
  ['products', '/shop'],
  ['categories', '/shop/categories'],
  ['sellers', '/shop/sellers'],
  ['stock', '/shop/stock'],
  ['slots', '/shop/slots'],
  ['reviews', '/shop/reviews'],
  ['discounts', '/shop/discount-codes'],
  ['layette', '/shop/layette'],
] as const;

const ORDERS = [
  ['shipments', '/shop/shipments'],
  ['returns', '/shop/returns'],
] as const;

export function ShopTabs({ group = 'catalog' }: { group?: 'catalog' | 'orders' }) {
  const t = useTranslations('shopAdmin.tabs');
  const pathname = usePathname();
  const items = (group === 'catalog' ? CATALOG : ORDERS).map(([key, href]) => ({
    key,
    href,
    label: t(key),
    active: href === '/shop' ? pathname === '/shop' : pathname === href || pathname.startsWith(`${href}/`),
  }));
  return <LinkTabs label={t('label')} items={items} />;
}

/** Integer toman, in the UI locale's digits. */
export function Money({ value }: { value: number | null | undefined }) {
  const t = useTranslations('shopAdmin');
  const locale = useLocale();
  if (value === null || value === undefined) return <>—</>;
  return <span className="whitespace-nowrap">{t('toman', { amount: formatNumber(value, locale) })}</span>;
}

const TONES: Record<string, BadgeTone> = {
  awaiting_seller: 'amber',
  confirmed: 'brand',
  shipped: 'data',
  delivered: 'green',
  cancelled: 'neutral',
  placed: 'amber',
  requested: 'amber',
  approved: 'green',
  rejected: 'red',
  pending: 'amber',
  published: 'green',
};

/** A shipment / order / return / review status, labelled from `shopAdmin.status`. */
export function StatusBadge({ status }: { status: string }) {
  const t = useTranslations('shopAdmin.status');
  const key = status as 'pending';
  const label = t.has(key) ? t(key) : status;
  return <Badge tone={TONES[status] ?? 'neutral'}>{label}</Badge>;
}
