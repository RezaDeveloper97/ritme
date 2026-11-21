'use client';

import { useTranslations } from 'next-intl';
import { useRef, useState } from 'react';

import { uploadPhoto, useAssistantError } from '@/entities/assistant';
import { Icon } from '@/shared/ui';
import '@/shared/ui/styles/as.css';

const MAX_RUNES = 2000;
const MAX_PHOTO = 5 * 1024 * 1024;

interface Props {
  busy: boolean;
  disabled?: boolean;
  /** Text restored after a failed send (the user can edit and resend). */
  restore?: string | null;
  initial?: string;
  onSend: (text: string, photoId: number | null) => void;
  onStop: () => void;
}

/** Composer: growing textarea, photo attach (JPEG/PNG/WebP ≤ 5 MB), send / stop. Enter sends, Shift+Enter breaks the line. */
export function Composer({ busy, disabled, restore, initial = '', onSend, onStop }: Props) {
  const t = useTranslations('assistant');
  const errText = useAssistantError();
  const [text, setText] = useState(initial);
  const [photo, setPhoto] = useState<{ id: number; name: string } | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const [lastRestore, setLastRestore] = useState<string | null>(null);
  if (restore && restore !== lastRestore) {
    setLastRestore(restore);
    setText(restore);
  }

  const pick = async (file: File | undefined) => {
    if (!file) return;
    setError(null);
    if (!/^image\/(jpeg|png|webp)$/.test(file.type)) return setError(t('errors.photo_type'));
    if (file.size > MAX_PHOTO) return setError(t('errors.photo_too_large'));
    setUploading(true);
    try {
      setPhoto({ id: await uploadPhoto(file), name: file.name });
    } catch (e) {
      setError(errText(e));
    } finally {
      setUploading(false);
      if (fileRef.current) fileRef.current.value = '';
    }
  };
  const submit = () => {
    const body = text.trim();
    if (!body || busy || uploading || disabled) return;
    onSend(body, photo?.id ?? null);
    setText('');
    setPhoto(null);
  };

  return (
    <div className="as-composer">
      {photo ? (
        <div className="as-attached">
          <Icon name="image" size={16} />
          <span>{t('chat.photoAttached')}</span>
          <button type="button" className="as-x" aria-label={t('chat.removePhoto')} onClick={() => setPhoto(null)}><Icon name="x" size={14} /></button>
        </div>
      ) : null}
      {error ? <p className="as-error" role="alert">{error}</p> : null}
      <div className="as-composer-row">
        <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/webp" hidden onChange={(e) => void pick(e.target.files?.[0])} />
        <button type="button" className="as-icon-btn" aria-label={t('chat.attach')} disabled={disabled || busy || uploading} onClick={() => fileRef.current?.click()}>
          <Icon name={uploading ? 'loader' : 'camera'} size={20} />
        </button>
        <textarea
          className="as-input"
          rows={1}
          value={text}
          maxLength={MAX_RUNES}
          placeholder={t('chat.placeholder')}
          aria-label={t('chat.placeholder')}
          disabled={disabled}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              submit();
            }
          }}
        />
        {busy ? (
          <button type="button" className="as-send is-stop" aria-label={t('chat.stop')} onClick={onStop}><Icon name="x" size={20} /></button>
        ) : (
          <button type="button" className="as-send" aria-label={t('chat.send')} disabled={disabled || uploading || !text.trim()} onClick={submit}>
            <Icon name="send" size={20} className="as-flip" />
          </button>
        )}
      </div>
    </div>
  );
}
