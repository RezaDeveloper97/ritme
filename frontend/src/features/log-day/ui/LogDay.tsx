'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useRef, useState } from 'react';

import { usePlusLocked } from '@/entities/plus';
import { useRouter, type Locale } from '@/shared/i18n';
import { toApiDate, today } from '@/shared/lib/date';
import {
  EmptyState,
  HeaderButton,
  Icon,
  ScreenHeader,
  SearchField,
  SecondaryButton,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
} from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';

import { matchesSearch } from '../model/draft';
import { isMenopausePreset } from '../model/menopause-preset';
import { categoryLook, type PanelKind } from '../model/presentation';
import { useLogDayStore } from '../model/store';
import { useLogDayHeading } from '../model/use-heading';
import { useLogDayController } from '../model/use-log-day';
import { CategorySection } from './CategorySection';
import { LogDateStrip } from './LogDateStrip';
import { BleedingPanel } from './panels/BleedingPanel';
import { DetailPanel } from './panels/DetailPanel';
import { MeasurePanel } from './panels/MeasurePanel';
import { PainPanel, type BodyMapSlot } from './panels/PainPanel';
import { MenopausePreset } from './presets/MenopausePreset';
import { QuickTiles } from './QuickTiles';
import { SummaryFooter } from './SummaryFooter';
import { VOICE_FEATURE, VoiceTab, type VoiceLogSlot } from './VoiceTab';

/**
 * Where the gear opens the log customisation screen (B-N3-04). `null` until that screen exists — the
 * header keeps a spacer so the title stays centred.
 */
export const LOG_CUSTOMIZE_HREF: string | null = '/log/customize';

/** Where a logged post-menopause bleeding leads (CB-MENO-09, nbl_Meno_Alert). */
const MENOPAUSE_ALERT_HREF = '/menopause/alert';

export interface LogDayProps {
  /** `sheet` = content of the `?sheet=log` panel (the host draws the title); `page` = the `/log` route. */
  variant: 'sheet' | 'page';
  /** API date to open on (`YYYY-MM-DD`); today when absent, invalid or in the future. */
  initialDate?: string | null;
  /** Taxonomy mode; omit for the user's current life-stage mode (pregnancy / postpartum sheets pass theirs). */
  mode?: string;
  /** The body map widget for the pain panel (a widget can't be imported from a feature). */
  BodyMap?: BodyMapSlot;
  /** The voice recorder for the «ثبت با صدا» tab (B-N3-05 `features/voice-log`, passed by the screen). */
  VoiceLog?: VoiceLogSlot;
  /** Page variant: the header's ×. */
  onClose?: () => void;
  /**
   * Mode presets (CB-MENO-06): a menopause day opens on the board's grouped rows (nbl_Meno_Log) above
   * the full category list. `false` keeps bloom's quick tiles + accordion for every mode.
   */
  preset?: boolean;
  /** Page variant: own title/subtitle and a back arrow instead of «ثبت امروز» + × + gear (`/menopause/log`). */
  pageHeader?: { title: string; subtitle: string };
  /** The date strip above the tabs (off on `/menopause/log`, whose day comes from `?date=`). */
  dateStrip?: boolean;
  /** After a successful save (the sheet closes itself). */
  onSaved?: () => void;
}

function validDate(value: string | null | undefined): string {
  const todayKey = toApiDate(today());
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value) || value > todayKey) return todayKey;
  return value;
}

type Tab = 'manual' | 'voice';

/**
 * Log sheet v2 (nbl_/nbd_Log_Sheet_Cycle): date strip, manual / voice tabs, quick tiles that open their
 * section, every category of the mode as an accordion rendered from `/logs/taxonomy`, search, the detail
 * panels (bleeding, pain + body map, weight/BBT/tests) and a summary footer that saves the day in one PUT.
 * Mode-agnostic: the pregnancy and postpartum sheets (B-N3-06) reuse it with their `mode`.
 */
