'use client';

import { useRef } from 'react';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

/** «۰۹۱۲ ••• ••۴۵» — never show the full number on the review step (CB-SHOP-08, shared with the desktop checkout). */
export function maskPhone(phone: string, locale: Locale): string {
  const d = phone.replace(/\D/g, '');
  if (d.length < 6) return formatNumber(phone, locale);
  return formatNumber(`${d.slice(0, 4)} ••• ••${d.slice(-2)}`, locale);
}

/** One idempotency key per checkout page, reused on every retry (a double tap or a lost reply never orders twice). */
export function useIdempotencyKey(): () => string {
  const ref = useRef<string | null>(null);
  return () => {
    if (!ref.current) ref.current = crypto.randomUUID();
    return ref.current;
  };
}

const BLOCKERS = ['cart_empty', 'cart_unavailable', 'address_invalid', 'discount_invalid', 'discount_exhausted', 'discount_used', 'discount_min_total'] as const;

/** `shop.checkout.blockers.*` key of a checkout blocker (anything unknown reads as an unavailable item). */
export const checkoutBlockerKey = (b: string): (typeof BLOCKERS)[number] =>
  (BLOCKERS as readonly string[]).includes(b) ? (b as (typeof BLOCKERS)[number]) : 'cart_unavailable';
