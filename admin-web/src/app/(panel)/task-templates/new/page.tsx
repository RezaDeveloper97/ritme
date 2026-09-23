import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { TaskTemplateFormScreen } from '@/screens/task-templates';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('taskTemplates');
  return { title: t('new') };
}

export default function TaskTemplatesNewPage() {
  return <TaskTemplateFormScreen id={null} />;
}
