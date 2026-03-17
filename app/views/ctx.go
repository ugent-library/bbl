package views

type Ctx struct {
	AssetPath func(string) string
	Loc       func(string, ...any) string
	Lang      string            // BCP-47 tag, e.g. "en", "nl"
	Langs     []string          // all supported UI languages
	LangNames map[string]string // ISO 639-2 code → localized name (code as fallback)
	MainLangs []string          // preferred languages shown first in selects
	Path      string            // current request path (for lang switcher redirect)
}
