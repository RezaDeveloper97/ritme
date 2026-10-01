import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PlanFormScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.plans');
  return { title: t('new') };
}

export default function PlusPlanNewPage() {
  return <PlanFormScreen id={null} />;
}
