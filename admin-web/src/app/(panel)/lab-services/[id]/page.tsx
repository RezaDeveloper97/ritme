import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { LabServiceFormScreen } from '@/screens/partner-lab';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('partnerLab.services');
  return { title: t('edit') };
}

export default async function LabServiceIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <LabServiceFormScreen id={id} />;
}
