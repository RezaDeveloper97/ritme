import { can, type AdminRole, type Permissions } from '@/entities/admin';
import type { IconName } from '@/shared/ui';

/**
 * The sidebar: one titled, collapsible category per business area, so no screen hides inside another one
 * (the sub-screens that used to be reachable only from a parent page's buttons — message rules, shop tabs, city
 * cities/categories, insurers — have their own entry). `role: 'super'` items are hidden from editors and `cap`
 * items need that capability's `view` — the API enforces the same with 403 (admin-api.md §4, §21).
 */
export interface NavItem {
  key: string; // messages key under `nav`
  href: string;
  icon: IconName;
  role?: AdminRole;
  /** B-N9-02: the capability the screen reads (admin-api.md §21); hidden when the role lacks its `view`. */
  cap?: string;
  ready: boolean;
  devOnly?: boolean;
  /** Extra path prefixes that belong to this item (detail pages living outside its href). */
  also?: readonly string[];
}

export interface NavGroup {
  key: string; // messages key under `nav`
  icon: IconName;
  items: NavItem[];
}

export const NAV: readonly NavGroup[] = [
  {
    key: 'groupOverview',
    icon: 'dashboard',
    items: [
      { key: 'dashboard', href: '/', icon: 'dashboard', cap: 'dashboard', ready: true },
      { key: 'kpi', href: '/kpi', icon: 'sparkle', cap: 'dashboard', ready: true }, // B-N9-08
      { key: 'okr', href: '/okr', icon: 'flag', cap: 'dashboard', ready: true }, // B-N9-09
    ],
  },
  {
    key: 'groupUsers',
    icon: 'users',
    items: [
      { key: 'users', href: '/users', icon: 'users', cap: 'users', ready: true },
      { key: 'supportReports', href: '/support-reports', icon: 'inbox', cap: 'support', ready: true },
      { key: 'healthAccess', href: '/health-access', icon: 'key', cap: 'health', ready: true },
    ],
  },
  {
    // Every text the app shows from the message engine: the copy (by group), the v2 display rules, the simulator.
    key: 'groupMessages',
    icon: 'message',
    items: [
      { key: 'messages', href: '/messages', icon: 'message', cap: 'messages', ready: true },
      { key: 'messageRules', href: '/messages/rules', icon: 'listOrdered', cap: 'messages', ready: true },
      { key: 'messageEngine', href: '/messages/rules/engine', icon: 'gear', cap: 'messages', ready: true },
    ],
  },
  {
    key: 'groupContent',
    icon: 'article',
    items: [
      { key: 'articles', href: '/articles', icon: 'article', cap: 'content', ready: true },
      { key: 'affirmations', href: '/affirmations', icon: 'sparkle', cap: 'content', ready: true },
      { key: 'phaseContents', href: '/phase-contents', icon: 'moon', cap: 'content', ready: true },
      { key: 'recommendations', href: '/recommendations', icon: 'idea', cap: 'content', ready: true },
      { key: 'taskTemplates', href: '/task-templates', icon: 'task', cap: 'content', ready: true },
      { key: 'banners', href: '/banners', icon: 'banner', cap: 'content', ready: true },
      { key: 'infoSections', href: '/info-sections', icon: 'lifeRing', cap: 'content', ready: true },
      { key: 'catalog', href: '/catalog', icon: 'listBullet', cap: 'clinical', ready: true },
    ],
  },
  {
    key: 'groupChallenges',
    icon: 'flag',
    items: [
      { key: 'challenges', href: '/challenges', icon: 'flag', cap: 'challenges', ready: true },
      { key: 'challengeCompletions', href: '/challenge-completions', icon: 'check', cap: 'challenges', ready: true },
    ],
  },
  {
    // B-N9-10 / B-N9-11: urgent-signal aggregates, the clinical review queue and the versioned thresholds.
    key: 'groupClinical',
    icon: 'shieldCheck',
    items: [
      { key: 'safety', href: '/safety', icon: 'alert', cap: 'clinical', ready: true },
      { key: 'clinicalConfig', href: '/clinical-config', icon: 'gear', cap: 'config', ready: true },
      { key: 'checkupTypes', href: '/checkup-types', icon: 'shieldCheck', cap: 'clinical', ready: true },
    ],
  },
  {
    // The alert rules are inbox rules now (T-M10-13): /pregnancy-alert-rules forwards to /messages/rules?placement=alerts_inbox.
    key: 'groupPregnancy',
    icon: 'baby',
    items: [
      { key: 'pregnancyWeeks', href: '/pregnancy-weeks', icon: 'baby', cap: 'clinical', ready: true },
      { key: 'pregnancyCarePlan', href: '/pregnancy-care-plan', icon: 'calendar', cap: 'clinical', ready: true },
    ],
  },
  {
    // «کودک و پس از زایمان» (B-N5-09): clinical content — every admin reads, super admins write (API 403 + hidden buttons).
    key: 'groupChildren',
    icon: 'heart',
    items: [
      { key: 'childVaccines', href: '/children-content/vaccines', icon: 'shieldCheck', cap: 'clinical', ready: true },
      { key: 'childMilestones', href: '/children-content/milestones', icon: 'flag', cap: 'clinical', ready: true },
      { key: 'childLearn', href: '/children-content/learn', icon: 'article', cap: 'clinical', ready: true },
      { key: 'childWho', href: '/children-content/who', icon: 'listOrdered', cap: 'clinical', ready: true },
      { key: 'postpartumContent', href: '/postpartum-content', icon: 'heart', cap: 'clinical', ready: true },
    ],
  },
  {
    // «همدم» (B-N4-07): tips are content (editor + super); the links overview is masked and read-only.
    key: 'groupCompanions',
    icon: 'link',
    items: [
      { key: 'companionTips', href: '/companions/tips', icon: 'heart', cap: 'content', ready: true },
      { key: 'companionLinks', href: '/companions/links', icon: 'link', cap: 'content', ready: true },
    ],
  },
  {
    // «پزشکان و تله‌مدیسین» (B-N7-08): directory + bookings for every admin; refunds are super-only; the inbox answers
    // 403 doctor_required for admins not linked to a doctor.
    key: 'groupTelemed',
    icon: 'stetho',
    items: [
      { key: 'telemedDoctors', href: '/telemed/doctors', icon: 'users', cap: 'telemed', ready: true },
      { key: 'telemedBookings', href: '/telemed/bookings', icon: 'calendar', cap: 'telemed', ready: true },
      { key: 'telemedReviews', href: '/telemed/reviews', icon: 'heart', cap: 'telemed', ready: true },
      { key: 'telemedInbox', href: '/telemed/inbox', icon: 'message', cap: 'telemed_doctor', ready: true },
    ],
  },
  {
    // B-N10-04: the home lab request queue and the orderable services.
    key: 'groupLab',
    icon: 'flask',
    items: [
      { key: 'labRequests', href: '/lab-requests', icon: 'inbox', cap: 'support', ready: true },
      { key: 'labServices', href: '/lab-services', icon: 'listBullet', cap: 'support', ready: true },
    ],
  },
  {
    // «مدرسین و دوره‌ها» (B-N8-08): every admin reads and reviews content; approve / revoke instructors is super-only.
    key: 'groupLearning',
    icon: 'article',
    items: [
      { key: 'learningInstructors', href: '/learning/instructors', icon: 'users', cap: 'learning', ready: true },
      { key: 'learningCourses', href: '/learning/courses', icon: 'article', cap: 'learning', ready: true },
      { key: 'learningReviews', href: '/learning/reviews', icon: 'shieldCheck', cap: 'learning', ready: true },
    ],
  },
  {
    // B-N10-03 + CB-SHOP-05: the catalog is «content»; the order queue and returns hold recipient PII → «support».
    key: 'groupShop',
    icon: 'tag',
    items: [
      { key: 'shop', href: '/shop', icon: 'tag', cap: 'content', ready: true },
      { key: 'shopCategories', href: '/shop/categories', icon: 'listBullet', cap: 'content', ready: true },
      { key: 'shopSellers', href: '/shop/sellers', icon: 'users', cap: 'content', ready: true },
      { key: 'shopStock', href: '/shop/stock', icon: 'inbox', cap: 'content', ready: true },
      { key: 'shopSlots', href: '/shop/slots', icon: 'calendar', cap: 'content', ready: true },
      { key: 'shopReviews', href: '/shop/reviews', icon: 'heart', cap: 'content', ready: true },
      { key: 'shopDiscounts', href: '/shop/discount-codes', icon: 'tag', cap: 'content', ready: true },
      { key: 'shopLayette', href: '/shop/layette', icon: 'baby', cap: 'content', ready: true },
      { key: 'shopOrders', href: '/shop/shipments', icon: 'inbox', cap: 'support', ready: true, also: ['/shop/orders'] },
      { key: 'shopReturns', href: '/shop/returns', icon: 'undo', cap: 'support', ready: true },
    ],
  },
  {
    // B-N10-05 / CB-DIR-05 city services directory, CB-INS-02 insurers + centres.
    key: 'groupDirectory',
    icon: 'globe',
    items: [
      { key: 'cityServices', href: '/city-services', icon: 'banner', cap: 'content', ready: true },
      { key: 'cityServiceBookings', href: '/city-service-bookings', icon: 'inbox', cap: 'support', ready: true },
      { key: 'cityServiceApplications', href: '/city-services/applications', icon: 'inbox', cap: 'content', ready: true },
      { key: 'cityServiceReviews', href: '/city-services/reviews', icon: 'heart', cap: 'content', ready: true },
      { key: 'cityServiceReports', href: '/city-services/reports', icon: 'alert', cap: 'content', ready: true },
      { key: 'cityServiceCities', href: '/city-services/cities', icon: 'globe', cap: 'content', ready: true },
      { key: 'cityServiceCategories', href: '/city-services/categories', icon: 'listBullet', cap: 'content', ready: true },
      { key: 'insuranceCenters', href: '/insurance-centers', icon: 'shieldCheck', cap: 'content', ready: true },
      { key: 'insurers', href: '/insurance-centers/insurers', icon: 'shield', cap: 'content', ready: true },
    ],
  },
  {
    // «اشتراک‌ها و پرداخت» (B-N2-09): every admin reads; writes are super-only (API 403 + hidden buttons).
    key: 'groupPlus',
    icon: 'crown',
    items: [
      { key: 'plusSubscriptions', href: '/plus/subscriptions', icon: 'users', cap: 'billing', ready: true },
      { key: 'plusPayments', href: '/plus/payments', icon: 'card', cap: 'billing', ready: true },
      { key: 'plusPlans', href: '/plus/plans', icon: 'crown', cap: 'billing', ready: true },
      { key: 'plusDiscounts', href: '/plus/discount-codes', icon: 'tag', cap: 'billing', ready: true },
      { key: 'plusSettings', href: '/plus/settings', icon: 'gear', cap: 'billing', ready: true },
    ],
  },
  {
    // «هوش مصنوعی و اعتبار» (docs/ai-credits.md, admin-api.md §29): the catalogue and slots are config, the credit packs
    // are billing; every write is super-only (API 403 + hidden buttons).
    key: 'groupAi',
    icon: 'sparkle',
    items: [
      { key: 'aiFeatures', href: '/ai/features', icon: 'sparkle', cap: 'config', ready: true },
      { key: 'aiModels', href: '/ai/models', icon: 'gear', cap: 'config', ready: true },
      { key: 'aiCreditPacks', href: '/ai/credit-packs', icon: 'tag', cap: 'billing', ready: true },
      { key: 'aiUsage', href: '/ai/usage', icon: 'listOrdered', cap: 'config', ready: true },
      { key: 'assistantDepartments', href: '/assistant/departments', icon: 'idea', role: 'super', ready: true },
    ],
  },
  {
    key: 'groupSystem',
    icon: 'gear',
    items: [
      // App module kill-switches (companion, TTC, IVF, menopause, teen, postpartum, children, contraception).
      { key: 'appModules', href: '/app-modules', icon: 'gear', cap: 'config', ready: true },
      { key: 'admins', href: '/admins', icon: 'shield', role: 'super', ready: true },
      // B-N9-02 «نقش‌ها و ممیزی»: the matrix (super), the audit log.
      { key: 'roles', href: '/roles', icon: 'shieldCheck', role: 'super', ready: true },
      { key: 'auditLog', href: '/audit-logs', icon: 'listOrdered', cap: 'audit', ready: true },
      { key: 'languages', href: '/languages', icon: 'language', role: 'super', ready: true },
      { key: 'uiKit', href: '/ui-kit', icon: 'sparkle', ready: true, devOnly: true },
    ],
  },
  {
    key: 'groupAccount',
    icon: 'key',
    items: [{ key: 'password', href: '/account/password', icon: 'key', ready: true }],
  },
];

