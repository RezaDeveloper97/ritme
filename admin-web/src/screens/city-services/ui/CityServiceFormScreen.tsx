'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, toIntOrNull } from '@/shared/lib';
import { Button, FormPage, ImageUpload, LoadGate, PageHeader, Panel, Select, Switch, TextArea, TextInput, toast, useNotifyError } from '@/shared/ui';

import { listingsApi, useAddPhoto, useCityServiceOptions, useRemovePhoto, type CityServiceOptions, type Listing } from '../api/city-services';

import { ListingDirectoryPanels } from './ListingDirectoryPanels';

/** /city-services/new and /city-services/:id — one venue (B-N10-05). */
export function CityServiceFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('cityServices');
  const detail = listingsApi.useDetail(id);
  const options = useCityServiceOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/city-services" backLabel={t('backToList')} />}
    >
      {() => (options.data ? <ListingForm id={id} row={detail.data?.listing ?? null} options={options.data} /> : null)}
    </LoadGate>
  );
}

function ListingForm({ id, row, options }: { id: number | null; row: Listing | null; options: CityServiceOptions }) {
  const t = useTranslations('cityServices');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = listingsApi.useSave(id);
  const err = (name: string) => fieldError(save.error, name);

  const [name, setName] = useState(row?.name ?? '');
  const [categoryId, setCategoryId] = useState(String(row?.category_id ?? options.categories[0]?.id ?? ''));
  const [cityId, setCityId] = useState(String(row?.city_id ?? options.cities[0]?.id ?? ''));
  const [district, setDistrict] = useState(row?.district ?? '');
  const [address, setAddress] = useState(row?.address ?? '');
  const [phone, setPhone] = useState(row?.phone ?? '');
  const [description, setDescription] = useState(row?.description ?? '');
  const [priceText, setPriceText] = useState(row?.price_text ?? '');
  const [hours, setHours] = useState(row?.hours_text ?? '');
  const [mode, setMode] = useState(row?.booking_mode ?? 'phone');
  const [url, setUrl] = useState(row?.booking_url ?? '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);

  const submit = () =>
    save.mutate(
      {
        name: name.trim(),
        category_id: toIntOrNull(categoryId),
        city_id: toIntOrNull(cityId),
        district: blankToNull(district),
        address: blankToNull(address),
        phone: blankToNull(phone),
        description: blankToNull(description),
        price_text: blankToNull(priceText),
        hours_text: blankToNull(hours),
        booking_mode: mode,
        booking_url: mode === 'link' ? blankToNull(url) : null,
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/city-services');
        },
        onError: notifyError,
      },
    );

  const named = (c: { id: number; slug: string; name: Record<string, string> }) => ({ value: String(c.id), label: localize(c.name) || c.slug });

  return (
    <div className="flex flex-col gap-4">
      <FormPage
        title={id === null ? t('new') : row?.name || t('edit')}
        backHref="/city-services"
        backLabel={t('backToList')}
        onSubmit={submit}
        submitLabel={id === null ? t('create') : tc('saveChanges')}
        saving={save.isPending}
      >
        <TextInput label={t('name')} value={name} onChange={(e) => setName(e.target.value)} maxLength={160} required error={err('name')} />
        <div className="form-grid">
          <Select label={t('category')} value={categoryId} onChange={(e) => setCategoryId(e.target.value)} options={options.categories.map(named)} required error={err('category_id')} />
          <Select label={t('city')} value={cityId} onChange={(e) => setCityId(e.target.value)} options={options.cities.map(named)} required error={err('city_id')} hint={options.cities.length === 0 ? t('noCities') : undefined} />
          <TextInput label={t('district')} value={district} onChange={(e) => setDistrict(e.target.value)} maxLength={120} error={err('district')} />
          <TextInput label={t('phone')} value={phone} onChange={(e) => setPhone(e.target.value)} dir="ltr" maxLength={32} error={err('phone')} />
        </div>
        <TextInput label={t('address')} value={address} onChange={(e) => setAddress(e.target.value)} maxLength={300} error={err('address')} />
        <TextArea label={t('description')} value={description} onChange={(e) => setDescription(e.target.value)} rows={4} maxLength={5000} error={err('description')} />
        <div className="form-grid">
          <TextInput label={t('priceText')} hint={t('priceTextHint')} value={priceText} onChange={(e) => setPriceText(e.target.value)} maxLength={120} error={err('price_text')} />
          <TextInput label={t('hours')} hint={t('hoursHint')} value={hours} onChange={(e) => setHours(e.target.value)} maxLength={200} error={err('hours_text')} />
          <Select
            label={t('bookingMode')}
            hint={t('bookingModeHint')}
            value={mode}
            onChange={(e) => setMode(e.target.value)}
            options={(options.admin_booking_modes.length > 0 ? options.admin_booking_modes : options.booking_modes).map((m) => ({ value: m, label: t(`modes.${m}` as 'modes.link') }))}
            error={err('booking_mode')}
          />
          {mode === 'link' ? (
            <TextInput label={t('bookingUrl')} hint={t('bookingUrlHint')} value={url} onChange={(e) => setUrl(e.target.value)} dir="ltr" maxLength={1000} required error={err('booking_url')} />
          ) : null}
          <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
          <Switch label={tc('isActive')} checked={active} onChange={setActive} />
        </div>
      </FormPage>
      {row ? <PhotosPanel listing={row} options={options} /> : <p className="panel-note">{t('photosAfterSave')}</p>}
      {row ? <ListingDirectoryPanels listing={row} options={options} /> : null /* CB-DIR-05 */}
    </div>
  );
}

