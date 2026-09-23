import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { TaskTemplateFormScreen } from '@/screens/task-templates';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('taskTemplates');
  return { title: t('edit') };
}

export default async function TaskTemplatesIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <TaskTemplateFormScreen id={id} />;
}
