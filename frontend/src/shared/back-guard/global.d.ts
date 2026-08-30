export {};

declare global {
  interface Window {
    /**
     * Installed by `BackGuard`. The Android shell calls it on a hardware back
     * press: `true` means the page handled it (a sheet was dismissed), `false`
     * that the shell should exit instead of navigating.
     */
    __ritmeBack?: () => boolean;
  }
}
