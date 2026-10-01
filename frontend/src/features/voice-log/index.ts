// Public API of the `voice-log` feature (B-N3-05, Plus): record → transcribe + parse on the server → review
// chips that merge into the log sheet's draft. The log sheet (`features/log-day`) receives it as its
// `VoiceLog` slot from the screen. Import only from here (CLAUDE.md §3.3).
export { VoiceLogPanel, type VoiceLogPanelProps } from './ui/VoiceLogPanel';
export { postVoiceLog, useVoiceLog, type VoiceResult, type VoiceSuggestion } from './api/voice';
