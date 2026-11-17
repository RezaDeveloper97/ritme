'use client';

/*
 * Dev-only playground of the UI autopilot, route /dev/autopilot (404 in production builds). Paste or pick
 * a script, choose a speed, watch it drive the real app, read the report — and copy the agent manifest
 * this account would send to a model. Tool copy for developers: never reaches users, so not translated
 * (same as /dev/ui-kit).
 */
import { useEffect, useMemo, useRef, useState } from 'react';

import { runAutopilot, useAutopilotStatus, type PilotPreset, type RunHandle, type RunReport } from '@/shared/autopilot';
import type { AutopilotManifest, PilotStep } from '@/shared/pilot';
import { ChipGroup, PillChip, PrimaryButton, SecondaryButton, SegmentedTabs } from '@/shared/ui';
import { useAutopilotManifest } from '@/widgets/autopilot-manifest';

type Tab = 'run' | 'manifest';

const PRESETS: readonly PilotPreset[] = ['human', 'brisk', 'instant'];

const script = (steps: PilotStep[]): string => JSON.stringify({ version: 1, steps }, null, 2);

/** A fill-in script built from this account's own taxonomy, so the codes are always real. */
function fillSample(manifest: AutopilotManifest | null, save: boolean): string {
  const fields = manifest?.targets.filter((t) => t.id.startsWith('log.param:')) ?? [];
  const single = fields.find((t) => t.kind === 'single' && t.options?.length);
  const multi = fields.find((t) => t.kind === 'multi' && (t.options?.length ?? 0) >= 2);
  const number = fields.find((t) => (t.kind === 'number' || t.kind === 'integer') && t.min !== undefined);
  const text = fields.find((t) => t.kind === 'text');
  const steps: PilotStep[] = [];
  if (single?.options) steps.push({ do: 'set', target: single.id, value: single.options[0].value });
  if (multi?.options) steps.push({ do: 'set', target: multi.id, value: multi.options.slice(0, 2).map((o) => o.value) });
  if (number) steps.push({ do: 'set', target: number.id, value: Math.ceil(((number.min ?? 0) + (number.max ?? 10)) / 2) });
  if (text) steps.push({ do: 'set', target: text.id, value: 'نمونه‌ی آزمایشی autopilot' });
  if (save) steps.push({ do: 'tap', target: 'log.save', say: 'ذخیره می‌کنم' });
  return script(steps.length ? steps : [{ do: 'open', sheet: 'log' }]);
}

const REPORT_KEY = 'ap-lab-report';

/** Ids and codes only — a report never carries a value (docs/autopilot.md §1). */
function saveLastReport(report: RunReport): void {
  try {
    window.sessionStorage.setItem(REPORT_KEY, JSON.stringify(report));
  } catch {
    // Storage blocked: the report still shows while this page stays mounted.
  }
}

function readLastReport(): RunReport | null {
  try {
    const raw = window.sessionStorage.getItem(REPORT_KEY);
    return raw ? (JSON.parse(raw) as RunReport) : null;
  } catch {
    return null;
  }
}

export function AutopilotLabPage() {
  const [tab, setTab] = useState<Tab>('run');
  const [text, setText] = useState(() => script([{ do: 'open', route: '/calendar' }]));
  const [preset, setPreset] = useState<PilotPreset>('human');
  // A script that leaves this page unmounts it: the last report is kept for when you come back.
  const [report, setReport] = useState<RunReport | null>(null);
  // After mount, not in the initial state: the server render has no session storage to match.
  useEffect(() => setReport(readLastReport()), []);
  const [copied, setCopied] = useState(false);
  const handle = useRef<RunHandle | null>(null);
  const { status } = useAutopilotStatus();
  const { manifest, loading } = useAutopilotManifest();
  const manifestJson = useMemo(() => (manifest ? JSON.stringify(manifest, null, 2) : ''), [manifest]);

  const samples: ReadonlyArray<{ label: string; build: () => string }> = [
    { label: 'Open a page', build: () => script([{ do: 'open', route: '/calendar' }]) },
    { label: 'Open the log sheet', build: () => script([{ do: 'open', sheet: 'log' }]) },
    { label: 'Fill, no save', build: () => fillSample(manifest, false) },
    { label: 'Fill and save', build: () => fillSample(manifest, true) },
    { label: 'Unknown target', build: () => script([{ do: 'tap', target: 'log.cat:does-not-exist', timeoutMs: 1500 }]) },
  ];

  const run = () => {
    setReport(null);
    const next = runAutopilot(text, { preset });
    handle.current = next;
    void next.done.then((result) => {
      saveLastReport(result);
      setReport(result);
    });
  };

  const copy = async () => {
    await navigator.clipboard.writeText(manifestJson);
    setCopied(true);
  };

  return (
    <div className="view aplab">
      <div className="scroll">
        <div className="aplab-body">
          <h1 className="aplab-title">Autopilot lab</h1>
          <SegmentedTabs<Tab>
            tabs={[
              { value: 'run', label: 'Run a script' },
              { value: 'manifest', label: 'Agent manifest' },
            ]}
            value={tab}
            onChange={setTab}
            label="Autopilot lab"
          />

          {tab === 'run' ? (
            <>
              <ChipGroup label="Samples">
                {samples.map((sample) => (
                  <PillChip key={sample.label} pressed={false} onPressedChange={() => setText(sample.build())}>
                    {sample.label}
                  </PillChip>
                ))}
              </ChipGroup>
              <textarea
                className="aplab-code"
                dir="ltr"
                spellCheck={false}
                aria-label="Script JSON"
                value={text}
                onChange={(event) => setText(event.target.value)}
              />
              <ChipGroup label="Speed" layout="fill">
                {PRESETS.map((name) => (
                  <PillChip key={name} pressed={preset === name} onPressedChange={() => setPreset(name)}>
                    {name}
                  </PillChip>
                ))}
              </ChipGroup>
              <div className="aplab-row">
                <PrimaryButton onClick={run} disabled={status !== 'idle'}>
                  Run
                </PrimaryButton>
                <SecondaryButton onClick={() => handle.current?.cancel()} disabled={status === 'idle'}>
                  Cancel
                </SecondaryButton>
              </div>
              {report ? (
                <pre className="aplab-code is-out" dir="ltr">
                  {JSON.stringify(report, null, 2)}
                </pre>
              ) : null}
            </>
          ) : (
            <>
              <p className="aplab-note">
                {loading ? 'Loading the log taxonomy…' : `${manifest?.targets.length ?? 0} targets · ${manifest?.routes.length ?? 0} routes`}
              </p>
              <SecondaryButton onClick={copy} disabled={!manifest}>
                {copied ? 'Copied' : 'Copy JSON'}
              </SecondaryButton>
              <pre className="aplab-code is-out" dir="ltr">
                {manifestJson}
              </pre>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
