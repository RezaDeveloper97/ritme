import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ShopLayetteFormScreen } from '@/screens/shop';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('shopAdmin');
  return { title: t('titles.layette') };
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <ShopLayetteFormScreen id={id} />;
}
