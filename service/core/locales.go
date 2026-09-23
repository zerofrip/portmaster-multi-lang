package core

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/base/database/record"
	"github.com/safing/portmaster/base/log"
	"github.com/safing/portmaster/base/runtime"
)

//go:embed locales/*.json
var localeFS embed.FS

// Locale describes a user-interface language available to the Angular settings
// UI. Each locale is backed by an embedded JSON file in service/core/locales.
type Locale struct {
	record.Base

	// ID is the stable BCP-47-like locale identifier (e.g. "en-GB").
	ID string `json:"id"`

	// DisplayName is the human readable name shown in the locale setting.
	DisplayName string `json:"displayName"`

	// AngularLocale is the locale identifier passed to Angular's LOCALE_ID.
	AngularLocale string `json:"angularLocale"`

	// AngularModule is the filename (without extension) used to import Angular
	// common locale data from @angular/common/locales.
	AngularModule string `json:"angularModule"`

	// NzLocale is the NG-Zorro i18n export name (e.g. "en_GB", "ja_JP").
	NzLocale string `json:"nzLocale"`

	// Translations maps English source strings to their translated values.
	Translations map[string]string `json:"translations"`
}

// LocaleRecord is the runtime record type exposed at runtime:core/locales/<id>.
type LocaleRecord struct {
	record.Base
	sync.Mutex

	ID            string            `json:"id"`
	DisplayName   string            `json:"displayName"`
	AngularLocale string            `json:"angularLocale"`
	AngularModule string            `json:"angularModule"`
	NzLocale      string            `json:"nzLocale"`
	Translations  map[string]string `json:"translations"`
}

var (
	// localeCatalog holds all validated locales indexed by ID.
	localeCatalog map[string]*Locale

	// localeIDPattern restricts locale identifiers to BCP-47-like tags.
	localeIDPattern = regexp.MustCompile(`^[a-zA-Z]{2}(-[a-zA-Z0-9]+)*$`)

	// nzLocalePattern restricts NG-Zorro export names to valid JavaScript
	// identifiers using underscores.
	nzLocalePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// angularModulePattern only allows locale modules shipped by Angular and
	// prevents an embedded definition from escaping the locale directory.
	angularModulePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z]{2})?$`)
)

func init() {
	var err error
	localeCatalog, err = loadLocales()
	if err != nil {
		// Loading happens at package initialization; malformed embedded
		// catalogs are a fatal build-time problem.
		panic(fmt.Sprintf("failed to load locale catalog: %s", err))
	}
}

// loadLocales parses and validates all embedded locale JSON files.
func loadLocales() (map[string]*Locale, error) {
	entries, err := localeFS.ReadDir("locales")
	if err != nil {
		return nil, fmt.Errorf("failed to read locales directory: %w", err)
	}

	catalog := make(map[string]*Locale, len(entries))
	var hasFallback bool

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := localeFS.ReadFile(path.Join("locales", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		locale := &Locale{}
		if err := json.Unmarshal(data, locale); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		if err := validateLocale(entry.Name(), locale); err != nil {
			return nil, fmt.Errorf("invalid locale %s: %w", entry.Name(), err)
		}

		if _, exists := catalog[locale.ID]; exists {
			return nil, fmt.Errorf("duplicate locale id %q", locale.ID)
		}

		if locale.ID == enGBLocale {
			hasFallback = true
		}

		catalog[locale.ID] = locale
	}

	if !hasFallback {
		return nil, errors.New("missing required fallback locale en-GB")
	}

	return catalog, nil
}

// validateLocale checks a single locale definition for consistency.
func validateLocale(fileName string, locale *Locale) error {
	if locale.ID == "" {
		return errors.New("missing locale id")
	}

	expectedFile := locale.ID + ".json"
	if fileName != expectedFile {
		return fmt.Errorf("filename %q does not match locale id %q", fileName, locale.ID)
	}

	if !localeIDPattern.MatchString(locale.ID) {
		return fmt.Errorf("locale id %q has invalid format", locale.ID)
	}

	if locale.DisplayName == "" {
		return fmt.Errorf("locale %q missing display name", locale.ID)
	}

	if locale.AngularLocale == "" {
		return fmt.Errorf("locale %q missing angularLocale", locale.ID)
	}

	if locale.AngularModule == "" {
		return fmt.Errorf("locale %q missing angularModule", locale.ID)
	}

	if !angularModulePattern.MatchString(locale.AngularModule) {
		return fmt.Errorf("locale %q angularModule %q has invalid format", locale.ID, locale.AngularModule)
	}

	if locale.NzLocale == "" {
		return fmt.Errorf("locale %q missing nzLocale", locale.ID)
	}

	if !nzLocalePattern.MatchString(locale.NzLocale) {
		return fmt.Errorf("locale %q nzLocale %q has invalid format", locale.ID, locale.NzLocale)
	}

	if locale.Translations == nil {
		return fmt.Errorf("locale %q missing translations map", locale.ID)
	}

	return nil
}

// getLocales returns a sorted slice of all catalog locales.
func getLocales() []*Locale {
	locales := make([]*Locale, 0, len(localeCatalog))
	for _, locale := range localeCatalog {
		locales = append(locales, locale)
	}

	sort.Slice(locales, func(i, j int) bool {
		return locales[i].DisplayName < locales[j].DisplayName
	})

	return locales
}

// getLocale returns the catalog locale for id, falling back to en-GB.
func getLocale(id string) *Locale {
	if locale, ok := localeCatalog[id]; ok {
		return locale
	}

	return localeCatalog[enGBLocale]
}

// localePossibleValues generates config.PossibleValue entries from the catalog.
func localePossibleValues() []config.PossibleValue {
	locales := getLocales()
	values := make([]config.PossibleValue, 0, len(locales))
	for _, locale := range locales {
		values = append(values, config.PossibleValue{
			Name:  locale.DisplayName,
			Value: locale.ID,
		})
	}

	return values
}

// registerLocales exposes every catalog locale as a read-only runtime record
// under runtime:core/locales/<id>.
func registerLocales() error {
	_, err := runtime.Register("core/locales/", runtime.SimpleValueGetterFunc(func(keyOrPrefix string) ([]record.Record, error) {
		// The runtime registry calls Get with the full database key. Strip the
		// prefix to recover the locale id.
		id := strings.TrimPrefix(keyOrPrefix, "core/locales/")
		if id == "" {
			locales := getLocales()
			records := make([]record.Record, 0, len(locales))
			for _, locale := range locales {
				records = append(records, localeToRecord(locale))
			}
			return records, nil
		}

		locale, ok := localeCatalog[id]
		if !ok {
			return nil, nil
		}
		rec := localeToRecord(locale)
		return []record.Record{rec}, nil
	}))

	return err
}

// localeToRecord converts a Locale into a runtime record with the correct key.
func localeToRecord(locale *Locale) *LocaleRecord {
	rec := &LocaleRecord{
		ID:            locale.ID,
		DisplayName:   locale.DisplayName,
		AngularLocale: locale.AngularLocale,
		AngularModule: locale.AngularModule,
		NzLocale:      locale.NzLocale,
		Translations:  locale.Translations,
	}
	rec.SetKey("runtime:core/locales/" + locale.ID)
	rec.UpdateMeta()
	return rec
}

// logLoadedLocales writes the number and IDs of loaded locales to the log.
func logLoadedLocales() {
	ids := make([]string, 0, len(localeCatalog))
	for id := range localeCatalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	log.Infof("core: loaded %d locale(s): %s", len(ids), strings.Join(ids, ", "))
}
