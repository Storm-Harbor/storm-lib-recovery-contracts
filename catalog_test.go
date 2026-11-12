package recovery

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestErrorCatalogsAreCompleteAndKeepSafeInterpolation(t *testing.T) {
	locales := []string{"en-GB", "en-US", "es-ES", "ca-ES", "pt-PT", "pt-BR", "it-IT", "ru-RU", "de-DE", "fr-FR"}
	codes := []ErrorCode{InvalidDocument, InvalidConfiguration, InvalidScope, ScopeMismatch, SourceTargetConflict, DuplicateSecretReference}
	pattern := regexp.MustCompile(`\{[^}]+\}`)
	for _, locale := range locales {
		data, err := os.ReadFile("i18n/" + locale + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		if len(catalog) != len(codes) {
			t.Fatalf("%s: unexpected keys", locale)
		}
		for _, code := range codes {
			key := "contracts." + string(code)
			message, ok := catalog[key]
			if !ok || strings.TrimSpace(message) == "" || message == key {
				t.Fatalf("%s: missing %s", locale, key)
			}
			placeholders := pattern.FindAllString(message, -1)
			if len(placeholders) != 1 || placeholders[0] != "{field}" {
				t.Fatalf("%s %s: mismatched placeholders", locale, key)
			}
		}
	}
}
