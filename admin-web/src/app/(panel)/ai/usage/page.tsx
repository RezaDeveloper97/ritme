import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { AiUsageScreen } from '@/screens/ai-credits';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('aiCredits.usage');
  return { title: t('title') };
}

export default function AiUsagePage() {
  return <AiUsageScreen />;
}
