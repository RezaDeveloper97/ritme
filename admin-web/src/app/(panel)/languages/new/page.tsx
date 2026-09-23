import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LanguageFormScreen } from '@/screens/languages';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('languages');
  return { title: t('new') };
}

export default function LanguagesNewPage() {
  return <LanguageFormScreen id={null} />;
}
