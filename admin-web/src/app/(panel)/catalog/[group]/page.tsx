import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { CatalogItemsScreen, isCatalogCode } from '@/screens/catalog';

type Params = { params: Promise<{ group: string }> };

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const t = await getTranslations('catalog');
  return { title: `${t('title')} · ${(await params).group}` };
}

export default async function CatalogGroupPage({ params }: Params) {
  const { group } = await params;
  if (!isCatalogCode(group)) notFound();
  return <CatalogItemsScreen group={group} />;
}
