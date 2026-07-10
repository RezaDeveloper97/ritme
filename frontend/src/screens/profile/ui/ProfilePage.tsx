'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type CSSProperties, type ReactNode, useState } from 'react';

import { useUserMode } from '@/entities/message';
import { useDeactivatePregnancy } from '@/entities/pregnancy';
import { useUserProfile } from '@/entities/user';
import { useLogout } from '@/features/auth';
import { DeleteAccountConfirm, useExportData } from '@/features/manage-account';
import { useSwitchLocale } from '@/features/switch-locale';
import { formatJalali } from '@/shared/lib/date';
import { type Locale, useRouter } from '@/shared/i18n';
import { Icon, type IconName } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

const APP_VERSION = '1.0.0';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (value: string | number, loc: Locale) =>
  loc === 'fa' ? String(value).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(value);

// ── A single settings row ─────────────────────────────────────
// Presentational only. Rows that lead somewhere pass `onClick`; the rest are
// static list items (rendered as <div>, not fake buttons) until their
// destination screens exist.
function Row({
  icon,
  label,
  trailing,
  danger,
  onClick,
  disabled,
}: {
  icon: IconName;
  label: string;
  trailing?: ReactNode;
  danger?: boolean;
  onClick?: () => void;
  disabled?: boolean;
}) {
  const color = danger ? '#E5484D' : 'var(--ink)';
  const inner = (
    <>
      <span
        style={{
          width: 34,
          height: 34,
          borderRadius: 11,
          flexShrink: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: danger ? 'rgba(229,72,77,.1)' : 'var(--line)',
          color: danger ? '#E5484D' : 'var(--brand)',
        }}
      >
        <Icon name={icon} size={19} />
      </span>
      <span style={{ flex: 1, fontSize: 14, fontWeight: 600, color, textAlign: 'start' }}>
        {label}
      </span>
      {trailing}
    </>
  );

  const style: CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    width: '100%',
    padding: '13px 14px',
    background: 'transparent',
    border: 0,
    font: 'inherit',
    textAlign: 'start',
    cursor: onClick && !disabled ? 'pointer' : 'default',
    opacity: disabled ? 0.55 : 1,
  };

  if (onClick) {
    return (
      <button type="button" style={style} onClick={onClick} disabled={disabled}>
        {inner}
      </button>
    );
  }
  return <div style={style}>{inner}</div>;
}

// ── A grouped card of rows ────────────────────────────────────
function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section style={{ padding: '0 18px', marginTop: 20 }}>
      <div style={{ fontSize: 13, fontWeight: 800, color: 'var(--muted)', margin: '0 4px 8px' }}>
        {title}
      </div>
      <div className="card" style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {children}
      </div>
    </section>
  );
}

// Hairline between rows, inset past the icon (logical start — RTL-safe).
function Divider() {
  return <div style={{ height: 1, background: 'var(--line)', marginInlineStart: 60 }} />;
}

// Direction-aware disclosure chevron: points "into" the row (§12 — logical, not
// hardcoded left/right). RTL forward = left, LTR forward = right.
function Chevron({ loc }: { loc: Locale }) {
  return (
    <Icon
      name={loc === 'fa' ? 'chevronLeft' : 'chevronRight'}
      size={18}
      style={{ color: 'var(--muted)', flexShrink: 0 }}
    />
  );
}

// Right-aligned value shown at the end of a stat row. Muted so the label
// (in ink) stays dominant; `--` while the profile is still loading.
function StatValue({ children }: { children: ReactNode }) {
  return (
    <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--muted)', textAlign: 'end' }}>
      {children}
    </span>
  );
}