export function LogDay({
  variant,
  initialDate,
  mode: modeOverride,
  BodyMap,
  VoiceLog,
  onClose,
  onSaved,
  preset = true,
  pageHeader,
  dateStrip = true,
}: LogDayProps) {
  const t = useTranslations('logSheet');
  const router = useRouter();
  const locale = useLocale() as Locale;
  const [date, setDate] = useState(() => validDate(initialDate));
  const [tab, setTab] = useState<Tab>('manual');
  const [openCat, setOpenCat] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [panel, setPanel] = useState<PanelKind | null>(null);
  const setStoreDate = useLogDayStore((s) => s.setDate);
  const sectionRefs = useRef(new Map<string, HTMLDivElement>());
  const searchRef = useRef<HTMLDivElement>(null);
  const tabsId = useId();
  const voiceLocked = usePlusLocked(VOICE_FEATURE);
  // A voice «ذخیره» merges suggestions with setParam and saves in the same event: the save runs on the
  // next render, once the merged draft is in the controller.
  const [saveQueued, setSaveQueued] = useState<{ onDone?: () => void } | null>(null);
  // Voice record / review / saved hide the strip and tabs (CB-VOICE-02).
  const [immersive, setImmersive] = useState(false);

  const c = useLogDayController(date, modeOverride);
  const heading = useLogDayHeading(date, modeOverride);
  const menopause = preset && !c.loading && !c.error && isMenopausePreset(c.mode);

  useEffect(() => {
    if (!saveQueued) return;
    setSaveQueued(null);
    if (saveQueued.onDone && !c.dirty) saveQueued.onDone();
    else c.save(saveQueued.onDone ?? onSaved);
  }, [saveQueued, c, onSaved]);

  useEffect(() => {
    setStoreDate(date);
    return () => setStoreDate(null);
  }, [date, setStoreDate]);

  // A sheet re-opened for another day (calendar «ثبت جزئیات») adopts it.
  useEffect(() => {
    setDate(validDate(initialDate));
  }, [initialDate]);

  const openSection = (code: string) => {
    setQuery('');
    setOpenCat(code);
    requestAnimationFrame(() => {
      const el = sectionRefs.current.get(code);
      if (!el) return;
      const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
      el.scrollIntoView({ block: 'start', behavior: reduce ? 'auto' : 'smooth' });
    });
  };

  const searchPanel = (q: string) => {
    setPanel(null);
    setOpenCat(null);
    setQuery(q);
    requestAnimationFrame(() => searchRef.current?.scrollIntoView({ block: 'start' }));
  };

  const visible = query
    ? c.categories.filter((cat) =>
        matchesSearch(
          cat,
          query,
          cat.params.flatMap((p) => c.extraOptions(cat.code, p.code).map((o) => o.label)),
        ),
      )
    : c.categories;

  const category = (code: string) => c.categories.find((cat) => cat.code === code);
  const bleeding = category('bleeding');
  const pain = category('pain');
  const measurements = category('measurements');

  const header =
    variant === 'page' && pageHeader ? (
      <ScreenHeader
        title={pageHeader.title}
        subtitle={pageHeader.subtitle}
        onBack={onClose}
        backLabel={t('presets.menopause.back')}
      />
    ) : variant === 'page' ? (
      <ScreenHeader
        title={heading.title}
        subtitle={heading.subtitle}
        onBack={onClose}
        backIcon="close"
        backLabel={t('close')}
        action={<CustomizeButton />}
      />
    ) : null;

  let body;
  if (c.loading) {
    body = (
      <SkeletonGroup label={t('loading')} className="lday-skel">
        <div className="lday-tiles">
          {Array.from({ length: 8 }, (_, i) => (
            <Skeleton key={i} shape="card" />
          ))}
        </div>
        <Skeleton shape="block" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (c.error) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <SecondaryButton block={false} onClick={c.retry}>
            {t('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = (
      <>
        {menopause ? (
          <MenopausePreset
            categories={c.categories}
            values={c.values}
            setParam={c.setParam}
            onVoice={() => setTab('voice')}
            voiceLocked={voiceLocked}
            onBleedingHelp={() => (c.dirty ? c.save(() => router.push(MENOPAUSE_ALERT_HREF)) : router.push(MENOPAUSE_ALERT_HREF))}
          />
        ) : (
          <QuickTiles
            tiles={c.tiles}
            categories={c.categories}
            values={c.values}
            labels={c.labels}
            openCategory={query ? null : openCat}
            onOpen={(code) => (openCat === code && !query ? setOpenCat(null) : openSection(code))}
          />
        )}
        {/* The full page (/menopause/log) is the board alone; the sheet keeps every category below the preset. */}
        {menopause && pageHeader ? null : (
          <section className="lday-all" aria-labelledby="lday-all-title">
            <h3 id="lday-all-title" className="lday-sec-title">
              {menopause ? t('presets.menopause.more') : t('all')}
            </h3>
            <div ref={searchRef}>
              <SearchField
                value={query}
                onValueChange={setQuery}
                label={t('search.label')}
                clearLabel={t('search.clear')}
                placeholder={t('search.placeholder')}
              />
            </div>
            {visible.length ? (
              visible.map((cat) => (
                <CategorySection
                  key={cat.code}
                  ref={(el) => {
                    if (el) sectionRefs.current.set(cat.code, el);
                    else sectionRefs.current.delete(cat.code);
                  }}
                  category={cat}
                  values={c.values}
                  setParam={c.setParam}
                  extraOptions={c.extraOptions}
                  mode={c.mode}
                  labels={c.labels}
                  locale={locale}
                  open={query ? true : openCat === cat.code}
                  onOpenChange={(open) => {
                    if (query) return;
                    setOpenCat(open ? cat.code : null);
                  }}
                  onOpenPanel={setPanel}
                />
              ))
            ) : (
              <EmptyState icon="search" title={t('search.emptyTitle')} body={t('search.emptyBody')} className="lday-empty" />
            )}
          </section>
        )}
      </>
    );
  }

  const content = (
    <div className="lday" data-variant={variant}>
      {dateStrip && !(immersive && tab === 'voice') ? <LogDateStrip date={date} onSelect={setDate} /> : null}
      {immersive && tab === 'voice' ? null : menopause ? (
        tab === 'voice' ? (
          <button type="button" className="mlog-manual" onClick={() => setTab('manual')}>
            <Icon name="chevronRight" size={18} className="mlog-manual-chev" />
            {t('presets.menopause.voice.back')}
          </button>
        ) : null
      ) : (
        <SegmentedTabs<Tab>
          tabs={[
            { value: 'manual', label: t('tabs.manual') },
            {
              value: 'voice',
              icon: 'mic',
              label: (
                <>
                  {t('tabs.voice')}
                  {voiceLocked ? <PlusBadge label={t('tabs.plus')} className="lday-tab-plus" /> : null}
                </>
              ),
            },
          ]}
          value={tab}
          onChange={setTab}
          label={t('tabs.label')}
          panelId={(v) => `${tabsId}-${v}`}
          track="surface"
          className="lday-tabs"
        />
      )}
      <div id={`${tabsId}-${tab}`} role={menopause ? undefined : 'tabpanel'} className="lday-panelwrap">
        {tab === 'voice' ? (
          <VoiceTab onManual={() => setTab('manual')}>
            {VoiceLog && !c.loading && !c.error ? (
              <VoiceLog
                date={date}
                mode={c.mode}
                categories={c.categories}
                values={c.values}
                setParam={c.setParam}
                markVoice={c.markVoice}
                openSection={(code) => {
                  setTab('manual');
                  openSection(code);
                }}
                toneOf={(code) => categoryLook(code).tone}
                iconOf={(code) => categoryLook(code).icon}
                entries={c.entries}
                save={(onDone) => setSaveQueued({ onDone })}
                saving={c.saving}
                saveError={c.saveError}
                onManual={() => setTab('manual')}
                onDone={() => (onSaved ?? onClose)?.()}
                onImmersive={setImmersive}
              />
            ) : VoiceLog ? (
              body
            ) : null}
          </VoiceTab>
        ) : (
          body
        )}
      </div>
      {tab === 'manual' && !c.loading && !c.error ? (
        <SummaryFooter
          entries={c.entries}
          dirty={c.dirty}
          saving={c.saving}
          saveError={c.saveError}
          justSaved={c.justSaved}
          queued={c.queued}
          onSave={() => c.save(onSaved)}
        />
      ) : null}

      {bleeding ? (
        <DetailPanel
          open={panel === 'bleeding'}
          onClose={() => setPanel(null)}
          title={bleeding.label}
          subtitle={heading.panelSub}
          action="confirm"
          onAction={() => setPanel(null)}
          onSearch={searchPanel}
        >
          <BleedingPanel
            category={bleeding}
            values={c.values}
            setParam={c.setParam}
            extraOptions={c.extraOptions}
            mode={c.mode}
            locale={locale}
          />
        </DetailPanel>
      ) : null}
      {pain ? (
        <DetailPanel
          open={panel === 'pain'}
          onClose={() => setPanel(null)}
          title={pain.label}
          subtitle={t('pain.sub')}
          action="confirm"
          onAction={() => setPanel(null)}
          onSearch={searchPanel}
        >
          <PainPanel category={pain} values={c.values} setParam={c.setParam} mode={c.mode} locale={locale} BodyMap={BodyMap} />
        </DetailPanel>
      ) : null}
      {measurements ? (
        <DetailPanel
          open={panel === 'measure'}
          onClose={() => setPanel(null)}
          title={measurements.label}
          subtitle={heading.panelSub}
          action="save"
          busy={c.saving}
          onAction={() => (c.dirty ? c.save(() => setPanel(null)) : setPanel(null))}
          onSearch={searchPanel}
        >
          <MeasurePanel
            category={measurements}
            values={c.values}
            setParam={c.setParam}
            mode={c.mode}
            locale={locale}
            date={date}
          />
        </DetailPanel>
      ) : null}
    </div>
  );

  return (
    <>
      {header}
      {content}
    </>
  );
}

/** The gear (B-N3-04 customisation) — a spacer until that screen exists. */
function CustomizeButton() {
  const t = useTranslations('logSheet');
  const router = useRouter();
  const href = LOG_CUSTOMIZE_HREF;
  if (!href) return <span className="nb-hdr-spacer" aria-hidden />;
  return <HeaderButton icon="cog" label={t('customize')} onClick={() => router.push(href)} />;
}

/**
 * The `?sheet=log` heading: «ثبت امروز» over «شنبه، ۱۲ مهر · روز ۲۵ سیکل», centred between the × and
 * the gear like the artboard. Reads the day from the sheet content through `useLogDayStore`.
 */
export function LogDaySheetTitle({ mode }: { mode?: string }) {
  const date = useLogDayStore((s) => s.date);
  const heading = useLogDayHeading(date, mode);
  return (
    <span className="lday-ttl">
      <span className="lday-ttl-main">{heading.title}</span>
      <span className="lday-ttl-sub">{heading.subtitle}</span>
    </span>
  );
}
