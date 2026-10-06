import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ChildItemFormScreen, isChildKind, type ChildItemPrefill } from '@/screens/child-content';

type Props = {
  params: Promise<{ kind: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const t = await getTranslations('childContent');
  const { kind } = await params;
  return { title: isChildKind(kind) ? t(`newKind.${kind}`) : t('title') };
}

/** A non-negative integer query value, else undefined. */
function int(v: string | string[] | undefined): number | undefined {
  return typeof v === 'string' && /^\d{1,3}$/.test(v) ? Number(v) : undefined;
}

export default async function ChildItemNewPage({ params, searchParams }: Props) {
  const { kind } = await params;
  if (!isChildKind(kind)) notFound();
  const q = await searchParams;
  const visit = typeof q.visit === 'string' && /^[a-z][a-z0-9_]{0,31}$/.test(q.visit) ? q.visit : undefined;
  const prefill: ChildItemPrefill = { month: int(q.month), visit, age: int(q.age) };
  return <ChildItemFormScreen kind={kind} id={null} prefill={prefill} />;
}
