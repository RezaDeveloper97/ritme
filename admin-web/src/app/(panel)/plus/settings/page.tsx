import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PlusSettingsScreen } from '@/screens/plus-billing';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('plus.settings');
  return { title: t('title') };
}

export default function PlusSettingsPage() {
  return <PlusSettingsScreen />;
}
