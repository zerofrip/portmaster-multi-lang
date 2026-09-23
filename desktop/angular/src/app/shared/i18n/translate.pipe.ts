import { Pipe, PipeTransform } from '@angular/core';
import { I18nService } from './i18n.service';

/**
 * Translate pipe that looks up an English source string in the active locale
 * translation map. Untranslated strings fall back to the original text.
 */
@Pipe({
  name: 'translate',
  pure: true,
})
export class TranslatePipe implements PipeTransform {
  constructor(private i18n: I18nService) { }

  transform(value: string | null | undefined): string {
    if (value === null || value === undefined) {
      return '';
    }

    return this.i18n.translate(value);
  }
}
