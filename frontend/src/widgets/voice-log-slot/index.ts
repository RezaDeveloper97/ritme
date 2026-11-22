// Public API of the `voice-log-slot` widget: the voice recorder wired to the AI gate. Import only from here.
// `VoiceLogGated` is a lazy wrapper, so importing `useVoiceLocked` alone stays light.
export { useVoiceLocked } from './model/useVoiceLocked';
export { VoiceLogGated } from './ui/VoiceLogGatedLazy';
