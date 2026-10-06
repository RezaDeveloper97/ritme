import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { MilestonesScreen } from '@/screens/child-content';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('childContent');
  return { title: t('pages.milestones') };
}

export default function ChildContentMilestonesPage() {
  return <MilestonesScreen />;
}
