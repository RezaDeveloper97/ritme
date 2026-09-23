import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ChallengeCompletionsScreen } from '@/screens/challenges';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('completions');
  return { title: t('title') };
}

export default function ChallengeCompletionsPage() {
  return <ChallengeCompletionsScreen />;
}
