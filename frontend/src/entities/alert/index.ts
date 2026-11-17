// Public API of the `alert` entity — the generic alerts inbox (SPEC §5.4.1). Import only from here.
export { ALERT_LEVELS, type AlertEvidence, type AlertItem, type AlertLevel, type AlertListState, type AlertPage, type AlertState } from './model/types';
export { alertKeys } from './api/keys';
export { alertItemSchema, parseAlertPage } from './api/schema';
export { fetchAlerts, useAlertAction, useAlerts, type AlertAction } from './api/queries';
export { callHref, canAck, canDismiss, evidenceRows, groupByLevel, isOpen, type AlertSection } from './model/view';
