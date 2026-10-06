/*
 * Client-side checks of the lab upload, mirroring the server (B-N6-06
 * `internal/labs/upload.go`): 1–5 files, JPEG / PNG / WebP photos ≤ 5 MB,
 * PDFs ≤ 10 MB, the whole body ≤ 20 MB, a category from the list, a sheet date
 * not in the future. The server still sniffs every file — these checks only
 * save a round trip and give a clear message before the upload starts.
 */

export type PageKind = 'image' | 'pdf';

export interface UploadLimits {
  maxFiles: number;
  maxImageBytes: number;
  maxPdfBytes: number;
  /** The whole multipart body (server `MaxUploadBytes`). */
  maxTotalBytes: number;
  categories: readonly string[];
}

export const DEFAULT_UPLOAD_LIMITS: UploadLimits = {
  maxFiles: 5,
  maxImageBytes: 5120 * 1024,
  maxPdfBytes: 10240 * 1024,
  maxTotalBytes: 20 * 1024 * 1024,
  categories: ['blood', 'hormone', 'thyroid', 'urine', 'other'],
};

/** Limits from `GET /labs` (`limits`), falling back to the defaults for anything missing. */
export function limitsFrom(api: { maxFiles: number; maxImageKb: number; maxPdfKb: number; categories: string[] } | undefined): UploadLimits {
  if (!api) return DEFAULT_UPLOAD_LIMITS;
  return {
    maxFiles: api.maxFiles > 0 ? api.maxFiles : DEFAULT_UPLOAD_LIMITS.maxFiles,
    maxImageBytes: api.maxImageKb > 0 ? api.maxImageKb * 1024 : DEFAULT_UPLOAD_LIMITS.maxImageBytes,
    maxPdfBytes: api.maxPdfKb > 0 ? api.maxPdfKb * 1024 : DEFAULT_UPLOAD_LIMITS.maxPdfBytes,
    maxTotalBytes: DEFAULT_UPLOAD_LIMITS.maxTotalBytes,
    categories: api.categories.length ? api.categories : DEFAULT_UPLOAD_LIMITS.categories,
  };
}

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const IMAGE_EXT = /\.(jpe?g|png|webp)$/i;
const PDF_EXT = /\.pdf$/i;

/** The accept lists of the three pickers. */
export const ACCEPT_IMAGES = IMAGE_TYPES.join(',');
export const ACCEPT_PDF = 'application/pdf';

/** A file the server would take, by its declared type (or extension when the browser gives none). Null otherwise. */
export function kindOf(file: { name: string; type: string }): PageKind | null {
  const type = file.type.toLowerCase();
  if (IMAGE_TYPES.includes(type)) return 'image';
  if (type === 'application/pdf') return 'pdf';
  if (type === '') {
    if (IMAGE_EXT.test(file.name)) return 'image';
    if (PDF_EXT.test(file.name)) return 'pdf';
  }
  return null;
}

export type FileError = 'file_type' | 'image_too_large' | 'pdf_too_large' | 'empty_file';

/** Why one file can't go up, or null. */
export function checkFile(file: { name: string; type: string; size: number }, limits: UploadLimits): FileError | null {
  const kind = kindOf(file);
  if (!kind) return 'file_type';
  if (file.size <= 0) return 'empty_file';
  if (kind === 'image' && file.size > limits.maxImageBytes) return 'image_too_large';
  if (kind === 'pdf' && file.size > limits.maxPdfBytes) return 'pdf_too_large';
  return null;
}

export interface DraftPage {
  /** Local id for keys and reordering. */
  id: string;
  file: File;
  kind: PageKind;
}

export type AddError = FileError | 'too_many_files';

/**
 * Adds picked files after the current pages: the ones that pass {@link checkFile}
 * and fit under `maxFiles`; the first refusal is returned for the message.
 */
export function addPages(
  pages: readonly DraftPage[],
  incoming: readonly File[],
  limits: UploadLimits,
  makeId: () => string,
): { pages: DraftPage[]; error: AddError | null } {
  const next = [...pages];
  let error: AddError | null = null;
  for (const file of incoming) {
    const problem = checkFile(file, limits);
    if (problem) {
      error ??= problem;
      continue;
    }
    if (next.length >= limits.maxFiles) {
      error ??= 'too_many_files';
      continue;
    }
    next.push({ id: makeId(), file, kind: kindOf(file) as PageKind });
  }
  return { pages: next, error };
}

/** Moves a page one step towards the start (-1) or the end (+1). */
export function movePage(pages: readonly DraftPage[], id: string, step: -1 | 1): DraftPage[] {
  const i = pages.findIndex((p) => p.id === id);
  const j = i + step;
  if (i < 0 || j < 0 || j >= pages.length) return [...pages];
  const next = [...pages];
  [next[i], next[j]] = [next[j]!, next[i]!];
  return next;
}

export function removePage(pages: readonly DraftPage[], id: string): DraftPage[] {
  return pages.filter((p) => p.id !== id);
}

export interface UploadDraft {
  pages: readonly DraftPage[];
  category: string | null;
  /** `Y-m-d`, or null when unknown. */
  takenOn: string | null;
  fasting: boolean | null;
}

export type DraftErrors = Partial<Record<'files' | 'category' | 'taken_on', AddError | 'files_required' | 'upload_too_large' | 'category_invalid' | 'date_future'>>;

/** Everything that must hold before «تحلیل کن»; `today` is `Y-m-d` in the user's day. Empty object = ready. */
export function validateDraft(draft: UploadDraft, limits: UploadLimits, today: string): DraftErrors {
  const errors: DraftErrors = {};
  if (draft.pages.length === 0) errors.files = 'files_required';
  else if (draft.pages.length > limits.maxFiles) errors.files = 'too_many_files';
  else {
    for (const p of draft.pages) {
      const problem = checkFile(p.file, limits);
      if (problem) {
        errors.files = problem;
        break;
      }
    }
    // multipart overhead is small; leave 64 KB for it
    if (!errors.files && draft.pages.reduce((sum, p) => sum + p.file.size, 0) > limits.maxTotalBytes - 64 * 1024) {
      errors.files = 'upload_too_large';
    }
  }
  if (!draft.category || !limits.categories.includes(draft.category)) errors.category = 'category_invalid';
  if (draft.takenOn && draft.takenOn > today) errors.taken_on = 'date_future';
  return errors;
}

/**
 * The multipart body of `POST /labs` (pages in order). Files go up under a
 * neutral name («page-1.jpg») — the device file name can carry a person's name.
 */
export function toFormData(draft: UploadDraft): FormData {
  const form = new FormData();
  draft.pages.forEach((p, i) => form.append('files[]', p.file, `page-${i + 1}.${p.kind === 'pdf' ? 'pdf' : 'jpg'}`));
  form.append('category', draft.category ?? '');
  if (draft.takenOn) form.append('taken_on', draft.takenOn);
  if (draft.fasting !== null) form.append('fasting', draft.fasting ? '1' : '0');
  return form;
}

/** The server only takes a PDF that starts with `%PDF-` (at most 8 bytes of leading junk, B-N6-06b). */
export const PDF_HEADER_WITHIN = 8;

export function hasPdfHeader(head: Uint8Array): boolean {
  const sig = [0x25, 0x50, 0x44, 0x46, 0x2d]; // %PDF-
  const end = Math.min(head.length, PDF_HEADER_WITHIN + sig.length) - sig.length;
  for (let i = 0; i <= end; i++) {
    if (sig.every((b, k) => head[i + k] === b)) return true;
  }
  return false;
}
