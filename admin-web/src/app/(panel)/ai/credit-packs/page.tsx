import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { PacksScreen } from '@/screens/ai-credits';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.packs');
  return { title: t('title') };
}

export default function AiCreditPacksPage() {
  return <PacksScreen />;
}
