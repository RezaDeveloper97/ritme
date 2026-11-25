'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useAppModules } from '@/entities/app-module';
import { companionName, useCompanions } from '@/entities/companion';
import { useUserMode } from '@/entities/message';
import { formatCredits, useAiCredits } from '@/entities/ai-credit';
import { usePlusStatus } from '@/entities/plus';
import { primaryParentLink, useTeenToday } from '@/entities/teen';
import { useUserProfile } from '@/entities/user';
import { useAppLock } from '@/features/app-lock';
import { useLogout } from '@/features/auth';
import { useSwitchLocale } from '@/features/switch-locale';
import { pilot } from '@/shared/pilot';
import { APP_VERSION } from '@/shared/config';
import { localizeHref, useDirection, useRouter, type Locale } from '@/shared/i18n';
import { calendarSystem } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { useThemeStore } from '@/shared/theme';
import {
  HeaderButton,
  Icon,
  IconCircle,
  ListGroup,
  ListRow,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  type IconName,
  type Tone,
  Art3d,
} from '@/shared/ui';
import { WomanAvatar } from '@/shared/ui/illustrations';
import { BottomNav, useNavMode } from '@/widgets/bottom-nav';

import { MeErrorCard } from './MeErrorCard';
import { localizeDigits, maskMobile, MODE_TONE } from './me-format';
import '@/shared/ui/styles/me.css';
import '@/shared/ui/styles/tone.css';

/** A hub row whose feature ships with a later bloom task: visible, not tappable, «به‌زودی». */
interface SoonRow {
  key: 'companions' | 'children' | 'courses' | 'bookings' | 'orders' | 'devices' | 'chat';
  icon: IconName;
  tone: Tone;
}

const FAMILY_SOON: readonly SoonRow[] = [
  { key: 'children', icon: 'user', tone: 'bloom' }, // B-N5
];
// «سفارش‌ها» (B-N10) is live since CB-SHOP-08 — its row is rendered directly below.
const TASK_ROWS: readonly SoonRow[] = [];

/**
 * «من» hub (B-N1-10, `nbl_Me_Hub` / `nbd_Me_Hub`): profile header (masked
 * phone, mode, Plus status) and the grouped sections. Rows whose screens ship
 * with later bloom tasks show «به‌زودی» and are not buttons until they land.
 */
