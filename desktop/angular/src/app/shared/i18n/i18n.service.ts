import { registerLocaleData } from '@angular/common';
import { Injectable } from '@angular/core';
import { ConfigService, PortapiService, StringSetting, getActualValue } from '@safing/portmaster-api';
import * as i18n from 'ng-zorro-antd/i18n';
import { firstValueFrom } from 'rxjs';
import { LocaleRecord } from './locale.types';

const fallbackLocaleID = 'en-GB';

/**
 * I18nService loads the user-selected locale from the Portmaster runtime
 * registry, registers the matching Angular locale data and NG-Zorro locale,
 * and provides English-keyed translations for the settings UI.
 */
@Injectable({
  providedIn: 'root'
})
export class I18nService {
  /** The currently active locale record, or null before initialization. */
  private currentLocale: LocaleRecord | null = null;

  /** Angular LOCALE_ID value. */
  private angularLocaleId = fallbackLocaleID;

  /** NG-Zorro locale object. */
  private nzLocale: any = i18n.en_GB;

  constructor(
    private configService: ConfigService,
    private portapi: PortapiService,
  ) { }

  /**
   * Initializes the locale. This is meant to be called from an
   * APP_INITIALIZER so the locale is ready before the UI renders.
   */
  async initialize(): Promise<void> {
    let localeID = fallbackLocaleID;

    try {
      const setting = await firstValueFrom(this.configService.get('core/locale'));
      const value = getActualValue(setting as StringSetting);
      if (typeof value === 'string' && value !== '') {
        localeID = value;
      }
    } catch (err) {
      console.error('i18n: failed to load core/locale setting, using fallback', err);
    }

    await this.loadLocale(localeID);
  }

  /**
   * Loads the given locale and applies it to Angular and NG-Zorro.
   *
   * @param id The locale id to load.
   */
  private async loadLocale(id: string): Promise<void> {
    let record: LocaleRecord;

    try {
      record = await firstValueFrom(this.portapi.get<LocaleRecord>(`runtime:core/locales/${id}`));
    } catch (err) {
      console.error(`i18n: failed to load runtime locale record for ${id}, falling back to ${fallbackLocaleID}`, err);
      if (id === fallbackLocaleID) {
        record = this.buildFallbackRecord();
      } else {
        return this.loadLocale(fallbackLocaleID);
      }
    }

    if (!record) {
      console.error(`i18n: runtime locale record for ${id} is empty, falling back to ${fallbackLocaleID}`);
      if (id === fallbackLocaleID) {
        record = this.buildFallbackRecord();
      } else {
        return this.loadLocale(fallbackLocaleID);
      }
    }

    const angularModule = record.angularModule || record.angularLocale || fallbackLocaleID;

    try {
      /* webpackInclude: /\/[a-zA-Z0-9-]+\.mjs$/ */
      /* webpackChunkName: "l10n-[request]" */
      const localeData = await import(`../../../../node_modules/@angular/common/locales/${angularModule}.mjs`);
      registerLocaleData(localeData.default);
    } catch (err) {
      console.error(`i18n: failed to register Angular locale data for ${angularModule}`, err);
    }

    let nzLocale = i18n.en_GB;
    try {
      const nzLocaleName = record.nzLocale || 'en_GB';
      nzLocale = (i18n as any)[nzLocaleName] || i18n.en_GB;
    } catch (err) {
      console.error(`i18n: failed to resolve NG-Zorro locale ${record.nzLocale}, using en-GB`, err);
    }

    this.currentLocale = record;
    this.angularLocaleId = record.angularLocale || fallbackLocaleID;
    this.nzLocale = nzLocale;
  }

  /**
   * Returns a minimal fallback locale record for when both the selected locale
   * and the runtime registry are unavailable.
   */
  private buildFallbackRecord(): LocaleRecord {
    return {
      id: fallbackLocaleID,
      displayName: 'English (UK)',
      angularLocale: fallbackLocaleID,
      angularModule: fallbackLocaleID,
      nzLocale: 'en_GB',
      translations: {},
    };
  }

  /**
   * Translates an English source string using the active locale. Falls back
   * to the original string if no translation exists.
   *
   * @param key The English source string.
   */
  translate(key: string): string {
    if (!key) {
      return key;
    }

    return this.currentLocale?.translations?.[key] ?? key;
  }

  /** Returns the Angular LOCALE_ID value. */
  getLocaleId(): string {
    return this.angularLocaleId;
  }

  /** Returns the active NG-Zorro locale object. */
  getNzLocale(): any {
    return this.nzLocale;
  }
}
