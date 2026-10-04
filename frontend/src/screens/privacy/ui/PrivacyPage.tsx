'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { companionName, sharedSections, useCompanions } from '@/entities/companion';
import {
  getLockController,
  isBiometricAvailable,
  LOCK_TIMEOUTS,
  type LockTimeout,
  PasscodeSheet,
  registerBiometric,
  useAppLock,
} from '@/features/app-lock';
import { DeleteAccountConfirm, useExportData, useExportPdf } from '@/features/manage-account';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatLongDate, formatNumber, today } from '@/shared/lib/date';
import {
  ListGroup,
  ListRow,
  ScreenHeader,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  Switch,
} from '@/shared/ui';

import { useConsents, useUpdateConsent } from '../api/consents';

function SectionLabel({ id, children }: { id: string; children: string }) {
  return (
    <h2 id={id} className="prv-label">
      {children}
    </h2>
  );
}

/** «قفل اپ»: passcode on/off (sheet), lock delay, biometrics, hide preview. */
function LockSection() {
  const t = useTranslations('me.privacy');
  const lock = useAppLock();
  const [sheet, setSheet] = useState<'create' | 'verify' | null>(null);
  const [bioAvailable, setBioAvailable] = useState(false);
  const [bioError, setBioError] = useState(false);
  const appName = useTranslations('common')('appName');

  useEffect(() => {
    let alive = true;
    void isBiometricAvailable().then((ok) => alive && setBioAvailable(ok));
    return () => {
      alive = false;
    };
  }, []);

  const enabled = lock?.enabled ?? false;
  const c = getLockController();

  const onBiometric = async (next: boolean) => {
    if (!c) return;
    setBioError(false);
    if (!next) return c.setBiometric(null);
    const id = await registerBiometric(appName);
    if (id) c.setBiometric(id);
    else setBioError(true);
  };

  return (
    <section className="prv-sec" aria-labelledby="prv-g-lock">
      <SectionLabel id="prv-g-lock">{t('groups.lock')}</SectionLabel>
      <ListGroup className="prv-list">
        <ListRow
          id="prv-lock"
          title={t('lock.title')}
          description={enabled ? t('lock.on') : t('lock.off')}
          trailing={
            <Switch
              checked={enabled}
              disabled={lock === null}
              labelledBy="prv-lock-title"
              onCheckedChange={(next) => setSheet(next ? 'create' : 'verify')}
            />
          }
        />
        {enabled && lock ? (
          <div className="prv-timeout">
            <span id="prv-timeout-l" className="prv-timeout-label">
              {t('lock.after')}
            </span>
            <SegmentedTabs
              label={t('lock.after')}
              tabs={LOCK_TIMEOUTS.map((m) => ({ value: String(m), label: t('lock.timeout', { minutes: m }) }))}
              value={String(lock.timeoutMin)}
              onChange={(v) => c?.setTimeoutMin(Number(v) as LockTimeout)}
              className="prv-timeout-tabs"
            />
          </div>
        ) : null}
        {enabled && lock && bioAvailable ? (
          <ListRow
            id="prv-bio"
            title={t('lock.biometric')}
            description={bioError ? t('lock.biometricFailed') : t('lock.biometricSub')}
            trailing={
              <Switch checked={lock.biometric} labelledBy="prv-bio-title" onCheckedChange={(n) => void onBiometric(n)} />
            }
          />
        ) : null}
        <ListRow
          id="prv-veil"
          title={t('lock.hidePreview')}
          description={t('lock.hidePreviewSub')}
          trailing={
            <Switch
              checked={lock?.hidePreview ?? false}
              disabled={lock === null}
              labelledBy="prv-veil-title"
              onCheckedChange={(next) => c?.setHidePreview(next)}
            />
          }
        />
      </ListGroup>
      <PasscodeSheet
        open={sheet !== null}
        mode={sheet ?? 'create'}
        onClose={() => setSheet(null)}
        onDone={() => {
          if (sheet === 'verify') c?.disable();
          setSheet(null);
        }}
      />
    </section>
  );
}

/** «داده‌ها و هوش مصنوعی»: server-side consents with their grant date. */
function ConsentSection() {
  const t = useTranslations('me.privacy');
  const loc = useLocale() as Locale;
  const query = useConsents();
  const update = useUpdateConsent();

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('consents.loading')} className="prv-skel">
        {[0, 1, 2].map((i) => (
          <div key={i} className="prv-skel-row">
            <span className="prv-skel-text">
              <Skeleton width="medium" />
              <Skeleton width="short" />
            </span>
            <Skeleton shape="block" className="prv-skel-switch" />
          </div>
        ))}
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
    body = (
      <div className="prv-inline-error" role="alert">
        <span>{t('consents.error')}</span>
        <button type="button" className="prv-retry" onClick={() => void query.refetch()}>
          {t('consents.retry')}
        </button>
      </div>
    );
  } else {
    body = query.data.map((c) => (
      <ListRow
        key={c.code}
        id={`prv-c-${c.code}`}
        title={t(`consents.${c.code}.title`)}
        description={
          c.granted && c.grantedAt
            ? t('consents.granted', { date: formatDayMonth(new Date(c.grantedAt), loc) })
            : t(`consents.${c.code}.sub`)
        }
        trailing={
          <Switch
            checked={c.granted}
            labelledBy={`prv-c-${c.code}-title`}
            onCheckedChange={(granted) => update.mutate({ code: c.code, granted, version: c.version })}
          />
        }
      />
    ));
  }

  return (
    <section className="prv-sec" aria-labelledby="prv-g-data">
      <SectionLabel id="prv-g-data">{t('groups.data')}</SectionLabel>
      <ListGroup className="prv-list">{body}</ListGroup>
      {update.isError ? (
        <p className="prv-error" role="alert">
          {t('consents.saveError')}
        </p>
      ) : null}
    </section>
  );
}

