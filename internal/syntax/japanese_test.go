package syntax

import (
	"reflect"
	"strings"
	"testing"
)

// japanese says what english (italian_test.go) says, in Japanese romaji
// (#32).
const japanese = `Kono puroguramu wa kazu zero, ichi, nijūsan to hyakuman o tsukaimasu.
Kono puroguramu wa itsumo dai-ichi no raberu ni tobi, dai-ichi no jōken ga shin nara dai-ni no raberu ni tobimasu.
Kono puroguramu wa dai-san no kazu o itariago no kisū to shite kakimasu.
Kono puroguramu wa dai-ni no kazu ga sasu seru o moji to shite yomimasu.
Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no wa to, dai-hachi no seru to dai-ichi no wa no wa o tsukaimasu.
Kono puroguramu wa dai-ichi no kazu ga dai-ni no kazu to hitoshii to iu jōken to, dai-ichi no wa ga dai-ichi no kazu ga sasu seru yori chiisai to iu jōken o tsukaimasu.
Kono puroguramu wa raberu o niko tsukaimasu.
Kono puroguramu wa dai-ichi no kazu to dai-jūichi no kazu no sa o tsukaimasu.
Kono puroguramu wa dai-ichi no kazu o dai-ni no kazu ni dainyū shi, dai-ichi no wa o dai-ichi no kazu ga sasu seru ni dainyū shi, dai-ichi no kazu o dai-yon no kazu ni dainyū shimasu.
Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no seki to, dai-ichi no seki to dai-ichi no hi no seki o tsukaimasu.
Kono puroguramu wa dai-ichi no dainyū, dai-ichi no raberu, dai-ichi no nyūryoku, dai-ichi no shutsuryoku, dai-ni no janpu to dai-ni no raberu o jissō shimasu.
Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no hi o tsukaimasu.
Kono puroguramu wa dai-ichi no kazu demo dai-ni no kazu demo nai ronri enzan to, dai-ichi no ronri enzan to dai-san no kazu no ryōhō de wa nai ronri enzan o tsukaimasu.
Kono puroguramu wa totemo totemo kakkoii desu.`

// Japanese parses into exactly the tables of its English twin, layout and
// scratch slots included.
func TestJapanese(t *testing.T) {
	want, err := parse(english)
	if err != nil {
		t.Fatalf("english: %v", err)
	}
	for _, tt := range []struct{ name, src string }{
		{"as written", japanese},
		{"without macrons", strings.NewReplacer("ū", "u", "ō", "o").Replace(japanese)},
		{"long vowels typed", strings.NewReplacer("ū", "uu", "ō", "ou").Replace(japanese)},
		{"upper case", strings.ToUpper(japanese)},
		{"wo", strings.Replace(japanese, "hyakuman o tsukaimasu", "hyakuman wo tsukaimasu", 1)},
		{"commas", strings.NewReplacer("nijūsan to hyakuman", "nijūsan, to hyakuman", "no wa to, dai-hachi", "no wa, to dai-hachi",
			"jōken to, dai-ichi", "jōken to dai-ichi", "dai-ni no janpu to", "dai-ni no janpu, to").Replace(japanese)},
		{"dai", strings.NewReplacer("dai-san no kazu o itariago", "daisan no kazu o itariago", "dai-jūichi", "dai jūichi").Replace(japanese)},
		{"verb forms", strings.NewReplacer("ni tobi,", "ni tobimasu,", "kakimasu.", "kaki.", "yomimasu.", "yomi.", "dainyū shi, dai-ichi no wa", "dainyū shimasu, dai-ichi no wa").Replace(japanese)},
		{"short marker", strings.Replace(japanese, "Kono puroguramu wa totemo totemo kakkoii desu.", "Totemo totemo kakkoii.", 1)},
		{"Portuguese marker", strings.Replace(japanese, "Kono puroguramu wa totemo totemo kakkoii desu.", "Muito muito legal.", 1)},
	} {
		got, err := parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tt.name, got, want)
		}
	}
}

