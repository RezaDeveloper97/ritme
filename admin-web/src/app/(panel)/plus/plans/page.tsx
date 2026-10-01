import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PlansScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.plans');
  return { title: t('title') };
}

export default function PlusPlansPage() {
  return <PlansScreen />;
}
