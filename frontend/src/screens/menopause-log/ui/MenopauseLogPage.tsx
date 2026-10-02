'use client';

import { useSearchParams } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';

import { LogDay } from '@/features/log-day';
import { VoiceLogPanel } from '@/features/voice-log';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { SkyLayer } from '@/shared/ui';
import { BodyMap } from '@/widgets/body-map';

/**
 * «ثبت علائم» (`/menopause/log[?date=]`, CB-MENO-06, nbl_Meno_Log): bloom's log sheet v2 as a full page
 * on its menopause preset — grouped severity rows, bleeding, triggers, «با صدا بگو» → the voice tab,
 * one save (offline → outbox). The home «ثبت علائم امروز» / «امروز · ثبت» open it. A form: no bottom nav.
 */
export function MenopauseLogPage() {
  const t = useTranslations('menopause.log');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const param = useSearchParams().get('date');
  const todayKey = toApiDate(today());
  const date = param && /^\d{4}-\d{2}-\d{2}$/.test(param) && param <= todayKey ? param : todayKey;
  const dayText = formatDayMonth(fromApiDate(date), locale);
  const goHome = () => router.push('/home');
  return (
    <div className="view lday-page mlog-page">
      <SkyLayer />
      <div className="scroll lday-page-scroll">
        <LogDay
          variant="page"
          initialDate={date}
          mode="menopause"
          dateStrip={false}
          pageHeader={{ title: t('title'), subtitle: date === todayKey ? t('today', { date: dayText }) : dayText }}
          BodyMap={BodyMap}
          VoiceLog={VoiceLogPanel}
          onClose={goHome}
          onSaved={goHome}
        />
      </div>
    </div>
  );
}
