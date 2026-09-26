import { setRequestLocale } from "next-intl/server";

import { FertilityInsightsPage } from "@/screens/fertility-insights";

import { RouteMessages } from "../../../RouteMessages";

interface Props {
  params: Promise<{ locale: string }>;
}

export default async function FertilityInsightsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="fertilityInsights">
      <FertilityInsightsPage />
    </RouteMessages>
  );
}
