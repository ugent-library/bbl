package views

type Ctx struct {
	AssetPath func(string) string
	Loc       func(string, ...any) string
	Lang      string   // BCP-47 tag, e.g. "en", "nl"
	Langs     []string // all supported languages
	Path      string   // current request path (for lang switcher redirect)
}