export function ProfilePage() {
  const t = useTranslations('me');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const mounted = useMounted();
  const profileQuery = useUserProfile();
  const { data: userMode } = useUserMode();
  const { data: plusStatus } = usePlusStatus();
  const { data: credits } = useAiCredits();
  const tc = useTranslations('aiCredit');
  const isPlus = plusStatus ? plusStatus.isPlus : Boolean(userMode?.isPremium);
  const rtl = useDirection() === 'rtl';
  const logout = useLogout();
  const { locale, languages } = useSwitchLocale();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  // The icon waits for mount so the server render can't disagree with the stored theme.
  const dark = mounted && theme === 'dark';
  const appLock = useAppLock();

  const profile = profileQuery.data;
  // Life-stage mode (B-N2-03) — menopause/teen/postpartum exist only there; falls back to /messages/mode.
  const navState = useNavMode();
  const navMode = navState.mode ?? 'cycle';
  // B-N4-04: «همدم‌ها و خانواده» → /companions; the row lists who is there («علی · مادر»).
  // CB-TEEN-04b: a teen's row opens /teen/parent, so it names her parent link — never a partner link
  // from before teen mode (dormant while she is a teen); the partner list waits until the mode is known.
  const teen = navMode === 'teen';
  // Admin module switches: the companions / children rows go with their module.
  const modules = useAppModules();
  const { data: companions } = useCompanions(!navState.pending && navMode !== 'companion' && !teen && modules.companion);
  const { data: teenToday } = useTeenToday(teen);
  const parentLink = teen ? primaryParentLink(teenToday?.parentLinks ?? []) : null;
  const companionNames = teen
    ? (parentLink?.displayName ?? '')
    : (companions ?? []).map((c) => companionName(c)).filter(Boolean).join(' · ');
  const modeName = t(`modes.${navMode}`);
  const languageName = languages.find((l) => l.code === locale)?.name ?? locale;
  const calendarName = t(`calendars.${calendarSystem(loc)}`);
  const soon = <StatusPill tone="neutral">{t('soon')}</StatusPill>;

  // Loading is decided only after mount: the profile query is gated on a
  // localStorage token, so the server always renders the skeleton and the first
  // client pass must match it (no hydration mismatch).
  const loading = !mounted || profileQuery.isLoading;

  // A full document replace: drops every React tree, cache and token of the
  // signed-in session; `onSettled` — a failed logout call still cleared it.
  const handleLogout = () => {
    if (logout.isPending) return;
    logout.mutate(undefined, {
      onSettled: () => window.location.replace(localizeHref('/signup', loc)),
    });
  };


  return (
    <div className="view me-page">
      <SkyLayer />
      <div className="scroll me-scroll">
        <header className="me-hdr">
          <h1 className="me-title">{t('title')}</h1>
          {/* One tap flips light/dark — there is no appearance screen. */}
          <HeaderButton
            label={dark ? t('themeToggle.toLight') : t('themeToggle.toDark')}
            icon={dark ? 'sun' : 'moon'}
            variant="soft"
            onClick={() => setTheme(dark ? 'light' : 'dark')}
            pilot={pilot('me.header:theme')}
          />
        </header>

        {loading ? (
          <SkeletonGroup label={t('loading')} className="me-id is-skel">
            <Skeleton shape="circle" />
            <span className="me-id-body">
              <Skeleton width="medium" />
              <Skeleton width="short" />
            </span>
          </SkeletonGroup>
        ) : profileQuery.isError ? (
          <MeErrorCard onRetry={() => profileQuery.refetch()} />
        ) : (
          <div className="me-id">
            <button type="button" className="me-id-main" onClick={() => router.push('/profile/account')} {...pilot('me.open:profile-account')}>
              <span className="me-avatar" aria-hidden>
                <WomanAvatar size={54} />
              </span>
              <span className="me-id-body">
                <span className="me-id-name">{profile?.name || t('guest')}</span>
                <bdi dir="ltr" className="me-id-phone">
                  {profile?.mobile ? maskMobile(profile.mobile, loc) : t('noPhone')}
                </bdi>
              </span>
            </button>
            <span className={`me-mode-pill tone-${MODE_TONE[navMode]}`} aria-label={t('modeLabel', { mode: modeName })}>
              <span className="me-mode-dot" aria-hidden />
              {modeName}
            </span>
          </div>
        )}

        {/* Plus status → /plus (paywall) or /plus/manage (B-N2-07).
            Teen mode never shows a Plus upsell (B-N2-03, gaps.md #3). */}
        {/* B-N4-05: a companion account has no cycle features to upsell either. */}
        {teen || navMode === 'companion' ? null : (
          <button
            type="button"
            className="me-plus"
            onClick={() => router.push(isPlus ? '/plus/manage' : '/plus')}
            {...pilot('me.open:plus')}
          >
            <IconCircle icon="sparkle" tone="brand" size="md" />
            <span className="nb-row-text">
              <span className="nb-row-title">{isPlus ? t('plus.active') : t('plus.title')}</span>
              <span className="nb-row-desc">{isPlus ? t('plus.activeSub') : t('plus.upsellSub')}</span>
            </span>
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="nb-row-chev" />
          </button>
        )}

        {/* AI credits at a glance → /profile/credits; hidden wherever the Plus card is. */}
        {teen || navMode === 'companion' ? null : (
          <button type="button" className="me-plus me-credits" onClick={() => router.push('/profile/credits')} {...pilot('me.open:profile-credits')}>
            <Art3d name="credits" size={40} />
            <span className="nb-row-text">
              <span className="nb-row-title">{tc('card.title')}</span>
              <span className="nb-row-desc">
                {!credits
                  ? tc('card.action')
                  : !credits.canUse
                    ? credits.extra.locked && credits.extra.remaining > 0
                      ? tc('card.locked', { count: formatCredits(credits.extra.remaining, loc) })
                      : tc('card.noSubscription')
                    : tc('card.left', { count: formatCredits(credits.spendable, loc) })}
              </span>
            </span>
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="nb-row-chev" />
          </button>
        )}

        <section className="me-sec" aria-labelledby="me-g-family">
          <SectionTitle id="me-g-family" title={t('groups.family')} />
          <ListGroup>
            {/* B-N4-05: a companion has no life stage to switch (PUT /profile/life-stage → 422);
                his row is «کد همدم» — link another partner or leave a link. */}
            {navMode === 'companion' ? (
              modules.companion ? (
                <ListRow
                  icon="users"
                  iconTone="brand"
                  title={t('rows.companionCode')}
                  description={t('rows.companionCodeSub')}
                  onClick={() => router.push('/companion/links')} pilot={pilot('me.open:companion-links')}
                />
              ) : null
            ) : (
              <ListRow
                icon="modeRing"
                iconTone={MODE_TONE[navMode]}
                title={t('rows.mode')}
                description={t('rows.modeSub', { mode: modeName })}
                onClick={() => router.push('/profile/mode')} pilot={pilot('me.open:profile-mode')}
              />
            )}
            {navMode !== 'companion' && (teen ? modules.teen : modules.companion) ? (
              <ListRow
                icon="users"
                iconTone="brand"
                title={t('rows.companions')}
                description={companionNames || undefined}
                // CB-TEEN-03: a teen shares with a parent only — her family row is «همراهی مادر».
                onClick={() => router.push(teen ? '/teen/parent' : '/companions')}
                pilot={pilot('me.open:companions')}
              />
            ) : null}
            {/* CB-NAV-03 (nbd_Nav_Me «فرزندان»): /children shipped with B-N5-05 — the row is live for an owner
                (a teen has no children row; a companion keeps the «به‌زودی» row). */}
            {navMode === 'companion'
              ? FAMILY_SOON.map((row) => (
                  <ListRow key={row.key} icon={row.icon} iconTone={row.tone} title={t(`rows.${row.key}`)} trailing={soon} />
                ))
              : teen || !modules.children
                ? null
                : FAMILY_SOON.map((row) => (
                    <ListRow
                      key={row.key}
                      icon={row.icon}
                      iconTone={row.tone}
                      title={t(`rows.${row.key}`)}
                      onClick={() => router.push('/children')} pilot={pilot('me.open:children')}
                    />
                  ))}
          </ListGroup>
        </section>

        {/* CB-TEEN-04b: «کارها و خریدها» (orders, bookings…) is a commercial surface — not for teens. */}
        {teen ? null : (
          <section className="me-sec" aria-labelledby="me-g-tasks">
            <SectionTitle id="me-g-tasks" title={t('groups.tasks')} />
            <ListGroup>
              {/* B-N6-08: «کارهای من» → /todo. */}
              {modules.todo ? (
                <ListRow icon="todo" iconTone="brand" title={t('rows.todo')} onClick={() => router.push('/todo')} pilot={pilot('me.open:todo')} />
              ) : null}
              {/* B-N8-03: «دوره‌های من» → /learn. */}
              {modules.learning && modules.my_courses ? (
                <ListRow icon="gradCap" iconTone="bloom" title={t('rows.courses')} onClick={() => router.push('/learn')} pilot={pilot('me.open:learn')} />
              ) : null}
              {/* B-N8-04: «دانلودها» → /learn/downloads (offline lessons). */}
              {modules.learning ? (
                <ListRow icon="download" iconTone="data" title={t('rows.downloads')} onClick={() => router.push('/learn/downloads')} pilot={pilot('me.open:learn-downloads')} />
              ) : null}
              {/* B-N10-05: «رزروها» → /profile/bookings (my city-service requests; CB-DIR-09 supersedes later). */}
              {modules.mother_child ? (
                <ListRow icon="calendar" iconTone="data" title={t('rows.bookings')} onClick={() => router.push('/profile/bookings')} pilot={pilot('me.open:profile-bookings')} />
              ) : null}
              {/* CB-SHOP-08: «سفارش‌ها» → /profile/orders (internal COD shop; supersedes the B-N10-03 partner row). */}
              {modules.shop ? (
                <ListRow icon="box" iconTone="brand" title={t('rows.orders')} onClick={() => router.push('/profile/orders')} pilot={pilot('me.open:profile-orders')} />
              ) : null}
              {TASK_ROWS.map((row) => (
                <ListRow key={row.key} icon={row.icon} iconTone={row.tone} title={t(`rows.${row.key}`)} trailing={soon} />
              ))}
            </ListGroup>
          </section>
        )}

        <section className="me-sec" aria-labelledby="me-g-data">
          <SectionTitle id="me-g-data" title={t('groups.data')} />
          <ListGroup>
            {/* The services hub left the bottom nav for «تحلیل»; «من» is its entry now. */}
            {navMode !== 'companion' ? (
              <ListRow
                icon="grid"
                iconTone="data"
                title={t('rows.services')}
                description={t('rows.servicesSub')}
                onClick={() => router.push('/services')} pilot={pilot('me.open:services')}
              />
            ) : null}
            {/* B-N6-02: Me entry into the vitals hub (also on «خدمات»). */}
            {navMode !== 'companion' ? (
              <ListRow
                icon="heartLine"
                iconTone="bloom"
                title={t('rows.vitals')}
                description={t('rows.vitalsSub')}
                onClick={() => router.push('/vitals')} pilot={pilot('me.open:vitals')}
              />
            ) : null}
            {/* B-N6-03: «پرونده سلامت من» (owner-only; a companion account has no record). */}
            {navMode !== 'companion' && modules.record ? (
              <ListRow
                icon="fileDoc"
                iconTone="brand"
                title={t('rows.healthRecord')}
                description={t('rows.healthRecordSub')}
                onClick={() => router.push('/record')} pilot={pilot('me.open:record')}
              />
            ) : null}
            {/* B-N10-07: honest «soon» devices screen (the web cannot pair watches). */}
            {modules.devices ? (
              <ListRow
                icon="watch"
                iconTone="data"
                title={t('rows.devices')}
                description={t('rows.devicesSub')}
                onClick={() => router.push('/profile/devices')} pilot={pilot('me.open:profile-devices')}
              />
            ) : null}
            {/* B-N10-07: last server sync + manual export (JSON/PDF/doctor report). */}
            <ListRow
              icon="download"
              title={t('rows.backup')}
              description={t('rows.backupSub')}
              onClick={() => router.push('/profile/backup')} pilot={pilot('me.open:profile-backup')}
            />
          </ListGroup>
        </section>

        <section className="me-sec" aria-labelledby="me-g-settings">
          <SectionTitle id="me-g-settings" title={t('groups.settings')} />
          <ListGroup>
            <ListRow
              icon="lock"
              iconTone="success"
              title={t('rows.privacy')}
              description={appLock?.enabled ? t('rows.privacyLockOn') : undefined}
              onClick={() => router.push('/profile/privacy')} pilot={pilot('me.open:profile-privacy')}
            />
            <ListRow
              icon="bell"
              title={t('rows.notifications')}
              onClick={() => router.push('/profile/notifications')} pilot={pilot('me.open:profile-notifications')}
            />
            {/* B-N1-09: cycle length + cycle reminders — not for a companion account (B-N4-05). */}
            {navMode !== 'companion' ? (
              <ListRow
                icon="drop"
                iconTone="period"
                title={t('rows.cycleSettings')}
                description={t('rows.cycleSettingsSub')}
                onClick={() => router.push('/cycle/settings')} pilot={pilot('me.open:cycle-settings')}
              />
            ) : null}
            <ListRow
              icon="calendar"
              title={t('rows.language')}
              description={`${languageName} · ${calendarName}`}
              onClick={() => router.push('/profile/language')} pilot={pilot('me.open:profile-language')}
            />
          </ListGroup>
        </section>

        <section className="me-sec" aria-labelledby="me-g-support">
          <SectionTitle id="me-g-support" title={t('groups.support')} />
          <ListGroup>
            <ListRow icon="help" title={t('rows.help')} onClick={() => router.push('/profile/support')} pilot={pilot('me.open:profile-support')} />
            <ListRow icon="chat" title={t('rows.chat')} onClick={() => router.push('/profile/support')} pilot={pilot('me.open:profile-support-chat')} />
            <ListRow icon="info" iconTone="neutral" title={t('rows.about')} onClick={() => router.push('/profile/about')} pilot={pilot('me.open:profile-about')} />
          </ListGroup>
        </section>

        <button type="button" className="me-logout" onClick={handleLogout} disabled={logout.isPending}>
          {logout.isPending ? t('loggingOut') : t('logout')}
        </button>
        <p className="me-version">{t('version', { version: localizeDigits(APP_VERSION, loc) })}</p>
      </div>
      <BottomNav />
    </div>
  );
}