export function ProfilePage() {
  const t = useTranslations('profile');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { data: profile } = useUserProfile();
  const { data: userMode } = useUserMode();
  const deactivatePregnancy = useDeactivatePregnancy();
  const logout = useLogout();
  const { locale, switchLocale, isPending: switching } = useSwitchLocale();
  const { exportData, isPending: exporting } = useExportData();
  const [deleteOpen, setDeleteOpen] = useState(false);

  const user = profile;
  const health = profile?.health;
  const bmi = profile?.bmi;

  // These fallbacks must depend ONLY on the fetched value — never on loading
  // state. `useUserProfile` is gated on `isAuthenticated()` (which reads
  // localStorage), so the server always sees "logged out" while the client
  // sees "loading"; branching on that would cause a hydration mismatch. On both
  // the server and the first client render `health` is undefined, so both emit
  // the "not set" label, then the client re-renders once data arrives.
  // Dates cross the API boundary in Gregorian ISO and are only ever shown as
  // Jalali (§7), converted here via the shared date layer.
  const jalaliOrEmpty = (iso: string | null | undefined) =>
    iso ? formatJalali(new Date(iso), loc) : t('health.empty');
  // A count of days, formatted through ICU so digits localize per locale (§6).
  const daysOrEmpty = (value: number | null | undefined) =>
    value != null ? t('health.days', { days: value }) : t('health.empty');
  const measureOrEmpty = (key: 'kg' | 'cm', value: number | null | undefined) =>
    value != null ? t(`health.${key}`, { value }) : t('health.empty');

  const handleLogout = () => {
    if (logout.isPending) return;
    logout.mutate(undefined, {
      onSuccess: () => router.replace('/signup'),
    });
  };

  // Two locales: tapping the language row flips to the other one.
  const toggleLocale = () => switchLocale(locale === 'fa' ? 'en' : 'fa');

  // App mode (CLAUDE.md §1). Entering pregnancy mode routes to /pregnancy,
  // which activates and runs onboarding via its own gate; leaving it flips the
  // backend mode back to cycle in place.
  const isPregnancy = userMode?.mode === 'pregnancy';
  const switchToCycle = () => {
    if (deactivatePregnancy.isPending) return;
    deactivatePregnancy.mutate();
  };

  const chevron = <Chevron loc={loc} />;

  return (
    <div className="view" style={{ background: 'var(--page)' }}>

      <div className="scroll" style={{ paddingBottom: 24 }}>
        <div style={{ padding: '8px 20px 4px' }}>
          <div className="titr">{t('title')}</div>
        </div>

        {/* Identity card */}
        <section style={{ padding: '0 18px', marginTop: 12 }}>
          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 14, padding: 16 }}>
            <span
              style={{
                width: 60,
                height: 60,
                borderRadius: '50%',
                flexShrink: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'var(--pink-soft)',
                color: 'var(--brand)',
              }}
            >
              <Icon name="user" size={30} />
            </span>
            <div style={{ flex: 1, minWidth: 0, textAlign: 'start' }}>
              <div
                style={{
                  fontSize: 16,
                  fontWeight: 800,
                  color: 'var(--ink)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {user?.name ?? t('guest')}
              </div>
              <div
                style={{
                  fontSize: 13,
                  color: 'var(--muted)',
                  marginTop: 3,
                  direction: 'ltr',
                  textAlign: 'start',
                }}
              >
                {user?.mobile ? localizeNum(user.mobile, loc) : t('noPhone')}
              </div>
            </div>
            <button
              className="iconbtn"
              style={{ color: 'var(--brand)', flexShrink: 0 }}
              aria-label={t('editProfile')}
              onClick={() => router.push('/profile/personal')}
            >
              <Icon name="pencil" size={18} />
            </button>
          </div>
        </section>

        {/* App mode — the visible entry point for switching between cycle and
            pregnancy mode (§1). In cycle mode this is how the user starts
            pregnancy tracking; in pregnancy mode it links to the tracker and
            offers a way back. */}
        <Group title={t('sections.mode')}>
          {isPregnancy ? (
            <>
              <Row
                icon="heart"
                label={t('mode.pregnancyTracker')}
                trailing={chevron}
                onClick={() => router.push('/pregnancy')}
              />
              <Divider />
              <Row
                icon="refresh"
                label={deactivatePregnancy.isPending ? t('mode.switching') : t('mode.switchToCycle')}
                onClick={switchToCycle}
                disabled={deactivatePregnancy.isPending}
              />
            </>
          ) : (
            <Row
              icon="heart"
              label={t('mode.switchToPregnancy')}
              trailing={chevron}
              onClick={() => router.push('/pregnancy')}
            />
          )}
        </Group>

        {/* Cycle & health — the user's profile data from GET /profile. When the
            account has no health profile yet, a single hint row stands in for
            the stats. */}
        <Group title={t('health.title')}>
          {profile && !health ? (
            <Row icon="heart" label={t('health.notSetUp')} />
          ) : (
            <>
              <Row
                icon="refresh"
                label={t('health.cycleDuration')}
                trailing={<StatValue>{daysOrEmpty(health?.cycleDuration)}</StatValue>}
              />
              <Divider />
              <Row
                icon="drop"
                label={t('health.periodDuration')}
                trailing={<StatValue>{daysOrEmpty(health?.periodDuration)}</StatValue>}
              />
              <Divider />
              <Row
                icon="calendar"
                label={t('health.lastPeriod')}
                trailing={<StatValue>{jalaliOrEmpty(health?.lastPeriodStart)}</StatValue>}
              />
              <Divider />
              <Row
                icon="sparkle"
                label={t('health.birthday')}
                trailing={<StatValue>{jalaliOrEmpty(health?.birthday)}</StatValue>}
              />
              <Divider />
              <Row
                icon="chart"
                label={t('health.weight')}
                trailing={<StatValue>{measureOrEmpty('kg', health?.weight)}</StatValue>}
              />
              <Divider />
              <Row
                icon="walk"
                label={t('health.height')}
                trailing={<StatValue>{measureOrEmpty('cm', health?.height)}</StatValue>}
              />
              {/* BMI is server-computed from weight+height; the row is present
                  only once both are set (§11 — shown, never logged). */}
              {bmi ? (
                <>
                  <Divider />
                  <Row
                    icon="chart"
                    label={t('health.bmi')}
                    trailing={
                      <StatValue>
                        {t('health.bmiValue', {
                          value: localizeNum(bmi.value, loc),
                          category: bmi.categoryLabel,
                        })}
                      </StatValue>
                    }
                  />
                </>
              ) : null}
            </>
          )}
        </Group>

        {/* Supportive, band-specific BMI message. The backend owns this copy
            (admin-editable) and returns it already localized; we only render
            it. Not medical advice (§11). */}
        {bmi ? (
          <section style={{ padding: '0 18px', marginTop: 14 }}>
            <div className="card" style={{ padding: 16 }}>
              <p
                style={{
                  fontSize: 13.5,
                  lineHeight: 2,
                  color: 'var(--muted)',
                  margin: 0,
                  textAlign: 'start',
                }}
              >
                {bmi.message}
              </p>
            </div>
          </section>
        ) : null}

        {/* Account */}
        <Group title={t('sections.account')}>
          <Row icon="user" label={t('rows.personalInfo')} trailing={chevron} onClick={() => router.push('/profile/personal')} />
          <Divider />
          <Row icon="heart" label={t('rows.healthPrefs')} trailing={chevron} onClick={() => router.push('/profile/health')} />
          <Divider />
          <Row icon="alarm" label={t('rows.reminders')} trailing={chevron} onClick={() => router.push('/profile/reminders')} />
          <Divider />
          <Row icon="bell" label={t('rows.notifications')} trailing={chevron} onClick={() => router.push('/profile/notifications')} />
        </Group>

        {/* App */}
        <Group title={t('sections.app')}>
          <Row
            icon="globe"
            label={t('rows.language')}
            onClick={toggleLocale}
            disabled={switching}
            trailing={
              <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--brand)' }}>
                  {t(`language.${locale}`)}
                </span>
                {chevron}
              </span>
            }
          />
        </Group>

        {/* Privacy & data (§11 — export & delete are first-class) */}
        <Group title={t('sections.privacy')}>
          <Row icon="shield" label={t('rows.privacyPolicy')} trailing={chevron} onClick={() => router.push('/profile/info/privacy')} />
          <Divider />
          <Row
            icon="download"
            label={exporting ? t('exporting') : t('rows.exportData')}
            trailing={chevron}
            onClick={() => exportData()}
            disabled={exporting}
          />
          <Divider />
          <Row icon="trash" label={t('rows.deleteAccount')} danger trailing={chevron} onClick={() => setDeleteOpen(true)} />
        </Group>

        {/* Support */}
        <Group title={t('sections.support')}>
          <Row icon="info" label={t('rows.help')} trailing={chevron} onClick={() => router.push('/profile/info/help')} />
          <Divider />
          <Row icon="sparkle" label={t('rows.about')} trailing={chevron} onClick={() => router.push('/profile/info/about')} />
          <Divider />
          <Row icon="book" label={t('rows.terms')} trailing={chevron} onClick={() => router.push('/profile/info/terms')} />
        </Group>

        {/* Logout */}
        <section style={{ padding: '0 18px', marginTop: 20 }}>
          <div className="card" style={{ overflow: 'hidden' }}>
            <Row
              icon="logout"
              label={logout.isPending ? t('loggingOut') : t('logout')}
              danger
              onClick={handleLogout}
              disabled={logout.isPending}
            />
          </div>
        </section>

        <div style={{ textAlign: 'center', fontSize: 12, color: 'var(--muted)', marginTop: 18 }}>
          {t('version', { version: localizeNum(APP_VERSION, loc) })}
        </div>
      </div>

      <DeleteAccountConfirm open={deleteOpen} onClose={() => setDeleteOpen(false)} />

      <BottomNav />
    </div>
  );
}
