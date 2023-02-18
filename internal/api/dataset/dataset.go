package dataset

import (
	"fmt"
	"io"
	"strings"
)

func GetSupportedLanguages() []string {
	return []string{"deu", "eng"}
}

func NewLanguageMapper(languages map[string]string) LanguageMapper {
	return LanguageMapper{languages: languages}
}

type LanguageMapper struct {
	languages map[string]string
}

func (l LanguageMapper) ByExternal(lang string) string {
	trimmedLang := strings.TrimSpace(lang)
	for k, v := range l.languages {
		if v == trimmedLang {
			return k
		}
	}

	return ""
}

func (l LanguageMapper) Get(lang string) (string, error) {
	extLang, ok := l.languages[strings.TrimSpace(lang)]
	if !ok || extLang == "" {
		return "", fmt.Errorf("language %s not supported, available languages %v", lang, l.languages)
	}

	return extLang, nil
}

type Report struct {
	CardCount int
	SetCount  int
}

type Dataset interface {
	Import(r io.Reader) (*Report, error)
}
