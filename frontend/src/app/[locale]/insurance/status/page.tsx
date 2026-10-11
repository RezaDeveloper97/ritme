import { setRequestLocale } from 'next-intl/server';

import { InsuranceStatusPage } from '@/screens/insurance-status';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/insurance/status` — وضعیت درخواست‌ها (CB-INS-05, nbl_Ins_Status). */
export default async function InsuranceStatusRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="insuranceStatus">
      <InsuranceStatusPage />
    </RouteMessages>
  );
}
