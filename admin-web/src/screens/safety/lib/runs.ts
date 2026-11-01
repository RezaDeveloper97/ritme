import type { SelfTestResult } from "../api/safety";

/** The failure lines of a scenario: every probe whose verdict differs, then the problems (synthetic inputs only). */
export function failureLines(result: Pick<SelfTestResult, "detail">): string[] {
  const d = result.detail;
  if (!d) return [];
  const probes = (d.probes ?? [])
    .filter((p) => p.want !== p.got)
    .map((p) => `${p.input}: ${p.want} → ${p.got}`);
  return [...probes, ...(d.problems ?? [])];
}
