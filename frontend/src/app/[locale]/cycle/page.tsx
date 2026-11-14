import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/cycle` — the cycle history now lives in the «تاریخچه» tab of `/analysis`. The old URL keeps working. */
export default async function CycleRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}/analysis?tab=history`);
}
