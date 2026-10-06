/**
 * Clinical copy groups whose rows only super admins may create or change (B-N5-09, admin-api.md §18; the API
 * answers 403 to editors). Mirrors backend registry.SuperOnlyGroups; `GET /messages` sends the live list as
 * `super_only_groups`, this is the fallback for screens that do not load the list.
 */
export const SUPER_ONLY_GROUPS: readonly string[] = ['postpartum_week_tip', 'postpartum_alert', 'postpartum_safety'];

/** Whether `role` may write rows of `group`. */
export function canWriteGroup(group: string, role: string | undefined, superOnly: readonly string[] = SUPER_ONLY_GROUPS): boolean {
  return role === 'super' || !superOnly.includes(group);
}
