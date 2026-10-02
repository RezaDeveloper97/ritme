'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { messageKeys } from '@/entities/message';
import {
  type DeliveryType,
  fieldError,
  MAX_BABY_COUNT,
  useActivatePostpartum,
  usePostpartum,
} from '@/entities/postpartum';
import { pregnancyKeys } from '@/entities/pregnancy';
import { lifeStageKeys, useLifeStage, writeLifeModeHint } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import {
  type DateParts,
  formatLongDate,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today as todayDate,
} from '@/shared/lib/date';
import {
  Card,
  CalendarPicker,
  ChipGroup,
  InfoNote,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SkyLayer,
} from '@/shared/ui';

import { birthDateProblem } from '../model/setup';

type DeliveryChoice = DeliveryType | 'none';
const DELIVERY_CHOICES: readonly DeliveryChoice[] = ['vaginal', 'cesarean', 'none'];

/**
 * `/postpartum/setup` — activation (B-N5-01 `POST /postpartum/activate`): birth
 * date, delivery type, baby count. Reached from the postpartum home's setup
 * prompt and from the pregnancy home's «زایمان کردم». A form: no bottom nav.
 */
export function PostpartumSetupPage() {
  const t = useTranslations('postpartum.setup');
  const tc = useTranslations('postpartum.common');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const life = useLifeStage();
  const overview = usePostpartum();
  const activate = useActivatePostpartum();

  const initialBirth = overview.data?.profile?.birthDate ?? null;
  const [birth, setBirth] = useState<string | null>(initialBirth);
  const [delivery, setDelivery] = useState<DeliveryChoice>('none');
  const [babies, setBabies] = useState(1);

  const fromPregnancy = life.data?.mode === 'pregnancy';
  const todayApi = toApiDate(todayDate());
  const problem = birth ? birthDateProblem(birth, todayApi) : null;
  const serverBirthError = fieldError(activate.error, 'birth_date');

  const submit = () => {
    if (!birth || problem || activate.isPending) return;
    activate.mutate(
      { birthDate: birth, deliveryType: delivery === 'none' ? null : delivery, babyCount: babies },
      {
        onSuccess: () => {
          writeLifeModeHint('postpartum');
          void queryClient.invalidateQueries({ queryKey: lifeStageKeys.all });
          void queryClient.invalidateQueries({ queryKey: messageKeys.all });
          void queryClient.invalidateQueries({ queryKey: pregnancyKeys.all });
          router.replace('/postpartum');
        },
      },
    );
  };

  const onPick = (parts: DateParts) => setBirth(toApiDate(partsToDate(parts, locale)));

  return (
    <div className="view pp-screen pp-form-page">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={t('subtitle')}
          onBack={() => router.push(fromPregnancy ? '/pregnancy' : '/postpartum')}
          backLabel={tc('back')}
        />
        <div className="pp-form">
          <Card as="section" className="pp-field" aria-labelledby="pp-birth">
            <h2 id="pp-birth" className="pp-field-title">
              {t('birthDate')}
            </h2>
            <p className="pp-field-value" aria-live="polite">
              {birth ? formatLongDate(fromApiDate(birth), locale) : t('pickDate')}
            </p>
            <CalendarPicker value={birth ? toParts(fromApiDate(birth), locale) : null} onSelect={onPick} />
            {(problem || serverBirthError) && (
              <p className="pp-error" role="alert">
                {problem ? t(problem) : serverBirthError}
              </p>
            )}
          </Card>

          <Card as="section" className="pp-field" aria-labelledby="pp-delivery">
            <h2 id="pp-delivery" className="pp-field-title">
              {t('deliveryType')}
            </h2>
            <p className="pp-field-hint">{t('deliveryHint')}</p>
            <ChipGroup label={t('deliveryType')}>
              {DELIVERY_CHOICES.map((c) => (
                <PillChip key={c} pressed={delivery === c} onPressedChange={() => setDelivery(c)}>
                  {t(c === 'none' ? 'noSay' : c)}
                </PillChip>
              ))}
            </ChipGroup>
          </Card>

          <Card as="section" className="pp-field">
            <NumberStepper
              label={t('babyCount')}
              value={babies}
              onChange={setBabies}
              min={1}
              max={MAX_BABY_COUNT}
              decrementLabel={t('decrease')}
              incrementLabel={t('increase')}
              locale={locale}
            />
          </Card>

          {fromPregnancy && <InfoNote>{t('fromPregnancy')}</InfoNote>}
          <InfoNote icon="shield">{t('privacy')}</InfoNote>

          {activate.isError && !serverBirthError && (
            <p className="pp-error" role="alert">
              {getApiSaveErrorMessage(activate.error, t('saveError'))}
            </p>
          )}
          <PrimaryButton disabled={!birth || Boolean(problem)} loading={activate.isPending} onClick={submit}>
            {t('submit')}
          </PrimaryButton>
        </div>
      </div>
    </div>
  );
}
