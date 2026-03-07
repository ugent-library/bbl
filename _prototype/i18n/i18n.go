package i18n

import (
	"embed"

	"github.com/leonelquinteros/gotext"
)

//go:embed locales
var localesFS embed.FS

var Locales = make(map[string]*gotext.Locale)

func init() {
	locale := gotext.NewLocaleFSWithPath("en", localesFS, "locales")
	locale.AddDomain("default")
	Locales["en"] = locale
}
