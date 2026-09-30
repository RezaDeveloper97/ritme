import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { SupportReportDetailScreen } from '@/screens/support-reports';
import { parseRouteId } from '@/shared/lib';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('supportReports');
  return { title: t('detailTitle') };
}

export default async function SupportReportsIdPage({ params }: { params: Promise<{ id: string }> }) {
  const id = parseRouteId((await params).id);
  if (id === null) notFound();
  return <SupportReportDetailScreen id={id} />;
}
