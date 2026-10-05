export interface LearnTopic {
  code: string;
  label: string;
}

export interface LearnTip {
  code: string;
  topic: string;
  topicLabel: string | null;
  title: string;
  body: string | null;
  minutes: number | null;
  /** A published article behind the tip (opens the article sheet by slug). */
  articleSlug: string | null;
}

/** GET /children/{id}/learn?topic= */
export interface LearnView {
  ageMonths: number;
  topics: LearnTopic[];
  topic: string | null;
  featured: LearnTip | null;
  tips: LearnTip[];
  disclaimer: string | null;
}
