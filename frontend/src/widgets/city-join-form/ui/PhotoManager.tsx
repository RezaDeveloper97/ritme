'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type ChangeEvent, useRef, useState } from 'react';

import { type JoinApplication, useAddJoinPhoto, useRemoveJoinPhoto } from '@/entities/city-service';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, ProgressBar, SecondaryButton } from '@/shared/ui';

import { fileProblem } from '../model/join';
import { useErrorText } from './parts';
import '@/shared/ui/styles/cjn.css';

/**
 * The photo grid of the application (step 2 and the done screen's «add more photos»): each picked file goes to
 * `POST …/photos` one after another with a progress bar; thumbnails come back from the API as 5-minute signed links
 * (the photos stay private until the team approves the application, then they turn public). The application query
 * is never stale, so every visit gets fresh links. The API re-encodes to WebP without EXIF / GPS. Nothing is kept on
 * the device.
 */
export function PhotoManager({ app }: { app: JoinApplication }) {
  const t = useTranslations('cityJoin');
  const locale = useLocale() as Locale;
  const errorText = useErrorText();
  const add = useAddJoinPhoto(app.id);
  const remove = useRemoveJoinPhoto(app.id);
  const input = useRef<HTMLInputElement>(null);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { min_photos: min, max_photos: max } = app.requirements;
  const num = (n: number) => formatNumber(n, locale);
  const full = app.photos.length >= max;

  const onPick = async (event: ChangeEvent<HTMLInputElement>) => {
    const picked = Array.from(event.target.files ?? []);
    event.target.value = '';
    setError(null);
    let count = app.photos.length;
    for (const file of picked) {
      if (count >= max) {
        setError(t('errors.tooMany', { max: num(max) }));
        break;
      }
      const problem = fileProblem(file, 'photo');
      if (problem) {
        setError(t(`errors.${problem}`));
        continue;
      }
      try {
        setProgress(0);
        await add.mutateAsync({ file, onProgress: setProgress });
        count += 1;
      } catch (e) {
        setError(errorText(e));
        break;
      }
    }
    setProgress(null);
  };

  return (
    <div className="cjn-photos">
      <div className="cjn-head">
        <h3 className="cjn-label">{t('place.photos')}</h3>
        <SecondaryButton block={false} icon="plus" disabled={full || progress !== null} onClick={() => input.current?.click()}>
          {t('place.add')}
        </SecondaryButton>
      </div>
      <p className="cjn-hint">{t('place.photosHint', { min: num(min) })}</p>
      <input
        ref={input}
        className="sr-only"
        type="file"
        accept="image/jpeg,image/png,image/webp"
        multiple
        tabIndex={-1}
        aria-hidden
        onChange={(e) => void onPick(e)}
      />
      {app.photos.length > 0 ? (
        <ul className="cjn-grid">
          {app.photos.map((p, i) => (
            <li key={p.id} className="cjn-thumb">
              {/* eslint-disable-next-line @next/next/no-img-element -- the API's WebP (signed link until approval), shown as is */}
              <img src={p.url} alt={t('place.photoAlt', { n: num(i + 1) })} className="cjn-img" loading="lazy" />
              <button
                type="button"
                className="cjn-x"
                aria-label={t('place.removePhoto', { n: num(i + 1) })}
                disabled={remove.isPending}
                onClick={() => remove.mutate(p.id, { onError: (e) => setError(errorText(e)) })}
              >
                <Icon name="x" size={12} strokeWidth={3} />
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      <p className="cjn-count" data-ok={app.photos.length >= min}>
        {t('place.photoCount', { n: num(app.photos.length), min: num(min), max: num(max) })}
      </p>
      {progress !== null ? (
        <ProgressBar value={Math.round(progress * 100)} max={100} label={t('place.uploading', { percent: num(Math.round(progress * 100)) })} />
      ) : null}
      {error ? (
        <p role="alert" className="cjn-err">
          {error}
        </p>
      ) : null}
    </div>
  );
}
