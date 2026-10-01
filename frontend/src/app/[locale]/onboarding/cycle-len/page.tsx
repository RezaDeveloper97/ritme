import { redirect } from 'next/navigation';

interface Props { params: Promise<{ locale: string }> }

/** Legacy step, merged into `/onboarding/cycle` by onboarding v2 (B-N2-02); kept so old links and resume markers land. */
export default async function CycleLenRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}/onboarding/cycle`);
}
