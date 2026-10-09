package numbers

import (
	"math"
	"strings"
	"testing"
)

func TestJapaneseCardinal(t *testing.T) {
	for _, tt := range []struct {
		n    int32
		want string
	}{
		{0, "zero"}, {1, "ichi"}, {4, "yon"}, {7, "nana"}, {9, "kyū"}, {10, "jū"}, {11, "jūichi"}, {20, "nijū"}, {23, "nijūsan"},
		{99, "kyūjūkyū"}, {100, "hyaku"}, {234, "nihyakusanjūyon"}, {300, "sanbyaku"}, {600, "roppyaku"}, {800, "happyaku"},
		{1000, "sen"}, {3000, "sanzen"}, {8000, "hassen"}, {9999, "kyūsenkyūhyakukyūjūkyū"}, {10000, "ichiman"},
		{10001, "ichimanichi"}, {12345, "ichimannisensanbyakuyonjūgo"}, {100000, "jūman"}, {1000000, "hyakuman"},
		{10000000, "issenman"}, {11000000, "senhyakuman"}, {100000000, "ichioku"}, {500000000, "gooku"},
		{600000000, "rokuoku"}, {1000000000, "jūoku"},
		{math.MaxInt32, "nijūichiokuyonsennanahyakuyonjūhachimansanzenroppyakuyonjūnana"},
		{-7, "mainasu nana"}, {math.MinInt32, "mainasu nijūichiokuyonsennanahyakuyonjūhachimansanzenroppyakuyonjūhachi"},
	} {
		if got := JapaneseCardinal(tt.n); got != tt.want {
			t.Errorf("JapaneseCardinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestJapaneseOrdinalAndCount(t *testing.T) {
	for _, tt := range []struct {
		n            int32
		ord, counted string
	}{
		{0, "dai-zero", "zeroko"}, {1, "dai-ichi", "ikko"}, {2, "dai-ni", "niko"}, {3, "dai-san", "sanko"}, {6, "dai-roku", "rokko"},
		{8, "dai-hachi", "hakko"}, {10, "dai-jū", "jukko"}, {21, "dai-nijūichi", "nijūikko"}, {100, "dai-hyaku", "hyakko"},
		{1000, "dai-sen", "senko"}, {-3, "mainasu dai-san", "mainasu sanko"},
		{300, "dai-sanbyaku", "sanbyakko"}, {600, "dai-roppyaku", "roppyakko"}, {800, "dai-happyaku", "happyakko"},
		{200, "dai-nihyaku", "nihyakko"}, {1300, "dai-sensanbyaku", "sensanbyakko"}, {106, "dai-hyakuroku", "hyakurokko"},
		{110, "dai-hyakujū", "hyakujukko"}, {108, "dai-hyakuhachi", "hyakuhakko"},
	} {
		if got := JapaneseOrdinal(tt.n); got != tt.ord {
			t.Errorf("JapaneseOrdinal(%d) = %q, want %q", tt.n, got, tt.ord)
		}
		if got := JapaneseCount(tt.n); got != tt.counted {
			t.Errorf("JapaneseCount(%d) = %q, want %q", tt.n, got, tt.counted)
		}
	}
}

// The parser reads what the formatter writes, without accents and with
// long vowels doubled.
func TestJapaneseRoundTrip(t *testing.T) {
	for _, n := range italianSamples() {
		for _, w := range []string{plainJa(JapaneseCardinal(n)), doubled(JapaneseCardinal(n))} {
			if v, next, ok := ParseJapaneseCardinal(w+" kazu", 0); !ok || v != n || next != len(w) {
				t.Fatalf("ParseJapaneseCardinal(%q) = %d, %d, %v", w, v, next, ok)
			}
		}
		if n < 1 {
			continue
		}
		o := plainJa(JapaneseOrdinal(n))
		if v, next := ParseJapaneseOrdinal(" "+o+" no kazu", 0); v != n || next != len(o)+1 {
			t.Fatalf("ParseJapaneseOrdinal(%q) = %d, %d", o, v, next)
		}
		c := plainJa(JapaneseCount(n))
		if v, next, ok := ParseJapaneseCount(c+" tsukaimasu", 0); !ok || v != n || next != len(c) {
			t.Fatalf("ParseJapaneseCount(%q) = %d, %d, %v", c, v, next, ok)
		}
	}
}

func plainJa(w string) string { return JaPlain(w) }

// doubled writes long vowels as they are typed: "juu", "jou".
func doubled(w string) string { return strings.NewReplacer("ū", "uu", "ō", "ou").Replace(w) }

func TestParseJapanese(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int32
		ok   bool
	}{
		{"shi", 4, true}, {"shichi", 7, true}, {"ku", 9, true}, {"shichijuku", 79, true}, {"issen", 1000, true},
		{"issenman", 10000000, true}, {"senman", 10000000, true}, {"rei", 0, true}, {"JUUICHI", 11, true},
		{"nijuuichi", 21, true}, {"nijuichi", 21, true}, {"ichimanman", 0, false}, {"man", 0, false}, {"oku", 0, false},
		{"sanjuoku", 0, false}, {"kazu", 0, false}, {"", 0, false}, {"ni ni", 2, true},
		{"nijuichioku", 2100000000, true}, {"nijuniokuman", 0, false},
	} {
		v, next, ok := ParseJapaneseCardinal(tt.s, 0)
		if ok != tt.ok || ok && v != tt.want || !ok && next != 0 {
			t.Errorf("ParseJapaneseCardinal(%q) = %d, %d, %v", tt.s, v, next, ok)
		}
	}
	for _, tt := range []struct {
		s    string
		want int32
	}{{"daiichi no", 1}, {"dai ichi no", 1}, {"dai-nijuusan", 23}, {"dai-zero", 0}, {"dai", 0}, {"dainyu", 0}, {"ichi", 0}} {
		if v, _ := ParseJapaneseOrdinal(tt.s, 0); v != tt.want {
			t.Errorf("ParseJapaneseOrdinal(%q) = %d, want %d", tt.s, v, tt.want)
		}
	}
	for _, tt := range []struct {
		s    string
		want int32
	}{{"hachiko", 8}, {"jikko", 10}, {"goko", 5}, {"ni", 0}, {"ko", 0}, {"sanbyakko", 300}, {"roppyakko", 600},
		{"happyakko", 800}, {"sensanbyakko", 1300}, {"hyakuhachiko", 108}, {"sanbyakuko", 300}} { // lenient: without the sound change too
		if v, _, _ := ParseJapaneseCount(tt.s, 0); v != tt.want {
			t.Errorf("ParseJapaneseCount(%q) = %d, want %d", tt.s, v, tt.want)
		}
	}
}
