import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { UiKitScreen } from '@/screens/ui-kit';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('uiKit');
  return { title: t('title') };
}

export default function UiKitPage() {
  return <UiKitScreen />;
}
