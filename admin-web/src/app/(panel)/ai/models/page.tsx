import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ModelsScreen } from '@/screens/ai-credits';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.models');
  return { title: t('title') };
}

export default function AiModelsPage() {
  return <ModelsScreen />;
}
