import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { ChallengeFormScreen } from '@/screens/challenges';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('challenges');
  return { title: t('new') };
}

export default function ChallengesNewPage() {
  return <ChallengeFormScreen id={null} />;
}
