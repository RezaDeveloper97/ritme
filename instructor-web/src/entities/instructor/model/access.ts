import type { Instructor } from './schema';

/**
 * Where the signed-in user stands with the instructor panel:
 *  - `required` — never applied (403 instructor_required on the panel API);
 *  - `revoked`  — access withdrawn by an admin (also instructor_required); may re-apply;
 *  - `pending`  — applied, awaiting admin approval (403 instructor_pending);
 *  - `approved` — the panel.
 */
export type InstructorAccess = 'required' | 'revoked' | 'pending' | 'approved';

export function accessOf(instructor: Instructor | null | undefined): InstructorAccess {
  if (!instructor) return 'required';
  return instructor.status;
}

/** The route that renders a given access state. */
export function pathForAccess(access: InstructorAccess): string {
  switch (access) {
    case 'approved':
      return '/';
    case 'pending':
      return '/pending';
    default:
      return '/apply';
  }
}

/** First letter of the display name for the avatar disc. */
export function initialOf(name: string): string {
  return Array.from(name.trim())[0] ?? '';
}
