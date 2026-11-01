import { can, type AdminRole, type Permissions } from '@/entities/admin';
import type { IconName } from '@/shared/ui';

/**
 * The sidebar, mirroring the Blade panel's sections (backend/resources/views/
 * admin/layouts/app.blade.php). `ready: true` items render disabled until
 * their screen lands (T-M2-23 flips them). `role: 'super'` items are hidden
 * from editors — the API enforces the same with 403 (admin-api.md §4).
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
}

export interface NavGroup {
  key: string; // messages key under `nav`
  /** Modules the Night & Bloom admin design does not draw sit under «سایر ماژول‌ها». */
  other?: boolean;
  items: NavItem[];
}

export const NAV: readonly NavGroup[] = [
  {
    // The design's main list (B-N9-01): dashboard, users, messages, challenges, roles (super-only).
    key: 'groupMain',
    items: [
      { key: 'dashboard', href: '/', icon: 'dashboard', cap: 'dashboard', ready: true },
      { key: 'users', href: '/users', icon: 'users', cap: 'users', ready: true },
      { key: 'messages', href: '/messages', icon: 'message', cap: 'messages', ready: true },
      { key: 'kpi', href: '/kpi', icon: 'sparkle', cap: 'dashboard', ready: true }, // B-N9-08
      { key: 'okr', href: '/okr', icon: 'flag', cap: 'dashboard', ready: true }, // B-N9-09
      { key: 'challenges', href: '/challenges', icon: 'flag', cap: 'challenges', ready: true },
      // B-N9-10 «ایمنی و محتوای بالینی»: urgent-signal aggregates + the clinical review queue (clinical capability).
      { key: 'safety', href: '/safety', icon: 'alert', cap: 'clinical', ready: true },
      // B-N9-11 «آستانه‌ها و پیکربندی»: versioned clinical thresholds (config capability).
      { key: 'clinicalConfig', href: '/clinical-config', icon: 'gear', cap: 'config', ready: true },
      // App module kill-switches (companion, TTC, IVF, menopause, teen, postpartum, children, contraception).
      { key: 'appModules', href: '/app-modules', icon: 'gear', cap: 'config', ready: true },
      { key: 'admins', href: '/admins', icon: 'shield', role: 'super', ready: true },
      // B-N9-02 «نقش‌ها و ممیزی»: the matrix (super), the audit log and temporary health-data access.
      { key: 'roles', href: '/roles', icon: 'shieldCheck', role: 'super', ready: true },
      { key: 'auditLog', href: '/audit-logs', icon: 'listOrdered', cap: 'audit', ready: true },
      { key: 'healthAccess', href: '/health-access', icon: 'key', cap: 'health', ready: true },
    ],
  },
  {
    key: 'groupGeneral',
    other: true,
    items: [
      // Any active admin, like /users (B-N1-12b).
      { key: 'supportReports', href: '/support-reports', icon: 'inbox', cap: 'support', ready: true },
    ],
  },
  {
    // «اشتراک‌ها و پرداخت» (B-N2-09): every admin reads; writes are super-only (API 403 + hidden buttons).
    key: 'groupPlus',
    other: true,
    items: [
      { key: 'plusSubscriptions', href: '/plus/subscriptions', icon: 'users', cap: 'billing', ready: true },
      { key: 'plusPayments', href: '/plus/payments', icon: 'card', cap: 'billing', ready: true },
      { key: 'plusPlans', href: '/plus/plans', icon: 'crown', cap: 'billing', ready: true },
      { key: 'plusDiscounts', href: '/plus/discount-codes', icon: 'tag', cap: 'billing', ready: true },
      { key: 'plusSettings', href: '/plus/settings', icon: 'gear', cap: 'billing', ready: true },
    ],
  },
  {
    // «همدم» (B-N4-07): tips are content (editor + super); the links overview is masked and read-only.
    key: 'groupCompanions',
    other: true,
    items: [
      { key: 'companionTips', href: '/companions/tips', icon: 'heart', cap: 'content', ready: true },
      { key: 'companionLinks', href: '/companions/links', icon: 'link', cap: 'content', ready: true },
    ],
  },
  {
    // «مدرسین و دوره‌ها» (B-N8-08): every admin reads and reviews content; approve / revoke instructors is super-only.
    key: 'groupLearning',
    other: true,
    items: [
      { key: 'learningInstructors', href: '/learning/instructors', icon: 'users', cap: 'learning', ready: true },
      { key: 'learningCourses', href: '/learning/courses', icon: 'article', cap: 'learning', ready: true },
      { key: 'learningReviews', href: '/learning/reviews', icon: 'shieldCheck', cap: 'learning', ready: true },
    ],
  },
  {
    // «پزشکان و تله‌مدیسین» (B-N7-08): directory + bookings for every admin; refunds and the department editor are
    // super-only; the inbox answers 403 doctor_required for admins not linked to a doctor.
    key: 'groupTelemed',
    other: true,
    items: [
      { key: 'telemedDoctors', href: '/telemed/doctors', icon: 'users', cap: 'telemed', ready: true },
      { key: 'telemedBookings', href: '/telemed/bookings', icon: 'calendar', cap: 'telemed', ready: true },
      { key: 'telemedReviews', href: '/telemed/reviews', icon: 'heart', cap: 'telemed', ready: true },
      { key: 'telemedInbox', href: '/telemed/inbox', icon: 'message', cap: 'telemed_doctor', ready: true },
      { key: 'assistantDepartments', href: '/assistant/departments', icon: 'idea', role: 'super', ready: true },
      { key: 'labRequests', href: '/lab-requests', icon: 'inbox', cap: 'support', ready: true }, // B-N10-04 home lab request queue
      { key: 'labServices', href: '/lab-services', icon: 'listBullet', cap: 'support', ready: true }, // B-N10-04 orderable services
    ],
  },
  {
    key: 'groupContent',
    other: true,
    items: [
      { key: 'articles', href: '/articles', icon: 'article', cap: 'content', ready: true },
      { key: 'affirmations', href: '/affirmations', icon: 'sparkle', cap: 'content', ready: true },
      { key: 'challengeCompletions', href: '/challenge-completions', icon: 'check', cap: 'challenges', ready: true },
      { key: 'taskTemplates', href: '/task-templates', icon: 'task', cap: 'content', ready: true },
      { key: 'phaseContents', href: '/phase-contents', icon: 'moon', cap: 'content', ready: true },
      { key: 'recommendations', href: '/recommendations', icon: 'idea', cap: 'content', ready: true },
      { key: 'checkupTypes', href: '/checkup-types', icon: 'shieldCheck', cap: 'clinical', ready: true },
      { key: 'banners', href: '/banners', icon: 'banner', cap: 'content', ready: true },
      { key: 'shop', href: '/shop', icon: 'banner', cap: 'content', ready: true }, // B-N10-03 link-out catalog
      // CB-SHOP-05: the internal shop's order queue + returns (recipient PII → support) and the layette template.
      { key: 'shopOrders', href: '/shop/shipments', icon: 'inbox', cap: 'support', ready: true },
      { key: 'shopReturns', href: '/shop/returns', icon: 'undo', cap: 'support', ready: true },
      { key: 'shopLayette', href: '/shop/layette', icon: 'baby', cap: 'content', ready: true },
      { key: 'cityServices', href: '/city-services', icon: 'banner', cap: 'content', ready: true }, // B-N10-05 city services listings
      { key: 'cityServiceBookings', href: '/city-service-bookings', icon: 'inbox', cap: 'support', ready: true }, // B-N10-05 booking request queue
      { key: 'cityServiceApplications', href: '/city-services/applications', icon: 'inbox', cap: 'content', ready: true }, // CB-DIR-05 join review queue
      { key: 'cityServiceReviews', href: '/city-services/reviews', icon: 'heart', cap: 'content', ready: true }, // CB-DIR-05 review moderation
      { key: 'cityServiceReports', href: '/city-services/reports', icon: 'alert', cap: 'content', ready: true }, // CB-DIR-05 wrong-information reports
      { key: 'insuranceCenters', href: '/insurance-centers', icon: 'shieldCheck', cap: 'content', ready: true }, // CB-INS-02 insurers + centres
      { key: 'infoSections', href: '/info-sections', icon: 'lifeRing', cap: 'content', ready: true },
      // Editor + super (the content permission); switch to B-N9-02's content role once it lands.
      { key: 'catalog', href: '/catalog', icon: 'listBullet', cap: 'clinical', ready: true },
    ],
  },
  {
    key: 'groupPregnancy',
    other: true,
    items: [
      { key: 'pregnancyWeeks', href: '/pregnancy-weeks', icon: 'baby', cap: 'clinical', ready: true },
      { key: 'pregnancyCarePlan', href: '/pregnancy-care-plan', icon: 'calendar', cap: 'clinical', ready: true },
      // The alert rules are inbox rules now (T-M10-13): /pregnancy-alert-rules forwards to /messages/rules?placement=alerts_inbox.
    ],
  },
  {
    // «کودک و پس از زایمان» (B-N5-09): clinical content — every admin reads, super admins write (API 403 + hidden buttons).
    key: 'groupChildren',
    other: true,
    items: [
      { key: 'childVaccines', href: '/children-content/vaccines', icon: 'shieldCheck', cap: 'clinical', ready: true },
      { key: 'childMilestones', href: '/children-content/milestones', icon: 'flag', cap: 'clinical', ready: true },
      { key: 'childLearn', href: '/children-content/learn', icon: 'article', cap: 'clinical', ready: true },
      { key: 'childWho', href: '/children-content/who', icon: 'listOrdered', cap: 'clinical', ready: true },
      { key: 'postpartumContent', href: '/postpartum-content', icon: 'heart', cap: 'clinical', ready: true },
    ],
  },
  {
    key: 'groupSettings',
    other: true,
    items: [
      { key: 'languages', href: '/languages', icon: 'language', role: 'super', ready: true },
      { key: 'uiKit', href: '/ui-kit', icon: 'sparkle', ready: true, devOnly: true },
    ],
  },
  {
    key: 'groupAccount',
    other: true,
    items: [
      { key: 'password', href: '/account/password', icon: 'key', ready: true },
    ],
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

/** The item whose section contains `pathname` (longest href wins; `/` only exact). */
export function activeItem(pathname: string, groups: readonly NavGroup[] = NAV): NavItem | null {
  let best: NavItem | null = null;
  for (const item of groups.flatMap((g) => g.items)) {
    const match =
      item.href === '/' ? pathname === '/' : pathname === item.href || pathname.startsWith(`${item.href}/`);
    if (match && (!best || item.href.length > best.href.length)) best = item;
  }
  return best;
}
