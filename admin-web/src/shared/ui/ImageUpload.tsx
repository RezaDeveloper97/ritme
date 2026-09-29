'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useRef, useState, type DragEvent } from 'react';

import { formatNumber } from '@/shared/lib';

import { Button } from './Button';
import { Field } from './Field';
import { Icon } from './Icon';

const DEFAULT_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const;

/** Pure check, exported for tests: null when the file is acceptable. */
export function checkImageFile(
  file: { type: string; size: number },
  accept: readonly string[],
  maxMb: number,
): 'badType' | 'tooLarge' | null {
  if (!accept.includes(file.type)) return 'badType';
  if (file.size > maxMb * 1024 * 1024) return 'tooLarge';
  return null;
}

/**
 * Single image picker with preview. Holds nothing but the chosen File: the
 * screen puts it in its FormData (`fd.append('image', file)`) and sends it with
 * api.post(path, fd). `currentUrl` is the saved image (`image_url`) shown until
 * a new file is picked. Server-side limits still apply (admin-api.md §7).
 */
export function ImageUpload({
  label,
  value,
  currentUrl,
  onChange,
  accept = DEFAULT_TYPES,
  maxMb = 4,
  error,
  required,
  hint,
}: {
  label: string;
  value: File | null;
  currentUrl?: string | null;
  onChange: (file: File | null) => void;
  accept?: readonly string[];
  maxMb?: number;
  error?: string;
  required?: boolean;
  hint?: string;
}) {
  const t = useTranslations('upload');
  const locale = useLocale();
  const id = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [drag, setDrag] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(null);

  useEffect(() => {
    if (!value) {
      setPreview(null);
      return;
    }
    const url = URL.createObjectURL(value);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [value]);

  const pick = (file: File | undefined) => {
    if (!file) return;
    const problem = checkImageFile(file, accept, maxMb);
    if (problem) {
      setLocalError(problem === 'tooLarge' ? t('tooLarge', { size: formatNumber(maxMb, locale) }) : t('badType'));
      return;
    }
    setLocalError(null);
    onChange(file);
  };

  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDrag(false);
    pick(e.dataTransfer.files[0]);
  };

  const shown = preview ?? currentUrl ?? null;
  const types = accept.map((a) => a.replace('image/', '').toUpperCase()).join(', ');

  return (
    <Field
      label={label}
      required={required}
      htmlFor={id}
      error={localError ?? error}
      hint={hint ?? t('hint', { types, size: formatNumber(maxMb, locale) })}
    >
      <input
        ref={inputRef}
        id={id}
        type="file"
        accept={accept.join(',')}
        className="sr-only"
        onChange={(e) => {
          pick(e.target.files?.[0]);
          e.target.value = '';
        }}
      />
      {shown ? (
        <div className="flex flex-col items-start gap-2">
          {/* eslint-disable-next-line @next/next/no-img-element -- blob: previews and API-hosted files */}
          <img src={shown} alt={t('preview')} className="upload-preview" />
          <div className="flex gap-2">
            <Button size="sm" onClick={() => inputRef.current?.click()}>
              <Icon name="upload" size={15} />
              {t('replace')}
            </Button>
            {value ? (
              <Button size="sm" variant="ghost" onClick={() => onChange(null)}>
                {t('remove')}
              </Button>
            ) : null}
          </div>
        </div>
      ) : (
        <label
          htmlFor={id}
          className="dropzone"
          data-drag={drag || undefined}
          onDragOver={(e) => {
            e.preventDefault();
            setDrag(true);
          }}
          onDragLeave={() => setDrag(false)}
          onDrop={onDrop}
        >
          <Icon name="upload" size={22} />
          <span>{t('choose')}</span>
        </label>
      )}
    </Field>
  );
}
