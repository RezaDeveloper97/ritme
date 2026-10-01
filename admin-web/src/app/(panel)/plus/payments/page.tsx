import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PaymentsScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.payments');
  return { title: t('title') };
}

export default function PlusPaymentsPage() {
  return <PaymentsScreen />;
}
