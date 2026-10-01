import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { PaymentDetailScreen } from '@/screens/plus-billing';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.payments');
  return { title: t('detailTitle') };
}

export default async function PlusPaymentPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <PaymentDetailScreen id={id} />;
}
