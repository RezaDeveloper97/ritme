export const PLUS_ANALYSIS_LONG_RANGE = 'plus.analysis_long_range';
export const PLUS_DEEP_ANALYSIS = 'plus.deep_analysis';
export const PLUS_SMS_REMINDERS = 'plus.sms_reminders';
/** «ریتمی همراه»: inviting a partner / spouse (owner's decision, 2026-11-11). */
export const PLUS_COMPANION = 'plus.companion';

export const entitlementKey = {
  longRange: PLUS_ANALYSIS_LONG_RANGE,
  deep: PLUS_DEEP_ANALYSIS,
  smsReminders: PLUS_SMS_REMINDERS,
  companion: PLUS_COMPANION,
} as const;