/**
 * Groups visible to `role`; empty groups are dropped. With `permissions` (B-N9-02) an item with a `cap` also needs
 * that capability's `view`; without them only `role` gates (super / editor see what they always saw).
 */
export function navFor(role: AdminRole, opts: { dev: boolean; permissions?: Permissions }): NavGroup[] {
  const capOk = (item: NavItem) =>
    !item.cap || (opts.permissions ? can(role, opts.permissions, item.cap) : role === 'super' || item.cap !== 'health');
  return NAV.map((group) => ({
    ...group,
    items: group.items.filter(
      (item) =>
        (!item.role || item.role === role || role === 'super') && (!item.devOnly || opts.dev) && capOk(item),
    ),
  })).filter((group) => group.items.length > 0);
}

const under = (pathname: string, prefix: string) =>
  prefix === '/' ? pathname === '/' : pathname === prefix || pathname.startsWith(`${prefix}/`);

/** The item whose section contains `pathname` (longest matching prefix wins; `/` only exact). */
export function activeItem(pathname: string, groups: readonly NavGroup[] = NAV): NavItem | null {
  let best: NavItem | null = null;
  let bestLen = -1;
  for (const item of groups.flatMap((g) => g.items)) {
    for (const prefix of [item.href, ...(item.also ?? [])]) {
      if (under(pathname, prefix) && prefix.length > bestLen) {
        best = item;
        bestLen = prefix.length;
      }
    }
  }
  return best;
}

/** The group that holds `item` (for the header breadcrumb and the sidebar's open category). */
export function groupOf(item: NavItem | null, groups: readonly NavGroup[] = NAV): NavGroup | null {
  if (!item) return null;
  return groups.find((g) => g.items.some((i) => i.key === item.key)) ?? null;
}
