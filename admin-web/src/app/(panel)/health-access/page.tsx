import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { HealthAccessScreen } from '@/screens/access-control';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('access');
  return { title: t('healthTitle') };
}

export default function Page() {
  return <HealthAccessScreen />;
}
