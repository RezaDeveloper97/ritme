'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, isoToLocalInput, localInputToApi, toFormData, toIntOrNull } from '@/shared/lib';
import {
  CheckboxGrid,
  FormPage,
  ImageUpload,
  LoadGate,
  PageHeader,
  Select,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { bannersApi, useBannerStats, type Banner, type BannerOptions } from '../api/banners';

/** /banners/new and /banners/:id (Blade banners.form). */
export function BannerFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('banners');
  const detail = bannersApi.useDetail(id);
  const options = bannersApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/banners" backLabel={t('backToList')} />}
    >
      {() => (options.data ? <BannerForm id={id} row={detail.data?.banner ?? null} options={options.data} /> : null)}
    </LoadGate>
  );
}

function BannerForm({ id, row, options }: { id: number | null; row: Banner | null; options: BannerOptions }) {
  const t = useTranslations('banners');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = bannersApi.useSave(id);

  const [image, setImage] = useState<File | null>(null);
  const names: Record<'adSlots' | 'kinds' | 'modeNames', Record<string, string>> = {
    adSlots: { ad_slider: t('adSlots.ad_slider'), ad_square: t('adSlots.ad_square'), ad_strip: t('adSlots.ad_strip') },
    kinds: { banner: t('kinds.banner'), ad: t('kinds.ad') },
    modeNames: {
      cycle: t('modeNames.cycle'), ttc: t('modeNames.ttc'), pregnancy: t('modeNames.pregnancy'),
      postpartum: t('modeNames.postpartum'), menopause: t('modeNames.menopause'), teen: t('modeNames.teen'),
      companion: t('modeNames.companion'),
    },
  };
  const [kind, setKind] = useState(row?.kind ?? 'banner');
  const isAd = kind === 'ad';
  const slotOptions = isAd
    ? options.ad_slots.map((v) => ({ value: v, label: names.adSlots[v] ?? v }))
    : options.positions;
  const [position, setPosition] = useState(row?.position ?? options.positions[0]?.value ?? '');
  const [advertiser, setAdvertiser] = useState(row?.advertiser ?? '');
  const [ctaLabel, setCtaLabel] = useState<Translations>(row?.cta_label ?? {});
  const [modes, setModes] = useState<string[]>(row?.modes ?? []);
  const stats = useBannerStats(id, isAd);
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [linkType, setLinkType] = useState(row?.link_type ?? '');
  const [linkUrl, setLinkUrl] = useState(row?.link_url ?? '');
  const [startsAt, setStartsAt] = useState(isoToLocalInput(row?.starts_at));
  const [endsAt, setEndsAt] = useState(isoToLocalInput(row?.ends_at));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);
  const img = options.image;
  const n = (v: number) => formatNumber(v, locale);

  const submit = () =>
    save.mutate(
      toFormData({
        image: image ?? undefined,
        position: isAd && !options.ad_slots.includes(position) ? (options.ad_slots[0] ?? position) : !isAd && options.ad_slots.includes(position) ? (options.positions[0]?.value ?? position) : position,
        kind,
        advertiser: isAd ? blankToNull(advertiser) : null,
        cta_label: isAd ? ctaLabel : undefined,
        modes: isAd ? modes : [],
        sort_order: toIntOrNull(sortOrder) ?? 0,
        title,
        link_type: blankToNull(linkType),
        link_url: blankToNull(linkUrl),
        starts_at: localInputToApi(startsAt),
        ends_at: localInputToApi(endsAt),
        is_active: active,
      }),
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/banners');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/banners"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? t('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <ImageUpload
        label={t('image')}
        value={image}
        currentUrl={row?.image_url}
        onChange={setImage}
        maxMb={Math.round(img.max_kb / 1024)}
        accept={img.types.map((type) => `image/${type}`)}
        required={id === null}
        hint={t('imageHint', {
          width: n(img.recommended_width),
          height: n(img.recommended_height),
          minWidth: n(img.min_width),
          minHeight: n(img.min_height),
          size: n(Math.round(img.max_kb / 1024)),
        })}
        error={err('image')}
      />
      <div className="form-grid">
        <Select
          label={t('kind')}
          value={kind}
          onChange={(e) => setKind(e.target.value)}
          options={options.kinds.map((v) => ({ value: v, label: names.kinds[v] ?? v }))}
        />
        <Select
          label={t('position')}
          value={slotOptions.some((o) => o.value === position) ? position : (slotOptions[0]?.value ?? '')}
          onChange={(e) => setPosition(e.target.value)}
          options={slotOptions}
          required
          error={err('position')}
        />
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
      </div>
      {isAd && (
        <>
          <TextInput label={t('advertiser')} hint={t('advertiserHint')} value={advertiser} onChange={(e) => setAdvertiser(e.target.value)} maxLength={120} error={err('advertiser')} />
          <TranslatableField name="cta_label" label={t('ctaLabel')} value={ctaLabel} onChange={setCtaLabel} maxLength={60} errors={fieldErrorsOf(save.error)} />
          <CheckboxGrid
            label={t('modes')}
            hint={t('modesHint')}
            options={options.modes.map((v) => ({ value: v, label: names.modeNames[v] ?? v }))}
            value={modes}
            onChange={setModes}
          />
        </>
      )}
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} maxLength={255} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <Select
          label={t('linkType')}
          value={linkType}
          onChange={(e) => setLinkType(e.target.value)}
          options={[{ value: '', label: t('noLink') }, ...options.link_types]}
          error={err('link_type')}
        />
        <TextInput
          label={t('linkUrl')}
          hint={t('linkUrlHint')}
          value={linkUrl}
          onChange={(e) => setLinkUrl(e.target.value)}
          dir="ltr"
          maxLength={1000}
          error={err('link_url')}
        />
        <TextInput
          label={t('startsAt')}
          hint={t('startsAtHint')}
          type="datetime-local"
          value={startsAt}
          onChange={(e) => setStartsAt(e.target.value)}
          error={err('starts_at')}
        />
        <TextInput
          label={t('endsAt')}
          hint={t('endsAtHint')}
          type="datetime-local"
          value={endsAt}
          min={startsAt || undefined}
          onChange={(e) => setEndsAt(e.target.value)}
          error={err('ends_at')}
        />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
      {isAd && id !== null && stats.data && (
        <section aria-label={t('stats.title')}>
          <h3>{t('stats.title')}</h3>
          <p>
            {t('stats.totals', {
              impressions: n(stats.data.totals.impressions),
              clicks: n(stats.data.totals.clicks),
              ctr: n(Math.round(stats.data.totals.ctr * 1000) / 10),
            })}
          </p>
          <ul>
            {stats.data.items.map((d) => (
              <li key={d.day}>{t('stats.day', { day: d.day, impressions: n(d.impressions), clicks: n(d.clicks) })}</li>
            ))}
          </ul>
        </section>
      )}
    </FormPage>
  );
}
