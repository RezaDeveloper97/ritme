import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ShopProductFormScreen } from '@/screens/shop';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shop');
  return { title: t('edit') };
}

export default async function ShopIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <ShopProductFormScreen id={id} />;
}
