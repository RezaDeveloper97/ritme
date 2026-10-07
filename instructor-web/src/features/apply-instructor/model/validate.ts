import { APPLY_LIMITS, type ApplyInput } from '@/entities/instructor';

export interface ApplyDraft {
  display_name: string;
  title: string;
  bio: string;
}

export type ApplyFieldError = 'nameRequired' | 'nameShort' | 'nameLong' | 'titleLong' | 'bioLong';
export type ApplyErrors = Partial<Record<keyof ApplyDraft, ApplyFieldError>>;

/** Client copy of the server rules (ValidateApply), for instant feedback. */
export function validateApply(draft: ApplyDraft): ApplyErrors {
  const errors: ApplyErrors = {};
  const name = draft.display_name.trim();
  const len = Array.from(name).length;
  if (len === 0) errors.display_name = 'nameRequired';
  else if (len < APPLY_LIMITS.displayNameMin) errors.display_name = 'nameShort';
  else if (len > APPLY_LIMITS.displayNameMax) errors.display_name = 'nameLong';
  if (Array.from(draft.title.trim()).length > APPLY_LIMITS.titleMax) errors.title = 'titleLong';
  if (Array.from(draft.bio.trim()).length > APPLY_LIMITS.bioMax) errors.bio = 'bioLong';
  return errors;
}

/** The request body: trimmed, empty optional fields as null. */
export function toApplyInput(draft: ApplyDraft): ApplyInput {
  const title = draft.title.trim();
  const bio = draft.bio.trim();
  return { display_name: draft.display_name.trim(), title: title || null, bio: bio || null };
}