/** Photos of a saved listing: add one at a time, remove any; the first is the card photo. */
function PhotosPanel({ listing, options }: { listing: Listing; options: CityServiceOptions }) {
  const t = useTranslations('cityServices');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const add = useAddPhoto(listing.id);
  const remove = useRemovePhoto(listing.id);
  const [file, setFile] = useState<File | null>(null);
  const img = options.image;
  const full = listing.photos.length >= options.max_photos;

  return (
    <Panel title={t('photos')}>
      {listing.photo_urls.length > 0 ? (
        <ul className="flex flex-wrap gap-3" aria-label={t('photos')}>
          {listing.photo_urls.map((u, i) => (
            <li key={u} className="flex flex-col items-center gap-1">
              {/* eslint-disable-next-line @next/next/no-img-element -- API-hosted upload */}
              <img src={u} alt="" className="thumb" loading="lazy" />
              <Button size="sm" loading={remove.isPending && remove.variables === i} onClick={() => remove.mutate(i, { onSuccess: () => toast.success(t('photoRemoved')), onError: notifyError })}>
                {t('removePhoto')}
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="panel-note">{t('noPhotos')}</p>
      )}
      {full ? (
        <p className="panel-note">{t('photosFull', { max: formatNumber(options.max_photos, locale) })}</p>
      ) : (
        <div className="flex flex-col gap-2">
          <ImageUpload
            label={t('addPhoto')}
            value={file}
            onChange={setFile}
            maxMb={Math.round(img.max_kb / 1024)}
            accept={img.types.map((type) => `image/${type}`)}
            hint={t('photoHint', { minWidth: formatNumber(img.min_width, locale), minHeight: formatNumber(img.min_height, locale), size: formatNumber(Math.round(img.max_kb / 1024), locale) })}
            error={fieldError(add.error, 'photo')}
          />
          <div>
            <Button
              variant="primary"
              size="sm"
              disabled={!file}
              loading={add.isPending}
              onClick={() => file && add.mutate(file, { onSuccess: () => { setFile(null); toast.success(t('photoAdded')); }, onError: notifyError })}
            >
              {t('uploadPhoto')}
            </Button>
          </div>
        </div>
      )}
    </Panel>
  );
}
