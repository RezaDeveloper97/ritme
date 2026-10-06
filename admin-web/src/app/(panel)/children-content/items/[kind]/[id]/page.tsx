import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';

import { ChildItemFormScreen, isChildKind } from '@/screens/child-content';
import { parseRouteId } from '@/shared/lib';

type Props = { params: Promise<{ kind: string; id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const t = await getTranslations('childContent');
  const { kind } = await params;
  return { title: isChildKind(kind) ? t(`kinds.${kind}`) : t('title') };
}

export default async function ChildItemPage({ params }: Props) {
  const { kind, id: raw } = await params;
  const id = parseRouteId(raw);
  if (!isChildKind(kind) || id === null) notFound();
  return <ChildItemFormScreen kind={kind} id={id} />;
}
