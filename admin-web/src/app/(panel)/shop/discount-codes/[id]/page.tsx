import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ShopDiscountCodeFormScreen } from '@/screens/shop';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shopAdmin');
  return { title: t('titles.discounts') };
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <ShopDiscountCodeFormScreen id={id} />;
}
