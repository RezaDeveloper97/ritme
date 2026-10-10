'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { useCurrentAdmin } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, toIntOrNull, toFormData } from '@/shared/lib';
import {
  Button,
  CheckboxGrid,
  FormPage,
  ImageUpload,
  LoadGate,
  PageHeader,
  Panel,
  Select,
  Switch,
  TextInput,
  TranslatableField,
  confirm,
  toast,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import {
  doctorsApi,
  useCatalogOptions,
  useDoctorWrite,
  type CatalogOption,
  type Doctor,
  type DoctorOptions,
} from '../api/telemed';
import { DoctorSchedule } from './DoctorSchedule';

/** /telemed/doctors/new and /telemed/doctors/:id — profile, photo, schedule (admin-api.md §19). */
export function DoctorFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('telemed.doctor');
  const detail = doctorsApi.useDetail(id);
  const options = doctorsApi.useOptions();
  const specialties = useCatalogOptions('telemed_specialties');
  const cities = useCatalogOptions('telemed_cities');
  const insurers = useCatalogOptions('telemed_insurers');
  return (
    <LoadGate
      queries={id === null ? [options, specialties, cities, insurers] : [detail, options, specialties, cities, insurers]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/telemed/doctors" backLabel={t('back')} />}
    >
      {() =>
        options.data && specialties.data && cities.data && insurers.data ? (
          <DoctorForm
            id={id}
            row={detail.data?.doctor ?? null}
            options={options.data}
            specialties={specialties.data.items}
            cities={cities.data.items}
            insurers={insurers.data.items}
          />
        ) : null
      }
    </LoadGate>
  );
}

const num = (v: number | null | undefined) => (v === null || v === undefined ? '' : String(v));

function DoctorForm({
  id,
  row,
  options,
  specialties,
  cities,
  insurers,
}: {
  id: number | null;
  row: Doctor | null;
  options: DoctorOptions;
  specialties: CatalogOption[];
  cities: CatalogOption[];
  insurers: CatalogOption[];
}) {
  const t = useTranslations('telemed.doctor');
  const tc = useTranslations('crud');
  const localize = useLocalized();
  const router = useRouter();
  const notifyError = useNotifyError();
  const isSuper = useCurrentAdmin()?.role === 'super';
  const save = doctorsApi.useSave(id);
  const remove = doctorsApi.useRemove();
  const photoWrite = useDoctorWrite(id ?? 0);

  const [kind, setKind] = useState(row?.kind ?? options.kinds[0] ?? 'doctor');
  const [name, setName] = useState<Translations>(row?.name ?? {});
  const [headline, setHeadline] = useState<Translations>(row?.headline ?? {});
  const [bio, setBio] = useState<Translations>(row?.bio ?? {});
  const [specialty, setSpecialty] = useState(row?.specialty ?? specialties[0]?.code ?? '');
  const [city, setCity] = useState(row?.city ?? '');
  const [licence, setLicence] = useState(row?.licence_no ?? '');
  const [experience, setExperience] = useState(num(row?.experience_years));
  const [response, setResponse] = useState(num(row?.response_minutes));
  const [insurerCodes, setInsurerCodes] = useState<string[]>(row?.insurers ?? []);
  const [active, setActive] = useState(row?.is_active ?? true);
  const [sortOrder, setSortOrder] = useState(num(row?.sort_order ?? options.next_sort_order));
  const [photo, setPhoto] = useState<File | null>(null);

  const err = (field: string) => fieldError(save.error, field);
  const label = (o: CatalogOption) => localize(o.title) || o.code;
  const clean = (v: Translations) => {
    const out: Translations = {};
    for (const [k, text] of Object.entries(v)) if (text.trim()) out[k] = text;
    return Object.keys(out).length ? out : null;
  };

  const uploadPhoto = () => {
    if (!photo || id === null) return;
    photoWrite.mutate(
      { method: 'post', path: 'photo', body: toFormData({ photo }) },
      {
        onSuccess: () => {
          setPhoto(null);
          toast.success(tc('saved'));
        },
        onError: notifyError,
      },
    );
  };

  const submit = () =>
    save.mutate(
      {
        kind,
        name,
        headline: clean(headline),
        bio: clean(bio),
        specialty,
        city: blankToNull(city),
        licence_no: licence.trim(),
        experience_years: toIntOrNull(experience),
        response_minutes: toIntOrNull(response),
        insurers: insurerCodes,
        is_active: active,
        sort_order: toIntOrNull(sortOrder) ?? undefined,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          if (id === null) router.push('/telemed/doctors');
        },
        onError: notifyError,
      },
    );

  const onDelete = async () => {
    if (id === null) return;
    if (!(await confirm({ message: t('confirmDelete'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
    remove.mutate(id, {
      onSuccess: () => {
        toast.success(tc('deleted'));
        router.push('/telemed/doctors');
      },
      onError: notifyError,
    });
  };

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.name) || t('edit')}
      backHref="/telemed/doctors"
      backLabel={t('back')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
      headerActions={
        id !== null && isSuper ? (
          <Button variant="danger" loading={remove.isPending} onClick={onDelete}>
            {tc('delete')}
          </Button>
        ) : null
      }
      after={id !== null && row ? <DoctorSchedule doctor={row} options={options} /> : null}
    >
      <Select
        label={t('kind')}
        value={kind}
        onChange={(e) => setKind(e.target.value)}
        error={err('kind')}
        options={options.kinds.map((k) => ({ value: k, label: t(`kindName.${k}` as 'kindName.doctor') }))}
      />
      <TranslatableField name="name" label={t('name')} value={name} onChange={setName} required maxLength={options.limits.name} errors={undefined} />
      <TranslatableField name="headline" label={t('headline')} value={headline} onChange={setHeadline} maxLength={options.limits.headline} />
      <TranslatableField name="bio" label={t('bio')} value={bio} onChange={setBio} kind="textarea" maxLength={options.limits.bio} />
      <div className="grid gap-4 sm:grid-cols-2">
        <Select
          label={t('specialty')}
          value={specialty}
          onChange={(e) => setSpecialty(e.target.value)}
          error={err('specialty')}
          options={specialties.map((o) => ({ value: o.code, label: label(o) }))}
        />
        <Select
          label={t('city')}
          value={city}
          onChange={(e) => setCity(e.target.value)}
          error={err('city')}
          options={[{ value: '', label: t('noCity') }, ...cities.map((o) => ({ value: o.code, label: label(o) }))]}
        />
        <TextInput label={t('licence')} value={licence} maxLength={options.limits.licence_no} dir="ltr" required error={err('licence_no')} onChange={(e) => setLicence(e.target.value)} />
        <TextInput label={t('experience')} type="number" value={experience} error={err('experience_years')} onChange={(e) => setExperience(e.target.value)} />
        <TextInput label={t('response')} hint={t('responseHint')} type="number" value={response} error={err('response_minutes')} onChange={(e) => setResponse(e.target.value)} />
        <TextInput label={t('sortOrder')} type="number" value={sortOrder} error={err('sort_order')} onChange={(e) => setSortOrder(e.target.value)} />
      </div>
      {insurers.length ? (
        <CheckboxGrid
          label={t('insurers')}
          options={insurers.map((o) => ({ value: o.code, label: label(o) }))}
          value={insurerCodes}
          onChange={setInsurerCodes}
          error={err('insurers')}
        />
      ) : null}
      <Switch label={t('active')} hint={t('activeHint')} checked={active} onChange={setActive} />
      {id !== null ? (
        <Panel title={t('photo')} className="shadow-none">
          <div className="flex flex-col gap-3">
            <ImageUpload label={t('photo')} value={photo} currentUrl={row?.photo_url} onChange={setPhoto} hint={t('photoHint')} />
            <div className="flex gap-2">
              <Button disabled={!photo} loading={photoWrite.isPending} onClick={uploadPhoto}>
                {t('uploadPhoto')}
              </Button>
              {row?.photo_url ? (
                <Button
                  variant="ghost"
                  onClick={() => photoWrite.mutate({ method: 'delete', path: 'photo' }, { onSuccess: () => toast.success(tc('saved')), onError: notifyError })}
                >
                  {t('removePhoto')}
                </Button>
              ) : null}
            </div>
          </div>
        </Panel>
      ) : (
        <p className="m-0 text-sm text-muted">{t('afterCreate')}</p>
      )}
    </FormPage>
  );
}
