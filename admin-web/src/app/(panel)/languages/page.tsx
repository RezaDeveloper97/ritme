import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LanguagesScreen } from '@/screens/languages';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('languages');
  return { title: t('title') };
}

export default function LanguagesPage() {
  return <LanguagesScreen />;
}
