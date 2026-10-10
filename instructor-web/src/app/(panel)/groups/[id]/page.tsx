import { notFound } from 'next/navigation';

import { GroupScreen } from '@/screens/group';

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return <GroupScreen id={n} />;
}
