'use client';

/*
 * Dev-only showcase of the Night & Bloom primitives (B-N1-03), route
 * /dev/ui-kit (404 in production builds). The copy below is sample content
 * lifted from the artboards (nbl_Cycle_Settings, nbl_Log_Sheet_Cycle,
 * nbl_Onb_Cycle) so the kit can be compared side by side with them — it is
 * not product copy and never reaches users, so it is not translated.
 */
import { useLocale } from 'next-intl';
import { useState } from 'react';

import { formatNumber, today as todayDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { useThemeStore } from '@/shared/theme';
import {
  Accordion,
  Avatar,
  BarChart,
  Card,
  Checkbox,
  ChipGroup,
  CountdownRing,
  DateStrip,
  EmptyState,
  HeaderButton,
  HeroCard,
  HubHeader,
  InfoNote,
  LineChart,
  ListGroup,
  ListRow,
  NumberStepper,
  NumericScale,
  PillChip,
  PlusLock,
  PrimaryButton,
  ProgressBar,
  ProgressRing,
  ProgressSteps,
  RadioCardGroup,
  ScreenHeader,
  SearchField,
  SecondaryButton,
  SectionTitle,
  SegmentedTabs,
  SeverityScale,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  StepTimeline,
  Switch,
  TileButton,
  UrgentCard,
  WeekDots,
  type TimelineStep,
} from '@/shared/ui';

const PAIN = ['بدون درد', 'شکم', 'لگن', 'یک‌طرفه پایین', 'کمر', 'سر', 'سینه'] as const;
const LEVELS = ['کم', 'متوسط', 'شدید'] as const;
const BBT = [36.35, 36.4, 36.3, 36.38, null, 36.42, 36.36, 36.5, 36.62, 36.7, 36.68, 36.72];
const CYCLES = [28, 30, 27, 29, 31, 29];

// Canvas-v1 samples (CB-CORE-02b) — lifted from nbl_Cond_Endo, nbl_Cond_PMDD,
// nbl_IVF_Home, nbl_Ins_ClaimDetail, nbl_Pelv_Session, nbl_Contra_Pill.
const SEVERITY = [
  { value: 'none', label: 'ندارم' },
  { value: 'mild', label: 'خفیف' },
  { value: 'moderate', label: 'متوسط' },
  { value: 'severe', label: 'شدید' },
] as const;
type Severity = (typeof SEVERITY)[number]['value'];
const STEP_STATES = { done: 'انجام شد', current: 'مرحله فعلی', todo: 'بعدی' } as const;
const IVF_STEPS: TimelineStep[] = [
  { id: 'stim', title: 'تحریک تخمک‌گذاری', meta: '۱ تا ۱۰ مهر', state: 'done' },
  { id: 'retrieval', title: 'تخمک‌کشی', meta: '۱۲ مهر', state: 'current' },
  { id: 'transfer', title: 'انتقال جنین', meta: 'حدود ۳ تا ۵ روز بعد', state: 'todo' },
];
const CLAIM_STEPS: TimelineStep[] = [
  { id: 'sent', title: 'ارسال شد', meta: '۵ مهر', state: 'done' },
  { id: 'review', title: 'درخواست مدرک', meta: 'مهلت تا ۱۶ مهر', state: 'current', tone: 'bloom' },
  { id: 'paid', title: 'پرداخت', state: 'todo' },
];
const WEEK = ['done', 'done', 'missed', 'done', 'done', 'future', 'future'] as const;
const WEEK_STATES = { done: 'خورده شد', missed: 'جا افتاد', future: 'هنوز نرسیده' } as const;
type Method = 'pill' | 'iud' | 'none';
const HOTLINES = [
  { label: 'اورژانس', number: '115', display: '۱۱۵' },
  { label: 'صدای مشاور', number: '1480', display: '۱۴۸۰' },
  { label: 'اورژانس اجتماعی', number: '123', display: '۱۲۳' },
];

export function UiKitPage() {
  const locale = useLocale();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const now = todayDate();

  const [day, setDay] = useState(now);
  const [tab, setTab] = useState<'manual' | 'quick'>('manual');
  const [period, setPeriod] = useState(5);
  const [cycle, setCycle] = useState(29);
  const [auto, setAuto] = useState(true);
  const [reminders, setReminders] = useState({ period: true, pms: true, fertile: false });
  const [pain, setPain] = useState<string[]>(['شکم']);
  const [level, setLevel] = useState<string>('متوسط');
  const [sheet, setSheet] = useState(false);
  const [severity, setSeverity] = useState<Severity | null>('moderate');
  const [pain10, setPain10] = useState<number | null>(7);
  const [mood6, setMood6] = useState<number | null>(3);
  const [method, setMethod] = useState<Method | null>('pill');
  const [query, setQuery] = useState('');
  const [consent, setConsent] = useState(true);
  const [share, setShare] = useState(false);

  const togglePain = (item: string) =>
    setPain((prev) => (prev.includes(item) ? prev.filter((p) => p !== item) : [...prev, item]));

  return (
    <div className="view uikit">
      <div className="scroll uikit-scroll">
        <SkyLayer />
        <ScreenHeader
          title="کیت رابط کاربری"
          subtitle="Night & Bloom · B-N1-03 + CB-CORE-02"
          onBack={() => setTab('manual')}
          backLabel="بازگشت"
          action={
            <HeaderButton
              label="تغییر پوسته"
              icon={theme === 'dark' ? 'sun' : 'moon'}
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            />
          }
        />
        <div className="uikit-body">
          <HubHeader
            date="شنبه ۱۲ مهر"
            greeting="سلام مریم"
            actions={
              <>
                <HeaderButton label="اعلان‌ها" icon="bell" variant="soft" badge />
                <HeaderButton label="تنظیمات" icon="cog" variant="soft" />
              </>
            }
            className="uikit-flush"
          />

          <DateStrip
            locale={locale}
            selected={day}
            today={now}
            onSelect={setDay}
            maxDate={now}
            marker={(d) => (d.getDate() % 3 === 0 ? 'period' : d.getDate() % 3 === 1 ? 'brand' : null)}
            markedLabel="ثبت شده"
            label="روزهای این هفته"
          />

          <SegmentedTabs
            label="نوع ثبت"
            value={tab}
            onChange={setTab}
            track="surface"
            tabs={[
              { value: 'manual', label: 'ثبت دستی' },
              { value: 'quick', label: 'ثبت سریع', icon: 'zap' },
            ]}
          />

          <SectionTitle title="سیکل تو" actionLabel="جزئیات" onAction={() => setSheet(true)} />
          <HeroCard as="section" aria-label="حلقه سیکل" className="uikit-hero">
            <ProgressRing value={14 / 29} label="پیشرفت سیکل" valueText="روز ۱۴ از ۲۹" size={148}>
              <span className="uikit-num">۱۴</span>
              <span className="uikit-cap">روز سیکل</span>
            </ProgressRing>
            <div className="uikit-hero-side">
              <StatusPill tone="data" icon="check">
                منظم
              </StatusPill>
              <StatusPill tone="warm">پنجره باروری</StatusPill>
              <StatusPill tone="period">پریود</StatusPill>
              <StatusPill solid>محبوب</StatusPill>
            </div>
          </HeroCard>

          <Card as="section" padding="none" className="uikit-stack">
            <NumberStepper
              label="طول پریود"
              description="معمولاً چند روز خون‌ریزی داری؟"
              unit="روز"
              value={period}
              min={2}
              max={10}
              onChange={setPeriod}
              decrementLabel="کم کردن"
              incrementLabel="زیاد کردن"
              locale={locale}
              boxed
            />
            <NumberStepper
              label="طول سیکل"
              description="از شروع یک پریود تا پریود بعد"
              unit="روز"
              value={cycle}
              min={21}
              max={45}
              onChange={setCycle}
              decrementLabel="کم کردن"
              incrementLabel="زیاد کردن"
              locale={locale}
              boxed
            />
          </Card>

          <ListGroup title="یادآورها">
            <ListRow
              id="kit-auto"
              icon="refresh"
              iconTone="data"
              iconOutlined
              title="محاسبه خودکار"
              description="از ۶ سیکل اخیر"
              trailing={<Switch checked={auto} onCheckedChange={setAuto} labelledBy="kit-auto-title" />}
            />
            <ListRow
              id="kit-period"
              icon="drop"
              iconTone="period"
              iconOutlined
              title="قبل از پریود"
              description="۲ روز قبل · ۹:۰۰"
              trailing={
                <Switch
                  checked={reminders.period}
                  onCheckedChange={(v) => setReminders((r) => ({ ...r, period: v }))}
                  labelledBy="kit-period-title"
                />
              }
            />
            <ListRow
              id="kit-fertile"
              icon="sparkle"
              iconTone="warm"
              iconOutlined
              title="پنجره باروری"
              description="روز اول پنجره"
              trailing={
                <Switch
                  checked={reminders.fertile}
                  onCheckedChange={(v) => setReminders((r) => ({ ...r, fertile: v }))}
                  labelledBy="kit-fertile-title"
                  compact
                />
              }
            />
            <ListRow icon="globe" title="زبان" value="فارسی" onClick={() => setSheet(true)} />
          </ListGroup>

          <SectionTitle title="ثبت سریع" level={2} />
          <div className="uikit-tiles">
            <TileButton layout="card" icon="drop" tone="period" label="پریود" pressed={false} />
            <TileButton layout="card" icon="zap" tone="bloom" label="درد" sub="باز است" pressed />
            <TileButton layout="card" icon="smile" tone="brand" label="حال" sub="زودرنج" pressed={false} />
            <TileButton layout="card" icon="moon" tone="data" label="خواب" pressed={false} />
          </div>
          <div className="uikit-tiles is-3">
            <TileButton icon="drop" tone="period" label="شروع پریود" />
            <TileButton icon="calendar" tone="brand" label="تقویم" />
            <TileButton icon="note" tone="neutral" label="یادداشت" />
          </div>

          <Accordion title="پریود و لکه‌بینی" summary="ثبت نشده" icon="drop" tone="period">
            <ChipGroup label="میزان خون‌ریزی">
              <PillChip pressed={false}>لکه‌بینی</PillChip>
              <PillChip pressed>متوسط</PillChip>
            </ChipGroup>
          </Accordion>
          <Accordion
            title="درد"
            summary={`${pain.join('، ') || 'ثبت نشده'} · ${level}`}
            icon="zap"
            tone="bloom"
            active={pain.length > 0}
            defaultOpen
          >
            <ChipGroup label="محل درد">
              {PAIN.map((item) => (
                <PillChip key={item} mode="multi" tone="bloom" pressed={pain.includes(item)} onPressedChange={() => togglePain(item)}>
                  {item}
                </PillChip>
              ))}
            </ChipGroup>
            <ChipGroup label="شدت" layout="fill">
              {LEVELS.map((item) => (
                <PillChip key={item} shape="square" pressed={level === item} onPressedChange={() => setLevel(item)}>
                  {item}
                </PillChip>
              ))}
            </ChipGroup>
            <ChipGroup label="روز پیشنهادی">
              <PillChip pressed={false} suggested>
                ۲۸ شهریور
              </PillChip>
            </ChipGroup>
          </Accordion>

          <SectionTitle title="نمودارها" />
          <Card as="section" aria-label="دمای پایه بدن">
            <LineChart
              label="دمای پایه بدن، ۱۲ روز اخیر"
              series={[{ values: BBT, tone: 'data', points: true }]}
              band={{ lower: BBT.map(() => 36.3), upper: BBT.map(() => 36.5), tone: 'brand' }}
              yTicks={[36.2, 36.5, 36.8]}
              yLabels={['۳۶٫۲', '۳۶٫۵', '۳۶٫۸']}
              xLabels={['۱', '', '', '۴', '', '', '۷', '', '', '۱۰', '', '۱۲']}
              min={36.2}
              max={36.8}
              highlightIndex={11}
            />
          </Card>
          <Card as="section" aria-label="طول سیکل‌ها">
            <BarChart
              label="طول ۶ سیکل اخیر"
              bars={CYCLES.map((v, i) => ({
                label: ['فرو', 'ارد', 'خرد', 'تیر', 'مرد', 'شهر'][i],
                value: v,
                valueLabel: formatNumber(v, locale),
                highlight: i === CYCLES.length - 1,
              }))}
              max={35}
            />
          </Card>

          <PlusLock label="پلاس" lockedText="این تحلیل در ریتمی پلاس باز می‌شود" onUnlock={() => setSheet(true)}>
            <Card>
              <SectionTitle title="الگوی علائم" level={3} />
              <p className="uikit-cap">نفخ و زودرنجی معمولاً ۳ روز قبل از پریودت بیشتر می‌شود.</p>
            </Card>
          </PlusLock>

          <InfoNote source="منبع: FIGO، WHO">
            پیش‌بینی‌ها اطلاعاتی‌اند و جای مشاوره پزشکی را نمی‌گیرند.
          </InfoNote>
          <UrgentCard title="اگر حالت خوب نیست، تنها نیستی" action={<PrimaryButton icon="phone">تماس با ۱۱۵</PrimaryButton>}>
            در شرایط اضطراری همین حالا تماس بگیر.
          </UrgentCard>

          <Card className="uikit-row">
            <Avatar name="مریم" />
            <Avatar name="سارا" size="sm" />
            <ProgressSteps total={8} current={3} label="مرحله ۳ از ۸" />
          </Card>

          <SkeletonGroup label="در حال بارگذاری">
            <Skeleton shape="card" />
            <Skeleton shape="line" width="medium" />
            <Skeleton shape="line" width="short" />
          </SkeletonGroup>

          <Card>
            <EmptyState
              icon="note"
              title="هنوز کاری نداری"
              body="اولین کار امروزت را اضافه کن تا اینجا ببینی‌اش."
              action={<SecondaryButton icon="plus">افزودن کار</SecondaryButton>}
            />
          </Card>

          <SectionTitle title="اجزای کانواس" level={2} />
          <p className="uikit-note">Canvas v1 · CB-CORE-02</p>
          <Card as="section" aria-label="شدت علائم">
            <SeverityScale
              label="گرگرفتگی"
              options={SEVERITY}
              value={severity}
              onChange={setSeverity}
            />
          </Card>
          <Card as="section" aria-label="مقیاس درد">
            <NumericScale
              label="چقدر درد داری؟"
              min={0}
              max={10}
              value={pain10}
              onChange={setPain10}
              locale={locale}
              minLabel="بدون درد"
              maxLabel="بدترین درد ممکن"
              variant="solid"
              tone="danger"
            />
          </Card>
          <Card as="section" aria-label="مقیاس خلق">
            <NumericScale
              label="غمگینی یا ناامیدی"
              min={1}
              max={6}
              value={mood6}
              onChange={setMood6}
              locale={locale}
              minLabel="اصلاً"
              maxLabel="خیلی شدید"
              tone="bloom"
            />
          </Card>
          <Card as="section" aria-label="مراحل درمان">
            <StepTimeline label="مراحل درمان" steps={IVF_STEPS} stateLabels={STEP_STATES} locale={locale} />
          </Card>
          <Card as="section" aria-label="روند خسارت">
            <StepTimeline
              label="روند خسارت"
              steps={CLAIM_STEPS}
              stateLabels={STEP_STATES}
              marker="dot"
              locale={locale}
            />
          </Card>
          <Card as="section" aria-label="تایمر تمرین" className="uikit-center">
            <CountdownRing
              elapsedMs={4 * 60_000 + 20_000}
              totalMs={10 * 60_000}
              mode="countdown"
              label="زمان باقی‌مانده تمرین"
              locale={locale}
              caption="انقباض"
              footer="ست ۲ از ۳"
              size={180}
              tone="data"
              glow
            />
          </Card>
          <Card as="section" aria-label="قرص این هفته">
            <WeekDots days={WEEK} todayIndex={4} locale={locale} label="قرص این هفته" stateLabels={WEEK_STATES} />
          </Card>
          <Card as="section" aria-label="پیشرفت">
            <ProgressBar value={3} max={8} label="برنامه کف لگن" valueLabel="۳ از ۸" valueText="هفته ۳ از ۸" hint="هفته بعد سطح ۲" tone="data" />
          </Card>
          <RadioCardGroup
            label="روش پیشگیری"
            value={method}
            onChange={setMethod}
            options={[
              { value: 'pill', title: 'قرص ترکیبی', description: 'هر روز یک قرص، ساعت ثابت', icon: 'pill', iconTone: 'brand' },
              { value: 'iud', title: 'آی‌یو‌دی', description: 'تا ۵ سال', icon: 'shield', iconTone: 'data' },
              { value: 'none', title: 'فعلاً هیچ‌کدام', disabled: true },
            ]}
          />
          <SearchField
            value={query}
            onValueChange={setQuery}
            label="جستجوی پزشک"
            clearLabel="پاک کردن"
            placeholder="نام پزشک یا تخصص"
          />
          <Card as="section" aria-label="رضایت‌ها">
            <Checkbox checked={consent} onCheckedChange={setConsent} label="قوانین را خواندم" />
            <Checkbox
              checked={share}
              onCheckedChange={setShare}
              label="اشتراک با پزشک"
              description="فقط خلاصه‌ی ۳ ماه اخیر"
            />
            <Checkbox checked={false} onCheckedChange={() => undefined} label="غیرفعال" disabled />
          </Card>
          <UrgentCard
            variant="note"
            title="اگر فکر آسیب به خودت داری، همین حالا با یکی از این شماره‌ها تماس بگیر."
            hotlines={HOTLINES}
            hotlinesLabel="شماره‌های کمک"
          />
          <UrgentCard
            title="درد شدید یک‌طرفه یا خون‌ریزی زیاد"
            action={<PrimaryButton icon="phone">تماس با ۱۱۵</PrimaryButton>}
            actions={<SecondaryButton>نوبت پزشک زنان</SecondaryButton>}
            hotlines={HOTLINES.slice(0, 1)}
            hotlinesLabel="شماره اورژانس"
          >
            اگر با این علائم همراه است، معطل نکن.
          </UrgentCard>

          <PrimaryButton onClick={() => setSheet(true)}>باز کردن برگه</PrimaryButton>
          <SecondaryButton>ویرایش</SecondaryButton>
          <SecondaryButton variant="text">هنوز نه</SecondaryButton>
        </div>
      </div>

      <AppSheet open={sheet} onClose={() => setSheet(false)} size="half" title="برگه نمونه">
        <InfoNote>BottomSheet همان AppSheet است با ظاهر Night &amp; Bloom.</InfoNote>
        <div className="uikit-sheet-cta">
          <PrimaryButton onClick={() => setSheet(false)}>ذخیره</PrimaryButton>
        </div>
      </AppSheet>
    </div>
  );
}
