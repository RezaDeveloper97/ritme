// Public API of the `message` entity. Import only from here (CLAUDE.md §3.3).
export type {
  AppMode,
  UserMode,
  PrimaryMessage,
  DailyMessage,
  MessageCorrelation,
  MessagePattern,
  SupplementTips,
  SymptomOverride,
} from './model/types';
export {
  messageKeys,
  useUserMode,
  useDailyMessage,
  fetchUserMode,
  fetchDailyMessage,
  isNoDailyMessage,
} from './api/queries';
export { FEED_SLOTS } from './model/feed';
export type {
  FeedChallenge,
  FeedEvidence,
  FeedItem,
  FeedSlot,
  MessageEventName,
  MessageFeed,
} from './model/feed';
export { parseFeed } from './api/feed-schema';
export {
  HOME_FEED_SLOTS,
  feedKeys,
  fetchFeed,
  findAlertRowId,
  isFeedUnsupported,
  postMessageEvent,
  useAckAlert,
  useMessageEvent,
  useMessageFeed,
} from './api/feed';
export { dailyAdvice, feedState, localizeFeedText, slotItems, slotWinner, telHref } from './lib/feed-view';
export type { FeedState } from './lib/feed-view';
export { feedLinkTarget } from './lib/feed-link';
export type { FeedAction, FeedLinkTarget } from './lib/feed-link';
