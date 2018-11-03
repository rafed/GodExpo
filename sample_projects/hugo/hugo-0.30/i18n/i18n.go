// Copyright 2017 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package i18n

import (
	"github.com/gohugoio/hugo/config"
	"github.com/gohugoio/hugo/helpers"
	"github.com/nicksnyder/go-i18n/i18n/bundle"
	jww "github.com/spf13/jwalterweatherman"
)

var (
	i18nWarningLogger = helpers.NewDistinctFeedbackLogger()
)

// Translator handles i18n translations.
type Translator struct {
	translateFuncs map[string]bundle.TranslateFunc
	cfg            config.Provider
	logger         *jww.Notepad
}

// NewTranslator creates a new Translator for the given language bundle and configuration.
func NewTranslator(b *bundle.Bundle, cfg config.Provider, logger *jww.Notepad) Translator {
	t := Translator{cfg: cfg, logger: logger, translateFuncs: make(map[string]bundle.TranslateFunc)}
	t.initFuncs(b)
	return t
}

// Func gets the translate func for the given language, or for the default
// configured language if not found.
func (t Translator) Func(lang string) bundle.TranslateFunc {
	if f, ok := t.translateFuncs[lang]; ok {
		return f
	}
	t.logger.WARN.Printf("Translation func for language %v not found, use default.", lang)
	if f, ok := t.translateFuncs[t.cfg.GetString("defaultContentLanguage")]; ok {
		return f
	}
	t.logger.WARN.Println("i18n not initialized, check that you have language file (in i18n) that matches the site language or the default language.")
	return func(translationID string, args ...interface{}) string {
		return ""
	}

}

func (t Translator) initFuncs(bndl *bundle.Bundle) {
	defaultContentLanguage := t.cfg.GetString("defaultContentLanguage")

	defaultT, err := bndl.Tfunc(defaultContentLanguage)
	if err != nil {
		jww.WARN.Printf("No translation bundle found for default language %q", defaultContentLanguage)
	}

	enableMissingTranslationPlaceholders := t.cfg.GetBool("enableMissingTranslationPlaceholders")
	for _, lang := range bndl.LanguageTags() {
		currentLang := lang

		t.translateFuncs[currentLang] = func(translationID string, args ...interface{}) string {
			tFunc, err := bndl.Tfunc(currentLang)
			if err != nil {
				jww.WARN.Printf("could not load translations for language %q (%s), will use default content language.\n", lang, err)
			}

			translated := tFunc(translationID, args...)
			if translated != translationID {
				return translated
			}
			// If there is no translation for translationID,
			// then Tfunc returns translationID itself.
			// But if user set same translationID and translation, we should check
			// if it really untranslated:
			if isIDTranslated(currentLang, translationID, bndl) {
				return translated
			}

			if t.cfg.GetBool("logI18nWarnings") {
				i18nWarningLogger.Printf("i18n|MISSING_TRANSLATION|%s|%s", currentLang, translationID)
			}
			if enableMissingTranslationPlaceholders {
				return "[i18n] " + translationID
			}
			if defaultT != nil {
				translated := defaultT(translationID, args...)
				if translated != translationID {
					return translated
				}
				if isIDTranslated(defaultContentLanguage, translationID, bndl) {
					return translated
				}
			}
			return ""
		}
	}
}

// If bndl contains the translationID for specified currentLang,
// then the translationID is actually translated.
func isIDTranslated(lang, id string, b *bundle.Bundle) bool {
	_, contains := b.Translations()[lang][id]
	return contains
}