// Every sentence also has a form that says there is none, and those that
// have one, a single entry.
func TestJapaneseNone(t *testing.T) {
	src := `Kono puroguramu wa kazu nana o tsukaimasu.
Kono puroguramu wa doko ni mo ikimasen.
Kono puroguramu wa kakemasen.
Kono puroguramu wa yomemasen.
Kono puroguramu wa dai-ichi no kazu to dai-ichi no kazu no wa o tsukaimasu.
Kono puroguramu wa jōken o tsukaimasen.
Kono puroguramu wa raberu o ikko tsukaimasu.
Kono puroguramu wa sa o tsukaimasen.
Kono puroguramu wa dai-ichi no wa o dai-ichi no kazu ni dainyū shimasu.
Kono puroguramu wa seki o tsukaimasen.
Kono puroguramu wa dai-ichi no dainyū o jissō shimasu.
Kono puroguramu wa hi o tsukaimasen.
Kono puroguramu wa hironriteki desu.
Totemo totemo kakkoii.`
	p, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.Data[0] != 7 || p.LabelsCount != 1 || p.Tables[Sums].Count != 1 || p.Tables[Assigns].Count != 1 || p.Tables[Statements].Count != 1 {
		t.Errorf("%+v", p)
	}
	for _, tt := range []struct{ from, to string }{
		{"kazu nana o tsukaimasu", "kazu o tsukaimasen"},
		{"dai-ichi no kazu to dai-ichi no kazu no wa o tsukaimasu", "wa o tsukaimasen"},
		{"raberu o ikko tsukaimasu", "raberu o tsukaimasen"},
		{"raberu o ikko tsukaimasu", "raberu o nijūikko tsukaimasu"},
		{"raberu o ikko tsukaimasu", "raberu o hachiko tsukaimasu"},
		{"dai-ichi no wa o dai-ichi no kazu ni dainyū shimasu", "dainyū shimasen"},
		{"hironriteki desu", "ronri enzan o tsukaimasen"},
		{"jōken o tsukaimasen", "dai-ichi no kazu ga dai-ichi no kazu yori chiisai to iu jōken o tsukaimasu"},
		{"doko ni mo ikimasen", "itsumo dai-ichi no raberu ni tobimasu"},
		{"sa o tsukaimasen", "dai-ichi no kazu to dai-ichi no kazu no sa to, dai-ichi no wa to dai-ichi no wa no sa o tsukaimasu"},
		{"seki o tsukaimasen", "dai-ichi no kazu to dai-ichi no kazu no seki o tsukaimasu"},
		{"hi o tsukaimasen", "dai-ichi no kazu to dai-ichi no kazu no hi, dai-ichi no kazu to dai-ichi no kazu no hi to dai-ichi no kazu to dai-ichi no kazu no hi o tsukaimasu"},
		{"hironriteki desu", "dai-ichi no kazu to dai-ichi no kazu no ryōhō de wa nai ronri enzan o tsukaimasu"},
		{"kakemasen", "dai-ichi no kazu o burajiru no josū to shite kakimasu"},
		{"yomemasen", "dai-ichi no kazu o moji to shite yomimasu"},
	} {
		if _, err := parse(strings.Replace(src, tt.from, tt.to, 1)); err != nil {
			t.Errorf("%s: %v", tt.to, err)
		}
	}
}

// Japanese is Very Very Sorted!, mixes with the other languages, and broken
// Japanese fails.
func TestJapaneseDialect(t *testing.T) {
	for _, end := range []string{"Cool.", "This code is very cool.", "Kono puroguramu wa totemo kakkoii desu."} {
		if _, err := parse(strings.Replace(japanese, "Kono puroguramu wa totemo totemo kakkoii desu.", end, 1)); err == nil || err.Error() != "ERROR, missing or invalid number declaration" {
			t.Errorf("%s: %v", end, err)
		}
	}
	lines := strings.Split(japanese, "\n")
	for _, other := range []string{english, italian, french, portuguese} {
		o := strings.Split(other, "\n")
		for i := range lines {
			mixed := append(append(append([]string{}, lines[:i]...), o[i]), lines[i+1:]...)
			if _, err := parse(strings.Join(mixed, "\n")); err != nil {
				t.Errorf("%s sentence %d: %v", o[0][:5], i+1, err)
			}
		}
	}
	for _, f := range []struct {
		phrase string
		format int32
	}{
		{"as a japanese cardinal", FormatJapaneseCardinal}, {"as a japanese ordinal", FormatJapaneseOrdinal},
		{"als ein japanischer Kardinal", FormatJapaneseCardinal}, {"als eine japanische Ordinalzahl", FormatJapaneseOrdinal},
		{"come cardinale giapponese", FormatJapaneseCardinal}, {"come ordinale giapponese", FormatJapaneseOrdinal},
		{"comme cardinal japonais", FormatJapaneseCardinal}, {"comme ordinal japonais", FormatJapaneseOrdinal},
		{"como cardinal japonês", FormatJapaneseCardinal}, {"como ordinal japonês", FormatJapaneseOrdinal},
	} {
		src := strings.Replace(minimal, "the first number as a character", "the first number "+f.phrase, 1)
		if _, err := parse(strings.Replace(src, "Cool.", "This code is very cool.", 1)); err == nil {
			t.Errorf("%s in Very Sorted!", f.phrase)
		}
		p, err := parse(strings.Replace(src, "Cool.", "Very very cool.", 1))
		if err != nil || p.Entries(Writes)[0].Flags != f.format {
			t.Errorf("%s: %v", f.phrase, err)
		}
	}
	for _, f := range jpFormats {
		src := strings.Replace(japanese, "o itariago no kisū to shite kakimasu", "o "+f.words+" to shite kakimasu", 1)
		if p, err := parse(src); err != nil || p.Entries(Writes)[0].Flags != f.format {
			t.Errorf("%s: %v", f.words, err)
		}
	}
	for _, tt := range []struct{ from, to string }{
		{"nijūsan to hyakuman", "nijūsan to nijūsan"},
		{"dai-ichi no raberu ni tobi", "dai-ichi no jōken ni tobi"},
		{"o itariago no kisū to shite", "o itariago no kisū"},
		{"dai-jūichi no kazu", "dai-jūichi"},
		{"dai-yon no kazu ni", "dai-yon no wa ni"},
		{"to hitoshii", "yori ōkii"},
		{"raberu o niko", "raberu o ni"},
		{"hyakuman", "jūoku"},                        // declared numbers stop below 1000000000
		{"nijūsan", "ni jūsan"},                      // a number is one word
		{"dai-ni no janpu to", "dai-ni no janpu de"}, // the list's "to"
		{"no ryōhō de wa nai", "no ryōhō nai"},
	} {
		if !strings.Contains(japanese, tt.from) {
			t.Fatalf("%q is not in the program", tt.from)
		}
		if _, err := parse(strings.Replace(japanese, tt.from, tt.to, 1)); err == nil {
			t.Errorf("%s parses", tt.to)
		}
	}
}
