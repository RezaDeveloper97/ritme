import type { AutopilotConfig } from '../config';
import { matchesAny } from '@/shared/pilot';
import { targetOf, type PilotScript, type PilotStep } from '@/shared/pilot';

/**
 * A script may come from a model, so it is checked before anything moves: it can only walk in-app routes,
 * only touch what the config allows, and only as many steps as the cap.
 */

export type SafetyCode = 'too_many_steps' | 'route_not_allowed' | 'target_blocked' | 'selector_not_allowed';

export interface SafetyIssue {
  /** Index into `script.steps`; -1 for the script as a whole. */
  step: number;
  code: SafetyCode;
}

const hasPrefix = (path: string, prefix: string): boolean =>
  prefix === '/' || path === prefix || path.startsWith(prefix.endsWith('/') ? prefix : `${prefix}/`);

/** An in-app, locale-agnostic path — never a URL, a protocol-relative `//host`, or a `..` walk. */
export function isSafeRoute(route: string, safety: AutopilotConfig['safety']): boolean {
  if (!route.startsWith('/') || route.startsWith('//')) return false;
  // `%` too: `/home/%2e%2e/otp` has no literal `..` but the URL parser resolves it to `/otp`.
  if (/[\s\\?#%]|\.\./.test(route) || route.includes(':')) return false;
  if (safety.blockedRoutes.some((p) => hasPrefix(route, p))) return false;
  return safety.allowedRoutes.some((p) => hasPrefix(route, p));
}

export function checkStep(step: PilotStep, index: number, safety: AutopilotConfig['safety']): SafetyIssue | null {
  if (step.do === 'open' && step.route !== undefined && !isSafeRoute(step.route, safety)) {
    return { step: index, code: 'route_not_allowed' };
  }
  const target = targetOf(step);
  if (target === undefined) return null;
  if (typeof target === 'string') {
    return matchesAny(safety.blockedTargets, target) ? { step: index, code: 'target_blocked' } : null;
  }
  const allowed = 'css' in target ? safety.allowRawSelectors : safety.allowRoleSelectors;
  return allowed ? null : { step: index, code: 'selector_not_allowed' };
}

export function checkScript(script: PilotScript, safety: AutopilotConfig['safety']): SafetyIssue[] {
  const issues: SafetyIssue[] = [];
  if (script.steps.length > safety.maxSteps) issues.push({ step: -1, code: 'too_many_steps' });
  script.steps.forEach((step, i) => {
    const issue = checkStep(step, i, safety);
    if (issue) issues.push(issue);
  });
  return issues;
}

/** Must the user say «ادامه» before this step runs? */
export function needsConfirm(step: PilotStep, safety: AutopilotConfig['safety']): boolean {
  const target = targetOf(step);
  return typeof target === 'string' && matchesAny(safety.confirmTargets, target);
}
