import { NextIntlClientProvider } from "next-intl";
import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import en from "../../../../messages/en.json";
import type { SelfTestResult, SelfTestRun } from "../api/safety";
import { RunResults, RunsList } from "./SelfTestRuns";

const render = (node: ReactNode) =>
  renderToStaticMarkup(
    <NextIntlClientProvider locale="en" messages={en} timeZone="Asia/Tehran">
      {node}
    </NextIntlClientProvider>,
  );

const runs: SelfTestRun[] = [
  {
    run_id: "01JB0000000000000000000002",
    started_at: "2026-10-31T03:00:00+03:30",
    scenarios: 26,
    passed: 25,
    failed: 1,
  },
  {
    run_id: "01JB0000000000000000000001",
    started_at: "2026-10-30T03:00:00+03:30",
    scenarios: 26,
    passed: 26,
    failed: 0,
  },
];

describe("RunsList", () => {
  it("lists every run with its result and marks the selected one", () => {
    const html = render(
      <RunsList
        runs={runs}
        selected="01JB0000000000000000000002"
        onSelect={() => undefined}
      />,
    );
    expect(html.match(/<button/g)).toHaveLength(2);
    expect(html).toContain("1 failed");
    expect(html).toContain("All passed");
    expect(html).toContain("25/26");
    expect(html.match(/aria-pressed="true"/g)).toHaveLength(1);
  });

  it("says when nothing ran yet", () => {
    expect(
      render(<RunsList runs={[]} selected={null} onSelect={() => undefined} />),
    ).toContain("No run yet");
  });
});

describe("RunResults", () => {
  it("shows each scenario with its status and the failure detail", () => {
    const results: SelfTestResult[] = [
      {
        scenario: "config.locked_rules",
        kind: "config",
        passed: false,
        duration_ms: 4,
        detail: {
          threshold: "locked, active",
          problems: ["preg.bp_severe: inactive"],
        },
      },
      {
        scenario: "vitals.bp_crisis_systolic",
        kind: "path",
        passed: true,
        duration_ms: 0,
        detail: {
          threshold: "systolic > 180",
          probes: [{ input: "181/119", want: true, got: true }],
        },
      },
    ];
    const html = render(<RunResults results={results} loading={false} />);
    expect(html).toContain("preg.bp_severe: inactive");
    expect(html).toContain("Configuration");
    expect(html).toContain("1 inputs checked");
    expect(html).toContain("systolic &gt; 180");
    expect(html).toContain("Failed");
  });
});