/**
 * «ریتمی همراه» rows (B-N4-04): one per active companion with what it sees
 * («علی · پریود، داروها»), then «مدیریت همدم‌ها» → /companions. With no
 * companion yet it is the single «هنوز با کسی به اشتراک نگذاشته‌ای» row.
 */
function CompanionRows() {
  const t = useTranslations('me.privacy');
  const tc = useTranslations('companions');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const query = useCompanions();
  const companions = query.data ?? [];
  const active = companions.filter((c) => c.status === 'active');
  const pending = companions.length - active.length;

  if (query.isPending) {
    return (
      <SkeletonGroup label={tc('privacy.loading')} className="prv-cmp-skel">
        <Skeleton width="medium" />
      </SkeletonGroup>
    );
  }
  if (query.isError || companions.length === 0) {
    return (
      <ListRow
        icon="users"
        iconTone="data"
        title={t('access.companion')}
        description={query.isError ? tc('privacy.error') : t('access.companionSub')}
        onClick={() => router.push('/companions')}
      />
    );
  }
  return (
    <>
      {active.map((c) => {
        const shared = sharedSections(c.grants).map(({ section }) => tc(`sectionsShort.${section}`));
        const name = companionName(c) ?? tc('unnamed');
        return (
          <ListRow
            key={c.id}
            icon="users"
            iconTone="data"
            title={t('access.companion')}
            description={`${name} · ${shared.length ? shared.join(tc('listSeparator')) : tc('nothingShared')}`}
            onClick={() => router.push(`/companions/${c.id}`)}
          />
        );
      })}
      <ListRow
        icon="cog"
        iconTone="brand"
        title={tc('privacy.manage')}
        description={pending > 0 ? tc('privacy.pending', { count: pending }) : undefined}
        value={pending > 0 ? undefined : formatNumber(companions.length, loc)}
        onClick={() => router.push('/companions')}
      />
    </>
  );
}

/**
 * Privacy & security (B-N1-12, `nbl_Me_Privacy` / `nbd_Me_Privacy`) at
 * `/profile/privacy`: the device-local app lock, the server-side consents,
 * «دسترسی دیگران» (companion + doctor links arrive with B-N4 / B-N6), export
 * (JSON / PDF) and account deletion.
 */
export function PrivacyPage() {
  const t = useTranslations('me.privacy');
  const soon = useTranslations('me')('soon');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const dir = useDirection();
  const json = useExportData();
  const pdf = useExportPdf();
  const [exportOpen, setExportOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const exporting = json.isPending || pdf.isPending;

  const exportPdf = () =>
    pdf.exportPdf({
      title: t('mine.pdfTitle'),
      subtitle: t('mine.pdfSubtitle', { date: formatLongDate(today(), loc) }),
      footer: t.raw('mine.pdfFooter') as string,
      dir,
      locale: loc,
    });

  return (
    <div className="view prv-page">
      <SkyLayer />
      <div className="scroll prv-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/profile')} backLabel={t('back')} />

        <LockSection />
        <ConsentSection />

        <section className="prv-sec" aria-labelledby="prv-g-access">
          <SectionLabel id="prv-g-access">{t('groups.access')}</SectionLabel>
          <ListGroup className="prv-list">
            <CompanionRows />
            <ListRow
              icon="stetho"
              iconTone="brand"
              title={t('access.doctors')}
              description={t('access.doctorsSub')}
              trailing={<StatusPill tone="neutral">{soon}</StatusPill>}
            />
          </ListGroup>
        </section>

        <section className="prv-sec" aria-labelledby="prv-g-mine">
          <SectionLabel id="prv-g-mine">{t('groups.mine')}</SectionLabel>
          <ListGroup className="prv-list">
            <ListRow
              icon="download"
              iconTone="brand"
              title={t('mine.export')}
              description={exporting ? t('mine.exporting') : t('mine.exportSub')}
              onClick={() => setExportOpen((v) => !v)}
            />
            {exportOpen ? (
              <div className="prv-export">
                <button type="button" className="prv-export-btn" disabled={exporting} onClick={() => json.exportData()}>
                  {t('mine.exportJson')}
                </button>
                <button type="button" className="prv-export-btn" disabled={exporting} onClick={exportPdf}>
                  {t('mine.exportPdf')}
                </button>
              </div>
            ) : null}
            <ListRow
              icon="shield"
              iconTone="data"
              title={t('mine.backup')}
              description={t('mine.backupSub')}
              trailing={<StatusPill tone="neutral">{soon}</StatusPill>}
            />
          </ListGroup>
          {json.isError || pdf.isError ? (
            <p className="prv-error" role="alert">
              {t('mine.exportError')}
            </p>
          ) : null}
        </section>

        <button type="button" className="prv-delete" onClick={() => setDeleteOpen(true)}>
          {t('mine.delete')}
        </button>
        <p className="prv-note">{t('mine.deleteNote')}</p>
      </div>
      <DeleteAccountConfirm open={deleteOpen} onClose={() => setDeleteOpen(false)} />
    </div>
  );
}
