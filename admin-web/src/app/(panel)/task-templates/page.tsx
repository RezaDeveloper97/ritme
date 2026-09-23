import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { TaskTemplatesScreen } from '@/screens/task-templates';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('taskTemplates');
  return { title: t('title') };
}

export default function TaskTemplatesPage() {
  return <TaskTemplatesScreen />;
}
