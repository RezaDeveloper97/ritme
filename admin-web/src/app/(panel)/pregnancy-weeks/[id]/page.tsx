import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { PregnancyWeekFormScreen } from '@/screens/pregnancy-weeks';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyWeeks');
  return { title: t('edit') };
}

export default async function PregnancyWeeksIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <PregnancyWeekFormScreen id={id} />;
}
