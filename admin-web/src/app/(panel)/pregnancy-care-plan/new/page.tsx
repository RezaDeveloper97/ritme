import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CareItemFormScreen } from '@/screens/pregnancy-care-plan';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyCarePlan');
  return { title: t('new') };
}

export default function PregnancyCarePlanNewPage() {
  return <CareItemFormScreen id={null} />;
}
