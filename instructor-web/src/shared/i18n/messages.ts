import fa from '../../../messages/fa.json';

// The panel ships Persian only for now (RTL). A second language means a second
// bundle here and a cookie in request.ts, like admin-web.
export const DEFAULT_LOCALE = 'fa';
export type Messages = typeof fa;
export const MESSAGES: Record<string, Messages> = { fa };
export const DIRECTION = 'rtl' as const;
