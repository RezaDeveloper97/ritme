'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { blankToNull, formatNumber, toFormData, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  CheckboxGrid,
  FormPage,
  ImageUpload,
  LoadGate,
  PageHeader,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { articlesApi, type Article, type PhaseOption } from '../api/articles';

/** /articles/new and /articles/:id (Blade articles.form). */
export function ArticleFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('articles');
  const detail = articlesApi.useDetail(id);
  const options = articlesApi.useOptions();
  return (
    <LoadGate
      queries={[detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/articles" backLabel={t('backToList')} />}
    >
      {() => (
        <ArticleForm
          id={id}
          article={detail.data?.article ?? null}
          phases={detail.data?.options.phases ?? options.data?.phases ?? []}
          maxWidth={options.data?.max_image_width}
          maxKb={options.data?.max_image_kb ?? 8192}
        />
      )}
    </LoadGate>
  );
}

function ArticleForm({
  id,
  article,
  phases,
  maxWidth,
  maxKb,
}: {
  id: number | null;
  article: Article | null;
  phases: PhaseOption[];
  maxWidth?: number;
  maxKb: number;
}) {
  const t = useTranslations('articles');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = articlesApi.useSave(id);

  const [slug, setSlug] = useState(article?.slug ?? '');
  const [category, setCategory] = useState(article?.category ?? '');
  const [title, setTitle] = useState<Translations>(article?.title ?? {});
  const [excerptText, setExcerpt] = useState<Translations>(article?.excerpt ?? {});
  const [body, setBody] = useState<Translations>(article?.body ?? {});
  const [cyclePhases, setCyclePhases] = useState<string[]>(article?.cycle_phases ?? []);
  const [readTime, setReadTime] = useState(article?.read_time_minutes?.toString() ?? '');
  const [image, setImage] = useState<File | null>(null);
  const [removeImage, setRemoveImage] = useState(false);
  const [imageUrl, setImageUrl] = useState(article?.image_url ?? '');
  const [sortOrder, setSortOrder] = useState(String(article?.sort_order ?? 0));
  const [published, setPublished] = useState(article?.is_published ?? false);

  const errors = fieldErrorsOf(save.error);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () => {
    const body_ = toFormData({
      slug: slug.trim(),
      category: blankToNull(category),
      title,
      excerpt: excerptText,
      body,
      cycle_phases: cyclePhases,
      read_time_minutes: toIntOrNull(readTime),
      image_url: blankToNull(imageUrl),
      sort_order: toIntOrNull(sortOrder) ?? 0,
      is_published: published,
      remove_image: id !== null && removeImage && !image ? true : undefined,
      image: image ?? undefined,
    });
    save.mutate(body_, {
      onSuccess: () => {
        toast.success(id === null ? tc('created') : tc('saved'));
        router.push('/articles');
      },
      onError: notifyError,
    });
  };

  const phaseOptions = phases.map((p) => ({
    value: p.value,
    label: p.label,
    extra: p.legacy ? <Badge tone="amber">{tc('legacy')}</Badge> : undefined,
  }));
  const maxMb = Math.round(maxKb / 1024);

  return (
    <FormPage
      title={id === null ? t('new') : localize(article?.title) || t('edit')}
      backHref="/articles"
      backLabel={t('backToList')}
      meta={article ? (article.is_published ? <Badge tone="green">{t('published')}</Badge> : <Badge tone="amber">{t('draft')}</Badge>) : null}
      onSubmit={submit}
      submitLabel={id === null ? t('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <TextInput
          label={t('slug')}
          hint={t('slugHint')}
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          required
          maxLength={255}
          dir="ltr"
          error={err('slug')}
        />
        <TextInput
          label={t('category')}
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          maxLength={255}
          error={err('category')}
        />
      </div>
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} required maxLength={255} errors={errors} />
      <TranslatableField name="excerpt" label={t('excerpt')} value={excerptText} onChange={setExcerpt} kind="rich" errors={errors} />
      <TranslatableField name="body" label={t('body')} value={body} onChange={setBody} kind="rich" errors={errors} />
      <CheckboxGrid
        label={t('phases')}
        hint={t('phasesHint')}
        options={phaseOptions}
        value={cyclePhases}
        onChange={setCyclePhases}
        bulk
        error={err('cycle_phases') ?? Object.entries(errors ?? {}).find(([k]) => k.startsWith('cycle_phases.'))?.[1][0]}
      />
      <div className="form-grid">
        <div className="flex flex-col gap-3">
          <ImageUpload
            label={t('cover')}
            value={image}
            currentUrl={removeImage ? null : article?.cover_url}
            onChange={(file) => {
              setImage(file);
              if (file) setRemoveImage(false);
            }}
            maxMb={maxMb}
            hint={t('coverHint', { size: formatNumber(maxMb, locale), width: formatNumber(maxWidth ?? 1080, locale) })}
            error={err('image')}
          />
          {article?.image_path && !image ? (
            <Switch label={t('removeCover')} checked={removeImage} onChange={setRemoveImage} />
          ) : null}
        </div>
        <TextInput
          label={t('imageUrl')}
          hint={t('imageUrlHint')}
          value={imageUrl}
          onChange={(e) => setImageUrl(e.target.value)}
          maxLength={1000}
          dir="ltr"
          error={err('image_url')}
        />
      </div>
      <div className="form-grid">
        <TextInput
          label={t('readTime')}
          type="number"
          min={1}
          max={120}
          value={readTime}
          onChange={(e) => setReadTime(e.target.value)}
          error={err('read_time_minutes')}
        />
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
        <Switch label={t('publish')} hint={t('publishHint')} checked={published} onChange={setPublished} />
      </div>
    </FormPage>
  );
}
