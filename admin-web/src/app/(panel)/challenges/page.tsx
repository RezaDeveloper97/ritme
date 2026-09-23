import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ChallengesScreen } from '@/screens/challenges';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('challenges');
  return { title: t('title') };
}

export default function ChallengesPage() {
  return <ChallengesScreen />;
}
