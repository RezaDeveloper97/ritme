// Public API of the `voice-log` feature: one round orb → record → the server transcribes the clip and a model
// writes a UI autopilot script for the log sheet → the autopilot fills «ثبت دستی» in front of the user, who
// reviews it there and saves herself. The log sheet (`features/log-day`) receives the panel as its `VoiceLog`
// slot from the screen. Import only from here (CLAUDE.md §3.3).
export { VoiceLogPanel, type VoiceLogPanelProps } from './ui/VoiceLogPanel';
export { logSheetManifest } from './model/pilot';
export { postVoicePilot, useVoicePilot, type VoicePilotResult } from './api/voice';
