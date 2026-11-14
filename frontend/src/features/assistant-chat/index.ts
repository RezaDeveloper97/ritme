// Public API of the `assistant-chat` feature (bloom B-N7-07): the SSE chat client + thread + composer.
export { parseSse, streamMessage, type SseEvent } from './api/stream';
export { itemsOf, useChat, type ChatGateHooks, type ChatItem, type ChatState } from './model/useChat';
export { ChatThread } from './ui/ChatThread';
export { Composer } from './ui/Composer';
