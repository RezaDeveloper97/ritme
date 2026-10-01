import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { DiscountsScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.discounts');
  return { title: t('title') };
}

export default function PlusDiscountsPage() {
  return <DiscountsScreen />;
}
