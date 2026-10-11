import { setRequestLocale } from 'next-intl/server';

import { ShopChecklistPage } from '@/screens/shop-checklist';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/shop/checklist` — «لیست سیسمونی» (CB-SHOP-09, nbd_Shop_Checklist). `?for=<ownerId>` = a family member's read-only view. */
export default async function ShopChecklistRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="shopChecklist">
      <ShopChecklistPage />
    </RouteMessages>
  );
}
