'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useUserMode } from '@/entities/message';
import { usePlusStatus } from '@/entities/plus';
import { useUserProfile } from '@/entities/user';
import { useAppLock } from '@/features/app-lock';
import { useLogout } from '@/features/auth';
import { useExportData } from '@/features/manage-account';
import { useSwitchLocale } from '@/features/switch-locale';
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
} from '@/shared/ui';
import { BottomNav, useNavMode } from '@/widgets/bottom-nav';

import { MeErrorCard } from './MeErrorCard';
import { firstLetter, localizeDigits, maskMobile, MODE_TONE } from './me-format';

const APP_VERSION = '1.0.0';

/** A hub row whose feature ships with a later bloom task: visible, not tappable, «به‌زودی». */
interface SoonRow {
  key: 'companions' | 'children' | 'courses' | 'todo' | 'bookings' | 'orders' | 'devices' | 'chat';
  icon: IconName;
  tone: Tone;
}

const FAMILY_SOON: readonly SoonRow[] = [
  { key: 'companions', icon: 'users', tone: 'brand' }, // B-N4
  { key: 'children', icon: 'user', tone: 'bloom' }, // B-N5
];
const TASK_ROWS: readonly SoonRow[] = [
  { key: 'courses', icon: 'gradCap', tone: 'bloom' }, // B-N8-03
  { key: 'todo', icon: 'todo', tone: 'brand' }, // B-N6-08
  { key: 'bookings', icon: 'calendar', tone: 'data' }, // B-N7
  { key: 'orders', icon: 'box', tone: 'brand' }, // B-N10
];

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
  const isPlus = plusStatus ? plusStatus.isPlus : Boolean(userMode?.isPremium);
  const rtl = useDirection() === 'rtl';
  const logout = useLogout();
  const { exportData, isPending: exporting, isError: exportFailed } = useExportData();
  const { locale, languages } = useSwitchLocale();
  const preference = useThemeStore((s) => s.preference);
  const appLock = useAppLock();

  const profile = profileQuery.data;
  // Life-stage mode (B-N2-03) — menopause/teen/postpartum exist only there; falls back to /messages/mode.
  const navMode = useNavMode().mode ?? 'cycle';
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
          <HeaderButton
            label={t('rows.appearance')}
            icon="sun"
            variant="soft"
            onClick={() => router.push('/profile/appearance')}
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
            <button type="button" className="me-id-main" onClick={() => router.push('/profile/account')}>
              <span className="me-avatar" aria-hidden>
                {profile?.name ? firstLetter(profile.name) : <Icon name="user" size={26} />}
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
        {navMode === 'teen' ? null : (
          <button
            type="button"
            className="me-plus"
            onClick={() => router.push(isPlus ? '/plus/manage' : '/plus')}
          >
            <IconCircle icon="sparkle" tone="brand" size="md" />
            <span className="nb-row-text">
              <span className="nb-row-title">{isPlus ? t('plus.active') : t('plus.title')}</span>
              <span className="nb-row-desc">{isPlus ? t('plus.activeSub') : t('plus.upsellSub')}</span>
            </span>
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="nb-row-chev" />
          </button>
        )}

        <section className="me-sec" aria-labelledby="me-g-family">
          <SectionTitle id="me-g-family" title={t('groups.family')} />
          <ListGroup>
            <ListRow
              icon="modeRing"
              iconTone={MODE_TONE[navMode]}
              title={t('rows.mode')}
              description={t('rows.modeSub', { mode: modeName })}
              onClick={() => router.push('/profile/mode')}
            />
            {FAMILY_SOON.map((row) => (
              <ListRow key={row.key} icon={row.icon} iconTone={row.tone} title={t(`rows.${row.key}`)} trailing={soon} />
            ))}
          </ListGroup>
        </section>

        <section className="me-sec" aria-labelledby="me-g-tasks">
          <SectionTitle id="me-g-tasks" title={t('groups.tasks')} />
          <ListGroup>
            {TASK_ROWS.map((row) => (
              <ListRow key={row.key} icon={row.icon} iconTone={row.tone} title={t(`rows.${row.key}`)} trailing={soon} />
            ))}
          </ListGroup>
        </section>

        <section className="me-sec" aria-labelledby="me-g-data">
          <SectionTitle id="me-g-data" title={t('groups.data')} />
          <ListGroup>
            <ListRow icon="watch" iconTone="data" title={t('rows.devices')} trailing={soon} />
            {/* §11 — export is a first-class right: GET /profile/export as a JSON file. */}
            <ListRow
              icon="download"
              title={t('rows.backup')}
              description={exporting ? t('rows.exporting') : t('rows.backupSub')}
              onClick={() => (exporting ? undefined : exportData())}
            />
          </ListGroup>
          {exportFailed ? (
            <p className="me-inline-error" role="alert">
              {t('rows.exportError')}
            </p>
          ) : null}
        </section>

        <section className="me-sec" aria-labelledby="me-g-settings">
          <SectionTitle id="me-g-settings" title={t('groups.settings')} />
          <ListGroup>
            <ListRow
              icon="lock"
              iconTone="success"
              title={t('rows.privacy')}
              description={appLock?.enabled ? t('rows.privacyLockOn') : undefined}
              onClick={() => router.push('/profile/privacy')}
            />
            <ListRow
              icon="bell"
              title={t('rows.notifications')}
              onClick={() => router.push('/profile/notifications')}
            />
            {/* B-N1-09: cycle length + cycle reminders. */}
            <ListRow
              icon="drop"
              iconTone="period"
              title={t('rows.cycleSettings')}
              description={t('rows.cycleSettingsSub')}
              onClick={() => router.push('/cycle/settings')}
            />
            <ListRow
              icon="moon"
              title={t('rows.appearance')}
              description={mounted ? t(`theme.${preference}`) : undefined}
              onClick={() => router.push('/profile/appearance')}
            />
            <ListRow
              icon="calendar"
              title={t('rows.language')}
              description={`${languageName} · ${calendarName}`}
              onClick={() => router.push('/profile/language')}
            />
          </ListGroup>
        </section>

        <section className="me-sec" aria-labelledby="me-g-support">
          <SectionTitle id="me-g-support" title={t('groups.support')} />
          <ListGroup>
            <ListRow icon="help" title={t('rows.help')} onClick={() => router.push('/profile/support')} />
            <ListRow icon="chat" title={t('rows.chat')} onClick={() => router.push('/profile/support')} />
            <ListRow icon="info" iconTone="neutral" title={t('rows.about')} onClick={() => router.push('/profile/about')} />
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
