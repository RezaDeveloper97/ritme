import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { CatalogItemFormScreen, isCatalogCode } from '@/screens/catalog';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('catalog');
  return { title: t('newItem') };
}

export default async function CatalogNewItemPage({ params }: { params: Promise<{ group: string }> }) {
  const { group } = await params;
  if (!isCatalogCode(group)) notFound();
  return <CatalogItemFormScreen group={group} id={null} />;
}
