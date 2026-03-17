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
	"golang.org/x/text/language/display"
)

//go:embed locales
var localeFS embed.FS

type locale struct {
	dev       bool
	dir       string                        // dev only: path to locales dir on disk
	matcher   language.Matcher
	langs     []string                      // discovered language codes
	locales   map[string]*gotext.Locale     // prod only: preloaded per lang
	langNames map[string]map[string]string // ISO 639-2 code → localized name, per UI locale
	mainLangs []string                    // preferred languages shown first in selects
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
	l.langNames = buildLangNames(l.langs)
	l.mainLangs = []string{"eng", "dut", "fre", "ger", "spa", "ita", "por"}
	return l, nil
}

// buildLangNames builds ISO 639-2 code → localized name maps for each UI locale.
func buildLangNames(uiLangs []string) map[string]map[string]string {
	type parsed struct {
		code string
		tag  language.Tag
	}
	var tags []parsed
	for _, code := range iso639_2Codes {
		tag, err := language.Parse(code)
		if err != nil {
			continue
		}
		tags = append(tags, parsed{code: code, tag: tag})
	}

	result := make(map[string]map[string]string, len(uiLangs))
	for _, lang := range uiLangs {
		localeTag, _ := language.Parse(lang)
		namer := display.Languages(localeTag)

		names := make(map[string]string, len(tags))
		for _, p := range tags {
			name := namer.Name(p.tag)
			if name == "" {
				name = p.code
			}
			names[p.code] = name
		}
		result[lang] = names
	}
	return result
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

// Full ISO 639-2/B code list (bibliographic). Source: Library of Congress.
// Matches github.com/ugent-library/vo.IsISO639_2.
var iso639_2Codes = []string{
	"aar", "abk", "ace", "ach", "ada", "ady", "afa", "afh", "afr", "ain",
	"aka", "akk", "alb", "ale", "alg", "alt", "amh", "ang", "anp", "apa",
	"ara", "arc", "arg", "arm", "arn", "arp", "art", "arw", "asm", "ast",
	"ath", "aus", "ava", "ave", "awa", "aym", "aze", "bad", "bai", "bak",
	"bal", "bam", "ban", "baq", "bas", "bat", "bej", "bel", "bem", "ben",
	"ber", "bho", "bih", "bik", "bin", "bis", "bla", "bnt", "bos", "bra",
	"bre", "btk", "bua", "bug", "bul", "bur", "byn", "cad", "cai", "car",
	"cat", "cau", "ceb", "cel", "cha", "chb", "che", "chg", "chi", "chk",
	"chm", "chn", "cho", "chp", "chr", "chu", "chv", "chy", "cmc", "cop",
	"cor", "cos", "cpe", "cpf", "cpp", "cre", "crh", "crp", "csb", "cus",
	"cze", "dak", "dan", "dar", "day", "del", "den", "dgr", "din", "div",
	"doi", "dra", "dsb", "dua", "dum", "dut", "dyu", "dzo", "efi", "egy",
	"eka", "elx", "eng", "enm", "epo", "est", "ewe", "ewo", "fan", "fao",
	"fat", "fij", "fil", "fin", "fiu", "fon", "fre", "frm", "fro", "frr",
	"frs", "fry", "ful", "fur", "gaa", "gay", "gba", "gem", "geo", "ger",
	"gez", "gil", "gla", "gle", "glg", "glv", "gmh", "goh", "gon", "gor",
	"got", "grb", "grc", "gre", "grn", "gsw", "guj", "gwi", "hai", "hat",
	"hau", "haw", "heb", "her", "hil", "him", "hin", "hit", "hmn", "hmo",
	"hrv", "hsb", "hun", "hup", "iba", "ibo", "ice", "ido", "iii", "ijo",
	"iku", "ile", "ilo", "ina", "inc", "ind", "ine", "inh", "ipk", "ira",
	"iro", "ita", "jav", "jbo", "jpn", "jpr", "jrb", "kaa", "kab", "kac",
	"kal", "kam", "kan", "kar", "kas", "kau", "kaw", "kaz", "kbd", "kha",
	"khi", "khm", "kho", "kik", "kin", "kir", "kmb", "kok", "kom", "kon",
	"kor", "kos", "kpe", "krc", "krl", "kro", "kru", "kua", "kum", "kur",
	"kut", "lad", "lah", "lam", "lao", "lat", "lav", "lez", "lim", "lin",
	"lit", "lol", "loz", "ltz", "lua", "lub", "lug", "lui", "lun", "luo",
	"lus", "mac", "mad", "mag", "mah", "mai", "mak", "mal", "man", "mao",
	"map", "mar", "mas", "may", "mdf", "mdr", "men", "mga", "mic", "min",
	"mis", "mkh", "mlg", "mlt", "mnc", "mni", "mno", "moh", "mon", "mos",
	"mul", "mun", "mus", "mwl", "mwr", "myn", "myv", "nah", "nai", "nap",
	"nau", "nav", "nbl", "nde", "ndo", "nds", "nep", "new", "nia", "nic",
	"niu", "nno", "nob", "nog", "non", "nor", "nqo", "nso", "nub", "nwc",
	"nya", "nym", "nyn", "nyo", "nzi", "oci", "oji", "ori", "orm", "osa",
	"oss", "ota", "oto", "paa", "pag", "pal", "pam", "pan", "pap", "pau",
	"peo", "per", "phi", "phn", "pli", "pol", "pon", "por", "pra", "pro",
	"pus", "qaa", "que", "raj", "rap", "rar", "roa", "roh", "rom", "rum",
	"run", "rup", "rus", "sad", "sag", "sah", "sai", "sal", "sam", "san",
	"sas", "sat", "scn", "sco", "sel", "sem", "sga", "sgn", "shn", "sid",
	"sin", "sio", "sit", "sla", "slo", "slv", "sma", "sme", "smi", "smj",
	"smn", "smo", "sms", "sna", "snd", "snk", "sog", "som", "son", "sot",
	"spa", "srd", "srn", "srp", "srr", "ssa", "ssw", "suk", "sun", "sus",
	"sux", "swa", "swe", "syc", "syr", "tah", "tai", "tam", "tat", "tel",
	"tem", "ter", "tet", "tgk", "tgl", "tha", "tib", "tig", "tir", "tiv",
	"tkl", "tlh", "tli", "tmh", "tog", "ton", "tpi", "tsi", "tsn", "tso",
	"tuk", "tum", "tup", "tur", "tut", "tvl", "twi", "tyv", "udm", "uga",
	"uig", "ukr", "umb", "und", "urd", "uzb", "vai", "ven", "vie", "vol",
	"vot", "wak", "wal", "war", "was", "wel", "wen", "wln", "wol", "xal",
	"xho", "yao", "yap", "yid", "yor", "ypk", "zap", "zbl", "zen", "zha",
	"znd", "zul", "zun", "zxx", "zza",
}
