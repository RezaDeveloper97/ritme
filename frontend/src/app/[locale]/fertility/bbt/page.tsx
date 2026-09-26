import { setRequestLocale } from "next-intl/server";

import { FertilityBbtPage } from "@/screens/fertility-bbt";

import { RouteMessages } from "../../../RouteMessages";

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ range?: string }>;
}

export default async function FertilityBbtRoute({
  params,
  searchParams,
}: Props) {
  const { locale } = await params;
  const { range } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="fertilityBbt">
      <FertilityBbtPage range={range} />
    </RouteMessages>
  );
}
