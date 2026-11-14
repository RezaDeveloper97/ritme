'use client';

import { useState } from 'react';

import {
  acceptErrorKind,
  type AcceptErrorKind,
  isCompleteCompanionCode,
  isPreviewUnsupported,
  useAcceptCompanion,
  useInvitePreview,
  type ViewerLink,
} from '@/entities/companion';

export type JoinStep = 'code' | 'confirm';

export interface JoinCompanion {
  step: JoinStep;
  code: string;
  changeCode: (next: string) => void;
  complete: boolean;
  /** «اتصال»: preview the code, then show the confirm step. */
  submitCode: () => void;
  /** «شروع همراهی»: accept the code. */
  confirm: () => void;
  backToCode: () => void;
  /** The owner's first name from the preview; null on Laravel (no preview) or when hidden. */
  ownerName: string | null;
  error: AcceptErrorKind | null;
  checking: boolean;
  accepting: boolean;
}

/**
 * The member's join flow (companion v2 `join` → `joinConfirm`): the typed code is previewed
 * (`POST /family/invites/preview`) so the confirm step can name her; a server without the preview
 * (Laravel production: 404/405) skips straight to the confirm step with generic copy. The code lives only in
 * this state and the request bodies — never logged, persisted or put in a URL.
 */
export function useJoinCompanion(onLinked: (link: ViewerLink) => void): JoinCompanion {
  const preview = useInvitePreview();
  const accept = useAcceptCompanion();
  const [step, setStep] = useState<JoinStep>('code');
  const [code, setCode] = useState('');
  const [ownerName, setOwnerName] = useState<string | null>(null);
  const [error, setError] = useState<AcceptErrorKind | null>(null);
  const complete = isCompleteCompanionCode(code);
  const checking = preview.isPending;
  const accepting = accept.isPending;

  const submitCode = () => {
    if (!complete || checking || accepting) return;
    setError(null);
    preview.mutate(code, {
      onSuccess: (p) => {
        setOwnerName(p.ownerFirstName);
        setStep('confirm');
      },
      onError: (e) => {
        if (isPreviewUnsupported(e)) {
          setOwnerName(null);
          setStep('confirm');
        } else {
          setError(acceptErrorKind(e));
        }
      },
    });
  };

  const confirm = () => {
    if (!complete || accepting) return;
    accept.mutate(code, {
      onSuccess: (link) => onLinked(link),
      // A refused code is fixed on the code step, where the error line sits under the field.
      onError: (e) => {
        setError(acceptErrorKind(e));
        setStep('code');
      },
    });
  };

  return {
    step,
    code,
    changeCode: (next) => {
      setCode(next);
      setError(null);
    },
    complete,
    submitCode,
    confirm,
    backToCode: () => setStep('code'),
    ownerName,
    error,
    checking,
    accepting,
  };
}
