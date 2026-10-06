import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { VaccinesScreen } from '@/screens/child-content';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('childContent');
  return { title: t('pages.vaccines') };
}

export default function ChildContentVaccinesPage() {
  return <VaccinesScreen />;
}
