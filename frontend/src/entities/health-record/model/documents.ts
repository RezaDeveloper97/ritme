/*
 * Record documents, categories, extras and timeline (canvas-build CB-REC-01/02, boards nbl_Rec_Home / _Timeline /
 * _Doc). Health data (§11): never logged, never put in a URL — only ids and the filter kind travel in routes.
 */

/** Document kinds a user adds here (lab sheets stay under /labs). */
export const DOCUMENT_KINDS = ['imaging', 'visit', 'prescription', 'hospital', 'other'] as const;
export type DocumentKind = (typeof DOCUMENT_KINDS)[number];

/** Timeline filter chips in board order (`all` + lab sheets + the document kinds). */
export const TIMELINE_KINDS = ['all', 'lab', 'imaging', 'visit', 'prescription', 'hospital', 'other'] as const;
export type TimelineKind = (typeof TIMELINE_KINDS)[number];
/** A timeline row's kind: a lab sheet or a document kind. */
export type TimelineItemKind = Exclude<TimelineKind, 'all'>;

/** The home grid's API categories (board order). */
export const CATEGORY_KEYS = ['labs', 'imaging', 'visit', 'prescription', 'hospital', 'other', 'surgeries', 'family_history'] as const;
export type CategoryKey = (typeof CATEGORY_KEYS)[number];

export type ReviewState = 'manual' | 'pending' | 'needs_review' | 'confirmed' | 'failed';
export type LinkType = 'claim' | 'pregnancy';
export type LinkState = 'attached' | 'waiting' | 'applied';

export interface RecordCategories {
  counts: Record<CategoryKey, number>;
  documentsCount: number;
  /** Documents + ready lab sheets. */
  allCount: number;
  needsReviewCount: number;
}

export const RELATIVES = ['mother', 'father', 'sister', 'brother', 'grandmother', 'grandfather', 'aunt', 'uncle', 'child', 'other'] as const;
export type Relative = (typeof RELATIVES)[number];

export interface Surgery {
  title: string;
  date: string | null;
}
export interface FamilyHistoryEntry {
  condition: string;
  relative: Relative | null;
}

export interface RecordExtras {
  /** null = never answered, [] = «ندارم» (edited through PUT /health-record/basics). */
  allergies: string[] | null;
  allergiesOnEmergencyCard: boolean;
  surgeries: Surgery[] | null;
  familyHistory: FamilyHistoryEntry[] | null;
}

/** PUT /health-record/extras — a present key replaces, an absent key keeps. */
export interface RecordExtrasInput {
  allergiesOnEmergencyCard?: boolean;
  surgeries?: Surgery[] | null;
  familyHistory?: FamilyHistoryEntry[] | null;
}

export const MAX_EXTRAS_ENTRIES = 20;
export const MAX_EXTRAS_TEXT = 120;

export interface TimelineLink {
  type: LinkType;
  state: LinkState;
}

export interface TimelineItem {
  type: 'document' | 'lab';
  id: number;
  kind: TimelineItemKind;
  /** null → the client shows the kind label. */
  title: string | null;
  date: string;
  /** false = no document date yet; filed under its upload day. */
  dateKnown: boolean;
  endedOn: string | null;
  centre: string | null;
  doctor: string | null;
  fileCount: number | null;
  reviewState: ReviewState | null;
  lab: { category: string; markerCount: number; attentionCount: number; allNormal: boolean } | null;
  links: TimelineLink[];
}

export interface TimelineMonth {
  key: string;
  jalaliYear: number;
  jalaliMonth: number;
  start: string;
  end: string;
  items: TimelineItem[];
}

export interface TimelinePage {
  kind: TimelineKind;
  months: TimelineMonth[];
  /** Pass as `before` for the next page; null = no more. */
  nextBefore: string | null;
}

export interface DocumentFile {
  id: number;
  mime: string;
  sizeBytes: number;
  /** Signed link valid 5 minutes; null while the storage is off. */
  url: string | null;
  urlExpiresAt: string | null;
}

export interface WhereUsed {
  type: LinkType;
  targetId: number;
  state: LinkState;
  updatedAt: string | null;
}

/** A read value with the model's confidence (0–1). */
export interface ReadValue {
  value: string | number | boolean;
  confidence: number;
}

export interface Extraction {
  schema: DocumentKind;
  status: 'pending' | 'done' | 'failed';
  errorCode: string | null;
  fields: Record<string, ReadValue>;
  /** Prescription rows: medicine / dose / frequency / duration. */
  items: Record<string, ReadValue>[];
  /** The values the user confirmed. */
  reviewed: Record<string, string | number | null> | null;
  reviewedItems: Record<string, string | null>[] | null;
  dating: 'applied' | 'dismissed' | null;
}

export interface RecordDocument {
  id: number;
  kind: DocumentKind;
  title: string | null;
  date: string | null;
  endedOn: string | null;
  centre: string | null;
  doctor: string | null;
  note: string | null;
  reviewState: ReviewState;
  extracted: Extraction | null;
  files: DocumentFile[];
  whereUsed: WhereUsed[];
}

/** POST / PUT /health-record/documents — PUT sends only the keys present. */
export interface RecordDocumentInput {
  kind?: DocumentKind;
  title?: string | null;
  date?: string | null;
  endedOn?: string | null;
  centre?: string | null;
  doctor?: string | null;
  note?: string | null;
  fileIds?: number[];
  confirm?: boolean;
}

export const MAX_DOCUMENT_FILES = 10;
export const MAX_DOCUMENT_TEXT = 120;
export const MAX_DOCUMENT_NOTE = 1000;
/** Server limit per uploaded file (CB-CORE-05: photo / PDF 10 MB). */
export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

/** The read fields per document kind, in the order the review form lists them (internal/healthrecord/extract). */
export const EXTRACT_FIELDS: Record<DocumentKind, readonly string[]> = {
  imaging: ['kind', 'date', 'centre', 'doctor', 'ga_weeks', 'ga_days', 'edd', 'findings'],
  visit: ['date', 'centre', 'doctor', 'specialty', 'reason', 'diagnosis', 'findings', 'next_visit'],
  prescription: ['date', 'centre', 'doctor', 'diagnosis'],
  hospital: ['date', 'ended_on', 'centre', 'doctor', 'reason', 'diagnosis', 'procedures', 'findings'],
  other: ['date', 'centre', 'doctor', 'title', 'findings'],
};
export const DATE_FIELDS: readonly string[] = ['date', 'ended_on', 'edd', 'next_visit'];
export const NUMBER_FIELDS: readonly string[] = ['ga_weeks', 'ga_days'];
export const IMAGING_STUDIES = ['ultrasound', 'xray', 'mri', 'ct', 'mammography', 'bone_density', 'other'] as const;
export const ITEM_FIELDS = ['medicine', 'dose', 'frequency', 'duration'] as const;
export type ItemField = (typeof ITEM_FIELDS)[number];

export type DatingState = 'offered' | 'applied' | 'dismissed' | 'unavailable';

export interface DatingOffer {
  state: DatingState;
  reason: string | null;
  /** The server's localized sentence. */
  message: string;
  scanDate: string | null;
  proposed: { gaWeeks: number; gaDays: number; dueDate: string; weeksToday: number; daysToday: number } | null;
  current: { source: string; dueDate: string } | null;
  differenceDays: number | null;
}

/** POST /files answer (CB-CORE-05). */
export interface StoredFile {
  id: number;
  mime: string;
  sizeBytes: number;
}
