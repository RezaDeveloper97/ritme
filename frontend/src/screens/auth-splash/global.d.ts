export {};

declare global {
  interface Window {
    /**
     * Installed by the splash route's inline script. Calling it cancels the
     * no-JS fallback navigation, which the hydrated screen does before running
     * its own timer.
     */
    __ritmeSplashFallback?: () => void;
  }
}
