package app

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/leonelquinteros/gotext"
	"golang.org/x/text/language"
)

//go:embed locales
var localeFS embed.FS

type locale struct {
	dev     bool
	dir     string                    // dev only: path to locales dir on disk
	matcher language.Matcher
	langs   []string                  // discovered language codes
	locales map[string]*gotext.Locale // prod only: preloaded per lang
}

func loadLocale(dev bool) (*locale, error) {
	l := &locale{dev: dev}

	var dirFS fs.FS
	if dev {
		l.dir = filepath.Join("app", "locales")
		dirFS = os.DirFS(l.dir)
	} else {
		sub, err := fs.Sub(localeFS, "locales")
		if err != nil {
			return nil, fmt.Errorf("loadLocale: %w", err)
		}
		dirFS = sub
	}

	entries, err := fs.ReadDir(dirFS, ".")
	if err != nil {
		return nil, fmt.Errorf("loadLocale: %w", err)
	}

	var tags []language.Tag
	if !dev {
		l.locales = make(map[string]*gotext.Locale)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lang := e.Name()
		tag, err := language.Parse(lang)
		if err != nil {
			continue
		}
		l.langs = append(l.langs, lang)
		tags = append(tags, tag)

		if !dev {
			loc := gotext.NewLocaleFSWithPath(lang, dirFS, ".")
			loc.AddDomain("default")
			l.locales[lang] = loc
		}
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("loadLocale: no locale directories found")
	}

	l.matcher = language.NewMatcher(tags)
	return l, nil
}

func (l *locale) translateFunc(lang string) func(string, ...any) string {
	var loc *gotext.Locale
	if l.dev {
		loc = gotext.NewLocaleFSWithPath(lang, os.DirFS(l.dir), ".")
		loc.AddDomain("default")
	} else {
		loc = l.locales[lang]
		if loc == nil {
			loc = l.locales[l.langs[0]]
		}
	}
	return loc.Get
}

func (l *locale) match(accept, cookie, query string) string {
	var preferred []language.Tag
	// Query param wins, then cookie, then Accept-Language header.
	for _, pref := range []string{query, cookie} {
		if pref != "" {
			if tag, err := language.Parse(pref); err == nil {
				preferred = append(preferred, tag)
			}
		}
	}
	if accept != "" {
		tags, _, err := language.ParseAcceptLanguage(accept)
		if err == nil {
			preferred = append(preferred, tags...)
		}
	}
	tag, _, _ := l.matcher.Match(preferred...)
	base, _ := tag.Base()
	return base.String()
}

func localeCookieValue(r *http.Request) string {
	c, err := r.Cookie("bbl.locale")
	if err != nil {
		return ""
	}
	return c.Value
}

func (app *App) switchLang(w http.ResponseWriter, r *http.Request) {
	lang := r.PathValue("lang")
	matched := app.locale.match("", "", lang)
	http.SetCookie(w, &http.Cookie{
		Name:     "bbl.locale",
		Value:    matched,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	dest := r.URL.Query().Get("return")
	if dest == "" || dest[0] != '/' {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}
