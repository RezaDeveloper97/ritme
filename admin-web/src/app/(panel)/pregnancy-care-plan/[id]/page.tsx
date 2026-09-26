import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { CareItemFormScreen } from '@/screens/pregnancy-care-plan';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('pregnancyCarePlan');
  return { title: t('edit') };
}

export default async function PregnancyCarePlanIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <CareItemFormScreen id={id} />;
}
