import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CarePlanScreen } from '@/screens/pregnancy-care-plan';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyCarePlan');
  return { title: t('title') };
}

export default function PregnancyCarePlanPage() {
  return <CarePlanScreen />;
}
