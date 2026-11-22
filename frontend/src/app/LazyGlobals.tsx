'use client';

import dynamic from 'next/dynamic';

/**
 * Root-layout pieces that render nothing (or a sheet) until something happens. Each one's barrel drags a
 * stylesheet (plus, lab, onb, pwa, ntf, aigate…) that would otherwise load before `globals.css` on EVERY route
 * (T-M12-27b, finding P-01). They are `ssr:false` — none is part of the first paint — and load after hydration.
 */
export const LazyNotificationSync = dynamic(
  () => import('@/features/push-notifications').then((m) => m.NotificationSync),
  { ssr: false },
);
export const LazyStoreBillingSync = dynamic(
  () => import('@/features/purchase-plus').then((m) => m.StoreBillingSync),
  { ssr: false },
);
export const LazySessionRefresher = dynamic(
  () => import('@/features/auth').then((m) => m.SessionRefresher),
  { ssr: false },
);
export const LazyPendingConsentsSheet = dynamic(
  () => import('@/features/auth').then((m) => m.PendingConsentsSheet),
  { ssr: false },
);
export const LazyAiGateSheet = dynamic(() => import('@/features/ai-gate').then((m) => m.AiGateSheet), {
  ssr: false,
});
export const LazyInstallPrompt = dynamic(() => import('@/shared/pwa').then((m) => m.InstallPrompt), {
  ssr: false,
});
