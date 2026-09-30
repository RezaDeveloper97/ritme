// Public API of the `notification-settings` screen (B-N1-11). Import only from here (§3.3).
// Its own slice, not part of `profile-notifications`: that slice's inbox sheet is in the
// layout's shell scope, and a shared index would ship `me` copy to every route.
export { NotificationSettingsPage } from './ui/NotificationSettingsPage';
