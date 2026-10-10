import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { DepartmentsScreen } from '@/screens/assistant-departments';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('assistantDepartments');
  return { title: t('title') };
}

export default function Page() {
  return <DepartmentsScreen />;
}
