import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/alerts` — alerts have no screen any more: they live in the notifications sheet. Old links open it on home. */
export default async function AlertsRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}?sheet=notifications`);
}
