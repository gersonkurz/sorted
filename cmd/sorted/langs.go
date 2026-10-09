package main

import (
	"flag"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/gersonkurz/sorted/internal/render"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// language is a language Sorted! speaks, or will: every name it has in every
// language Sorted! speaks or will (#43). Being non-judgmental, any of them
// selects it, and none is the official one.
type language struct {
	code   string // ISO 639-1, the neutral name
	names  []string
	lang   render.Lang
	spoken bool // false: named here, but not spoken yet
}

// languages names each language in English, German, French, Italian,
// Portuguese, Japanese and Mandarin, in that order, plus regional names.
var languages = []language{
	{"en", []string{"English", "Englisch", "anglais", "inglese", "inglês", "英語", "eigo", "英语", "yīngyǔ"}, render.English, true},
	{"de", []string{"German", "Deutsch", "allemand", "tedesco", "alemão", "ドイツ語", "doitsugo", "德语", "déyǔ"}, render.German, true},
	{"fr", []string{"French", "Französisch", "français", "francese", "francês", "フランス語", "furansugo", "法语", "fǎyǔ", "vaudois", "Waadtländisch"}, render.French, true},
	{"it", []string{"Italian", "Italienisch", "italien", "italiano", "italiano", "イタリア語", "itariago", "意大利语", "yìdàlìyǔ"}, render.Italian, true},
	{"pt", []string{"Portuguese", "Portugiesisch", "portugais", "portoghese", "português", "ポルトガル語", "porutogarugo", "葡萄牙语", "pútáoyáyǔ", "brasileiro"}, 0, false},
	{"ja", []string{"Japanese", "Japanisch", "japonais", "giapponese", "japonês", "日本語", "nihongo", "日语", "rìyǔ"}, 0, false},
	{"zh", []string{"Mandarin", "Chinese", "Chinesisch", "chinois", "mandarin", "cinese", "mandarino", "chinês", "mandarim", "中国語", "chūgokugo", "汉语", "hànyǔ", "中文", "zhōngwén", "普通话", "pǔtōnghuà"}, 0, false},
}

// fold folds case the Unicode way (ß becomes ss); syntax.Unaccent strips
// accents, as Very Very Sorted! reads its text (kana voicing stays: ド is
// not ト).
var fold = cases.Fold()

// keys are the spellings a name matches, ignoring case and accents, however
// they are encoded: plain, and with an umlaut (a combining diaeresis once
// decomposed) spelled as its vowel plus e ("franzosisch", "franzoesisch").
func keys(name string) [2]string {
	d := norm.NFD.String(fold.String(name))
	return [2]string{syntax.Unaccent(d), syntax.Unaccent(strings.ReplaceAll(d, "\u0308", "e"))}
}

// findLanguage looks a language up by any of its names or its code.
func findLanguage(name string) (language, bool) {
	k := keys(name)
	for _, l := range languages {
		if k[0] == l.code {
			return l, true
		}
		for _, n := range l.names {
			nk := keys(n)
			if k[0] == nk[0] || k[1] == nk[1] {
				return l, true
			}
		}
	}
	return language{}, false
}

// languageFlags takes the --NAME flags (--deutsch, --anglais, --英語) out of
// args, where the flag package would see a flag: not as the value of a
// flag that takes one (--dump --english writes to "--english"), and not
// after the first argument that is no flag, or after "--".
func languageFlags(fs *flag.FlagSet, args []string) (rest, names []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, isFlag := strings.CutPrefix(a, "--")
		if !isFlag {
			name, isFlag = strings.CutPrefix(a, "-")
		}
		if !isFlag || name == "" || strings.HasPrefix(name, "-") { // an argument, or "--"
			return append(rest, args[i:]...), names
		}
		key, _, hasValue := strings.Cut(name, "=")
		if f := fs.Lookup(key); f != nil {
			rest = append(rest, a)
			if b, ok := f.Value.(interface{ IsBoolFlag() bool }); !hasValue && !(ok && b.IsBoolFlag()) && i+1 < len(args) {
				i++
				rest = append(rest, args[i])
			}
			continue
		}
		if _, ok := findLanguage(name); ok {
			names = append(names, name)
			continue
		}
		rest = append(rest, a) // an unknown flag: the flag package reports it
	}
	return rest, names
}

// languageHelp lists every name of every language, for the usage text.
func languageHelp() string {
	var b strings.Builder
	b.WriteString("NAME is any language Sorted! speaks, named in any language it speaks or will:\n")
	for _, l := range languages {
		state := ""
		if !l.spoken {
			state = " (not yet)"
		}
		b.WriteString("  " + l.code + state + ": " + strings.Join(l.names, ", ") + "\n")
	}
	return b.String()
}
