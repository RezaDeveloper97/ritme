import { notFound } from 'next/navigation';

import { CourseScreen } from '@/screens/course';

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();
  return <CourseScreen id={n} />;
}
