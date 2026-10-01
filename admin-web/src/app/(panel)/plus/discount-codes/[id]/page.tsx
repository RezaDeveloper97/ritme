import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { DiscountFormScreen } from '@/screens/plus-billing';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.discounts');
  return { title: t('edit') };
}

export default async function PlusDiscountEditPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <DiscountFormScreen id={id} />;
}
