'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { getApiErrorStatus } from '@/shared/api';
import { Link } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { isAuthenticated } from '@/shared/session';
import { Card, EmptyState, Skeleton, SkeletonGroup } from '@/shared/ui';

import { STORE_HOME } from '../model/store';
import { ShopWebShell } from './ShopWebShell';

/**
 * Frame of the signed-in desktop pages (cart, checkout, order done): the shop shell plus a page-level login prompt
 * for a visitor without a session (these pages hold an account's cart and orders, so they are never public content).
 * `appHref` is where signed-in phone-width visitors are sent.
 */
export function WebShopGate({ appHref, children }: { appHref: string; children: ReactNode }) {
  const t = useTranslations('shop');
  const w = useTranslations('webShop');
  const mounted = useMounted();
  const authed = mounted && isAuthenticated();
  let body: ReactNode;
  if (!mounted) {
    body = (
      <SkeletonGroup label={t('loading')}>
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (!authed) {
    body = (
      <Card className="wsc-gate">
        <EmptyState
          icon="bag"
          title={w('gate.title')}
          body={w('gate.body')}
          action={
            <Link href="/signup" className="wsh-go">
              {w('product.login')}
            </Link>
          }
        />
        <Link href={STORE_HOME} className="wsh-link-btn">
          {w('gate.browse')}
        </Link>
      </Card>
    );
  } else {
    body = children;
  }
  return (
    <ShopWebShell appHref={appHref}>
      <div className="wsh wsc">{body}</div>
    </ShopWebShell>
  );
}

/** The teen-account refusal (403) shared by the three pages. */
export function WebShopTeen() {
  const t = useTranslations('shop');
  return (
    <Card>
      <EmptyState icon="shield" title={t('cart.teenTitle')} body={t('cart.teenBody')} />
    </Card>
  );
}

/** True when a failed query is the teen-account refusal. */
export const isTeenRefusal = (error: unknown): boolean => getApiErrorStatus(error) === 403;
