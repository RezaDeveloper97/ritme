import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { SubscriptionsScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.subscriptions');
  return { title: t('title') };
}

export default function PlusSubscriptionsPage() {
  return <SubscriptionsScreen />;
}
