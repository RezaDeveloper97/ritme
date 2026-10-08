import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { LearningCoursesScreen } from '@/screens/learning';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('learning.courses');
  return { title: t('title') };
}

export default function Page() {
  return <LearningCoursesScreen />;
}
