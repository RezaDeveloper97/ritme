import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { CheckupTypesScreen } from '@/screens/checkup-types';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('checkupTypes');
  return { title: t('title') };
}

export default function CheckupTypesPage() {
  return <CheckupTypesScreen />;
}
