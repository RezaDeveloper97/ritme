// Public API of the UI autopilot ENGINE (docs/autopilot.md, frontend/CLAUDE.md §11.6). The vocabulary —
// `pilot()`, the script contract, the catalog shape, the manifest builder — lives in `@/shared/pilot`.
export {
  AUTOPILOT_CONFIG,
  defineAutopilotConfig,
  resolveConfig,
  type AutopilotConfig,
  type PilotPreset,
  type PilotTiming,
  type RunOptions,
} from './config';
export { cancelAutopilot, pauseAutopilot, resumeAutopilot, runAutopilot, type RunHandle } from './dom/engine';
export type { RunCode, RunReport, StepCode, StepReport } from './core/runner';
export { useAutopilotCatalog, useAutopilotStatus, type PilotStatus } from './core/store';
export { AutopilotHost } from './view/AutopilotHost';
