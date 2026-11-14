// Public API of the `ai-gate` feature — the ONLY way an AI surface handles access (CLAUDE.md «AI gate»).
// Import only from here (CLAUDE.md §3.3).
export { useAiGate, type AiGateControls } from './model/useAiGate';
export { openAiGate } from './model/store';
export { AiGate } from './ui/AiGate';
export { AiGateSheet, CREDITS_BUY_ROUTE } from './ui/AiGateSheet';
