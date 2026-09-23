import { Record, RecordMeta } from '@safing/portmaster-api';

export interface LocaleRecord extends Record {
  /** Stable locale identifier (e.g. "en-GB", "ja-JP"). */
  id: string;

  /** Human readable name of the locale. */
  displayName: string;

  /** Value for Angular's LOCALE_ID. */
  angularLocale: string;

  /** Filename (without extension) for @angular/common/locales imports. */
  angularModule: string;

  /** NG-Zorro i18n export name (e.g. "en_GB", "ja_JP"). */
  nzLocale: string;

  /** English-keyed translation map. */
  translations: { [key: string]: string };

  _meta?: RecordMeta;
}
