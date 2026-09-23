package core

import (
	"testing"
)

func TestLocaleCatalogContainsJapanese(t *testing.T) {
	ja, ok := localeCatalog["ja-JP"]
	if !ok {
		t.Fatalf("expected ja-JP locale to be loaded")
	}

	if ja.DisplayName != "日本語" {
		t.Errorf("expected display name 日本語, got %q", ja.DisplayName)
	}

	if ja.AngularModule != "ja" {
		t.Errorf("expected angular module ja, got %q", ja.AngularModule)
	}

	if ja.NzLocale != "ja_JP" {
		t.Errorf("expected nzLocale ja_JP, got %q", ja.NzLocale)
	}

	got, ok := ja.Translations["Search"]
	if !ok {
		t.Fatalf("expected Japanese translation for Search")
	}
	if got != "検索" {
		t.Errorf("expected 検索 for Search, got %q", got)
	}

	got, ok = ja.Translations["Global Settings"]
	if !ok {
		t.Fatalf("expected Japanese translation for Global Settings")
	}
	if got != "全般設定" {
		t.Errorf("expected 全般設定 for Global Settings, got %q", got)
	}
}

func TestLocaleFallbackToGB(t *testing.T) {
	locale := getLocale("unknown-locale")
	if locale.ID != enGBLocale {
		t.Errorf("expected fallback to %s, got %s", enGBLocale, locale.ID)
	}
}

func TestLocalePossibleValuesGenerated(t *testing.T) {
	values := localePossibleValues()
	if len(values) != len(localeCatalog) {
		t.Fatalf("expected %d possible values, got %d", len(localeCatalog), len(values))
	}

	found := false
	for _, v := range values {
		if v.Value == "ja-JP" {
			found = true
			if v.Name != "日本語" {
				t.Errorf("expected name 日本語, got %q", v.Name)
			}
		}
	}
	if !found {
		t.Errorf("expected ja-JP in possible values")
	}
}

func TestValidateLocaleRejectsMissingID(t *testing.T) {
	locale := &Locale{
		DisplayName:   "Test",
		AngularLocale: "en",
		AngularModule: "en",
		NzLocale:      "en_US",
		Translations:  map[string]string{},
	}
	if err := validateLocale("test.json", locale); err == nil {
		t.Error("expected error for missing id")
	}
}

func TestValidateLocaleRejectsFilenameMismatch(t *testing.T) {
	locale := &Locale{
		ID:            "de-DE",
		DisplayName:   "German",
		AngularLocale: "de",
		AngularModule: "de",
		NzLocale:      "de_DE",
		Translations:  map[string]string{},
	}
	if err := validateLocale("de.json", locale); err == nil {
		t.Error("expected error for filename mismatch")
	}
}

func TestValidateLocaleRejectsInvalidID(t *testing.T) {
	locale := &Locale{
		ID:            "de_DE",
		DisplayName:   "German",
		AngularLocale: "de",
		AngularModule: "de",
		NzLocale:      "de_DE",
		Translations:  map[string]string{},
	}
	if err := validateLocale("de_DE.json", locale); err == nil {
		t.Error("expected error for invalid locale id")
	}
}

func TestValidateLocaleRejectsMissingDisplayName(t *testing.T) {
	locale := &Locale{
		ID:            "de-DE",
		AngularLocale: "de",
		AngularModule: "de",
		NzLocale:      "de_DE",
		Translations:  map[string]string{},
	}
	if err := validateLocale("de-DE.json", locale); err == nil {
		t.Error("expected error for missing display name")
	}
}

func TestValidateLocaleRejectsMissingAngularModule(t *testing.T) {
	locale := &Locale{
		ID:            "de-DE",
		DisplayName:   "German",
		AngularLocale: "de",
		NzLocale:      "de_DE",
		Translations:  map[string]string{},
	}
	if err := validateLocale("de-DE.json", locale); err == nil {
		t.Error("expected error for missing angular module")
	}
}

func TestValidateLocaleRejectsPathInAngularModule(t *testing.T) {
	locale := &Locale{
		ID:            "de-DE",
		DisplayName:   "German",
		AngularLocale: "de",
		AngularModule: "../de",
		NzLocale:      "de_DE",
		Translations:  map[string]string{},
	}
	if err := validateLocale("de-DE.json", locale); err == nil {
		t.Error("expected error for path separator in angular module")
	}
}

func TestValidateLocaleRejectsInvalidNzLocale(t *testing.T) {
	locale := &Locale{
		ID:            "de-DE",
		DisplayName:   "German",
		AngularLocale: "de",
		AngularModule: "de",
		NzLocale:      "de-DE",
		Translations:  map[string]string{},
	}
	if err := validateLocale("de-DE.json", locale); err == nil {
		t.Error("expected error for invalid nzLocale format")
	}
}

func TestValidateLocaleRejectsMissingTranslations(t *testing.T) {
	locale := &Locale{
		ID:            "de-DE",
		DisplayName:   "German",
		AngularLocale: "de",
		AngularModule: "de",
		NzLocale:      "de_DE",
	}
	if err := validateLocale("de-DE.json", locale); err == nil {
		t.Error("expected error for missing translations map")
	}
}
