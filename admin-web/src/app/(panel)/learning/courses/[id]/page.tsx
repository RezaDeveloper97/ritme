import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { LearningCourseDetailScreen } from '@/screens/learning';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('learning.course');
  return { title: t('title') };
}

export default async function LearningCoursePage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <LearningCourseDetailScreen id={id} />;
}
