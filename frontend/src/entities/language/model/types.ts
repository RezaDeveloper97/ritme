/** A language the app ships, as served by `GET /languages`. */
export interface Language {
  code: string;
  name: string;
  englishName: string;
  direction: 'rtl' | 'ltr';
  isDefault: boolean;
}
