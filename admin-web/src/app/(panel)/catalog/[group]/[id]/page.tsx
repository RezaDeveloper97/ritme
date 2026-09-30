import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { CatalogItemFormScreen, isCatalogCode } from '@/screens/catalog';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('catalog');
  return { title: t('editItem') };
}

export default async function CatalogItemPage({ params }: { params: Promise<{ group: string; id: string }> }) {
  const { group, id: raw } = await params;
  const id = parseRouteId(raw);
  if (!isCatalogCode(group) || id === null) notFound();
  return <CatalogItemFormScreen group={group} id={id} />;
}
