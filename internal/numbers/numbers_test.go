package numbers

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// golden reads a capture from the original Sorted.exe, with CRLF normalised.
func golden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// goldenData returns the data initialiser of a captured /C output
// ("long _[193719] = { 0,1,72 };").
func goldenData(t *testing.T, program string) []int32 {
	t.Helper()
	src := golden(t, program+".c")
	_, after, ok := strings.Cut(src, "long _[193719] = { ")
	if !ok {
		t.Fatalf("%s.c: no data initialiser", program)
	}
	list, _, _ := strings.Cut(after, "}")
	var data []int32
	for _, f := range strings.Split(list, ",") {
		n, err := strconv.ParseInt(strings.TrimSpace(f), 10, 32)
		if err != nil {
			t.Fatalf("%s.c: %v", program, err)
		}
		data = append(data, int32(n))
	}
	return data
}

// cardinal parses like the parser's CARDINAL: English first, then German from
// wherever the English attempt left the cursor.
func cardinal(s string) (int32, int, bool) {
	v, next, ok := ParseEnglishCardinal(s, 0)
	if ok {
		return v, next, true
	}
	return ParseGermanCardinal(s, next)
}

// ordinal parses like the parser's DIRECT_USE: English first, then German from
// wherever the English attempt left the cursor.
func ordinal(s string) (int32, int) {
	v, next := ParseEnglishOrdinal(s, 0)
	if v != 0 {
		return v, next
	}
	return ParseGermanOrdinal(s, next)
}

// The number declarations of the four sample programs, as the parser sees
// them (separators already turned into spaces or kept as commas). Expected
// values are the data initialisers in the C code that Sorted.exe generated
// for each program (testdata/golden/<program>.c).
func TestSampleDeclarations(t *testing.T) {
	tests := []struct {
		program string
		words   []string
	}{
		{"hello.s",
			[]string{"zero", "one", "seventy two", "one hundred and one", "one hundred eight", "two hundred", "one hundred eleven", "fourtyfour", "thirtytwo", "eightyseven", "twohundred one", "onehundred fourteen", "two hundred two", "onehundred", "fourtysix", "twohundredtwentytwo"}},
		{"hallo.s",
			[]string{"Null", "Eins", "Zweiundsiebzig", "Siebenundneunzig", "Einhundertacht", "Zweihundert", "Einhundertelf", "Vierundvierzig", "Zweiunddreissig", "Siebenundachtzig", "Einhunderteins", "Zweihundertzwei", "Einhundertsechzehn", "Sechsundvierzig", "Zweihundertzweiundzwanzig"}},
		{"fibo.s",
			[]string{"twenty", "seventyone", "three", "two", "one"}},
		{"itoa.s",
			[]string{"fourtyone milliontwohundredeightyonethousand ninehundredtwentyseven", "ten", "eleven", "fourtyeight", "eighty", "seventynine", "one", "zero"}},
	}
	for _, tt := range tests {
		want := goldenData(t, strings.TrimSuffix(tt.program, ".s"))
		if len(want) != len(tt.words) {
			t.Fatalf("%s: %d words, capture has %d numbers", tt.program, len(tt.words), len(want))
		}
		for i, w := range tt.words {
			// Followed by a comma, as in the programs: the parse must stop there.
			s := w + ", rest"
			v, next, ok := cardinal(s)
			if !ok || v != want[i] {
				t.Errorf("%s: cardinal(%q) = %d, %v; want %d", tt.program, w, v, ok, want[i])
				continue
			}
			if s[next] != ',' {
				t.Errorf("%s: cardinal(%q) stopped at %q, want the comma", tt.program, w, s[next:])
			}
		}
	}
}

// Ordinal references of the sample programs. The German expectations are the
// zero-based indices in the /D dump of hallo.s (testdata/golden/hallo.dump)
// plus one; the English ones follow from the identical
// C output of hello.s.
func TestSampleOrdinals(t *testing.T) {
	tests := []struct {
		text string
		want int32
		rest string
	}{
		{"first number", 1, "number"},
		{"second label", 2, "label"},
		{"third assignment", 3, "assignment"},
		{"fourth assignment", 4, "assignment"},
		{"fifth number", 5, "number"},
		{"sixth number", 6, "number"},
		{"seventh assignment", 7, "assignment"},
		{"eight number", 8, "number"}, // itoa.s's typo works: plain cardinals are accepted
		{"ninth assignment", 9, "assignment"},
		{"tenth number", 10, "number"},
		{"eleventh number", 11, "number"},
		{"thirteenth number", 13, "number"},
		{"sixteenth number", 16, "number"},
		{"erste Zahl", 1, "Zahl"},
		{"zweite Zuweisung", 2, "Zuweisung"},
		{"dritte Zuweisung", 3, "Zuweisung"},
		{"vierte Zuweisung", 4, "Zuweisung"},
		{"fuenfte Zahl", 5, "Zahl"},
		{"sechsten Zahl", 6, "Zahl"},
		{"siebente Zahl", 7, "Zahl"},
		{"zwoelfte Zahl", 12, "Zahl"},
		{"fuenfzehnte Zahl", 15, "Zahl"},
		{"sechzehnte Zahl", 16, "Zahl"},
		{"zweihundertelfte Zahl", 211, "Zahl"},
		{"ersten Sprungziel", 1, "Sprungziel"},
	}
	for _, tt := range tests {
		v, next := ordinal(tt.text)
		if v != tt.want || tt.text[next:] != tt.rest {
			t.Errorf("ordinal(%q) = %d, rest %q; want %d, rest %q", tt.text, v, tt.text[next:], tt.want, tt.rest)
		}
	}
}

// fibo.s prints the Fibonacci numbers 1, 1, 2, 3, ... as English cardinals;
// the expected words are its output from Sorted.exe (testdata/golden/fibo.out).
func TestEnglishCardinalFiboCapture(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(golden(t, "fibo.out"), "\n"), "\n")
	a, b := int32(1), int32(1)
	for _, want := range lines {
		if got, err := EnglishCardinal(a); err != nil || got != want {
			t.Errorf("EnglishCardinal(%d) = %q, %v; want %q", a, got, err, want)
		}
		a, b = b, a+b
	}
	if len(lines) != 19 {
		t.Errorf("fibo.out has %d lines, want 19", len(lines))
	}
}

// Behaviour that follows from reading the C code (no capture yet).
func TestFormatFromCode(t *testing.T) {
	type fn func(int32) (string, error)
	tests := []struct {
		name string
		f    fn
		n    int32
		want string
	}{
		{"en", EnglishCardinal, 0, ""}, // zero prints as an empty line
		{"en", EnglishCardinal, 15, "fiveteen"},
		{"en", EnglishCardinal, 40, "fourty"},
		{"en", EnglishCardinal, 100, "onehundred"},
		{"en", EnglishCardinal, 1000000, "onemillion"},
		{"en", EnglishCardinal, 999999999, "ninehundredninetyninemillionninehundredninetyninethousandninehundredninetynine"},
		{"en", EnglishCardinal, 1000000000, ""},    // n %= 1000000000
		{"en", EnglishCardinal, 2000000001, "one"}, // likewise
		{"en", EnglishCardinal, math.MaxInt32, "onehundredfourtysevenmillionfourhundredeightythreethousandsixhundredfourtyseven"},
		{"de", GermanCardinal, 0, ""},
		{"de", GermanCardinal, 1, "eins"},
		{"de", GermanCardinal, 21, "einundzwanzig"},
		{"de", GermanCardinal, 101, "einhunderteins"},
		{"de", GermanCardinal, 1000, "einstausend"},
		{"de", GermanCardinal, 1000000, "einsmillionen"},
		{"de", GermanCardinal, 1000000000, ""},
		{"en ord", EnglishOrdinal, 1, "first"},
		{"en ord", EnglishOrdinal, 2, "second"},
		{"en ord", EnglishOrdinal, 3, "third"},
		{"en ord", EnglishOrdinal, 5, "fifth"},
		{"en ord", EnglishOrdinal, 8, "eighth"},
		{"en ord", EnglishOrdinal, 9, "nineth"},
		{"en ord", EnglishOrdinal, 11, "eleventh"},
		{"en ord", EnglishOrdinal, 12, "twelveth"},
		{"en ord", EnglishOrdinal, 15, "fiveteenth"},
		{"en ord", EnglishOrdinal, 20, "twentieth"},
		{"en ord", EnglishOrdinal, 21, "twentyfirst"},
		{"en ord", EnglishOrdinal, 40, "fourtieth"},
		{"en ord", EnglishOrdinal, 100, "onehundredth"},
		{"en ord", EnglishOrdinal, 1000, "onethousandth"},
		{"en ord", EnglishOrdinal, 1000000000, "th"}, // empty cardinal; UB in the original, port-defined: reads NUL
		{"en ord", EnglishOrdinal, 0, OrdinalError},
		{"en ord", EnglishOrdinal, -7, OrdinalError},
		{"de ord", GermanOrdinal, 1, "erste"},
		{"de ord", GermanOrdinal, 3, "dritte"},
		{"de ord", GermanOrdinal, 7, "siebente"},
		{"de ord", GermanOrdinal, 13, "dreizehnte"},
		{"de ord", GermanOrdinal, 20, "zwanzigste"},
		{"de ord", GermanOrdinal, 101, "einhunderterste"},
		{"de ord", GermanOrdinal, 211, "zweihundertelfte"},
		{"de ord", GermanOrdinal, 1000, "einstausendste"},
		{"de ord", GermanOrdinal, 1000000, "einsmillionenste"},
		{"de ord", GermanOrdinal, 1000000000, "ste"}, // likewise
		{"de ord", GermanOrdinal, 0, OrdinalError},
	}
	for _, tt := range tests {
		if got, err := tt.f(tt.n); err != nil || got != tt.want {
			t.Errorf("%s(%d) = %q, %v; want %q", tt.name, tt.n, got, err, tt.want)
		}
	}
}

// Negative cardinals are undefined behaviour in the original (negative table
// subscripts). By the maintainer's ruling they are not emulated: every one is
// ErrCrash. Negative ordinals are defined and print OrdinalError.
func TestNegativeCardinalsCrash(t *testing.T) {
	for _, n := range []int32{-1, -11, -38200000, math.MinInt32} {
		for name, f := range map[string]func(int32) (string, error){"en": EnglishCardinal, "de": GermanCardinal} {
			if got, err := f(n); !errors.Is(err, ErrCrash) {
				t.Errorf("%s(%d) = %q, %v; want ErrCrash", name, n, got, err)
			}
		}
	}
	for name, f := range map[string]func(int32) (string, error){"en": EnglishOrdinal, "de": GermanOrdinal} {
		if got, err := f(-1); err != nil || got != OrdinalError {
			t.Errorf("%s ordinal(-1) = %q, %v; want %q", name, got, err, OrdinalError)
		}
	}
}

// Parser behaviour that follows from reading the C code.
func TestParseFromCode(t *testing.T) {
	t.Run("cardinals", func(t *testing.T) {
		tests := []struct {
			name string
			f    func(string, int) (int32, int, bool)
			text string
			want int32
			ok   bool
			next int
		}{
			{"case-insensitive", ParseEnglishCardinal, "TWENTY one", 21, true, 10},
			{"fifteen is not a word", ParseEnglishCardinal, "fifteen", 0, false, 0},
			{"fiveteen is", ParseEnglishCardinal, "fiveteen", 15, true, 8},
			{"one hundred thousand is 100", ParseEnglishCardinal, "one hundred thousand", 100, true, 20},
			{"onehundredthousand is not", ParseEnglishCardinal, "onehundredthousand", 100000, true, 18},
			{"zero only before blanks", ParseEnglishCardinal, " zero", 0, false, 1},
			{"zero prefix", ParseEnglishCardinal, "zeroes", 0, true, 4},
			{"cursor moves on failure", ParseEnglishCardinal, "hundred and", 0, false, 11},
			{"stops at period", ParseEnglishCardinal, "seven.", 7, true, 5},
			{"int32 wraps", ParseEnglishCardinal, "ninehundred million ninehundred million ninehundred million", -1594967296, true, 59},
			{"and after a hundred word", ParseEnglishCardinal, "onehundred and one", 101, true, 18},
			{"german wraps negative and fails", ParseGermanCardinal, "neunhundertmillionen neunhundertmillionen neunhundertmillionen", -1594967296, false, 62},
			{"german separate hundert", ParseGermanCardinal, "zwei hundert", 200, true, 12},
			{"german cardinal", ParseGermanCardinal, "einhundertundzwanzig", 120, true, 20},
			{"null", ParseGermanCardinal, "Null", 0, true, 4},
			{"tausend", ParseGermanCardinal, "zweitausend", 2000, true, 11},
			{"millionen", ParseGermanCardinal, "dreimillionen", 3000000, true, 13},
			{"no number", ParseGermanCardinal, "Zahl", 0, false, 0},
		}
		for _, tt := range tests {
			v, next, ok := tt.f(tt.text, 0)
			if v != tt.want || ok != tt.ok || next != tt.next {
				t.Errorf("%s: parse(%q) = %d, %d, %v; want %d, %d, %v", tt.name, tt.text, v, next, ok, tt.want, tt.next, tt.ok)
			}
		}
	})
	t.Run("ordinals", func(t *testing.T) {
		tests := []struct {
			name string
			f    func(string, int) (int32, int)
			text string
			want int32
			next int
		}{
			{"twelfth is not a word", ParseEnglishOrdinal, "twelfth x", 0, 0},
			{"twelveth is", ParseEnglishOrdinal, "twelveth x", 12, 9},
			{"compound", ParseEnglishOrdinal, "twentysecond x", 22, 13},
			{"hundred and first", ParseEnglishOrdinal, "one hundred and first x", 101, 22},
			{"th needs a blank after it", ParseEnglishOrdinal, "fourth,", 4, 4},
			{"hundred word and ordinal", ParseEnglishOrdinal, "twohundred and first x", 201, 21},
			{"decade ordinal", ParseEnglishOrdinal, "twentieth x", 20, 10},
			{"compound cardinal as ordinal", ParseEnglishOrdinal, "twentytwo x", 22, 10},
			{"thousandth", ParseEnglishOrdinal, "onethousandth x", 1000, 14},
			{"millionth", ParseEnglishOrdinal, "two millionth x", 2000000, 14},
			{"h as the whole rest", ParseEnglishOrdinal, "onehundredh", 100, 11},
			{"german te ending", ParseGermanOrdinal, "zwanzigte x", 20, 10},
			// After a plain cardinal the German ordinal parser skips an ending, not
			// blanks, so a separate "hundert" ends the number.
			{"german separate hundert stops", ParseGermanOrdinal, "drei hundert zweite x", 3, 5},
			{"german hundert after cardinal", ParseGermanOrdinal, "elfhundert zweite x", 1102, 18},
			{"german hundertste after cardinal", ParseGermanOrdinal, "elfhundertste x", 1100, 14},
			{"german separate hundertste stops", ParseGermanOrdinal, "zwei hundertste x", 2, 5},
			{"german tausend then ordinal", ParseGermanOrdinal, "zweitausend erste x", 2001, 18},
			{"german tausendste", ParseGermanOrdinal, "zweitausendste x", 2000, 15},
			{"german millionste", ParseGermanOrdinal, "zweimillionste x", 2000000, 15},
			{"german compound", ParseGermanOrdinal, "einundzwanzigste x", 21, 17},
			{"german decade", ParseGermanOrdinal, "zwanzigste x", 20, 11},
			{"german hundredth", ParseGermanOrdinal, "einhundertste x", 100, 14},
			{"ordinal does not stop the search", ParseGermanOrdinal, "erstezwanzig x", 21, 13},
			{"millionen counted twice", ParseGermanOrdinal, "zweimillionenste", 2000002, 16},
		}
		for _, tt := range tests {
			v, next := tt.f(tt.text, 0)
			if v != tt.want || next != tt.next {
				t.Errorf("%s: parse(%q) = %d, %d; want %d, %d", tt.name, tt.text, v, next, tt.want, tt.next)
			}
		}
	})
}

// Every cardinal the formatter produces parses back, in both languages.
func TestCardinalRoundTrip(t *testing.T) {
	for n := int32(1); n < 1000000; n++ {
		for _, lang := range []struct {
			format func(int32) (string, error)
			parse  func(string, int) (int32, int, bool)
		}{{EnglishCardinal, ParseEnglishCardinal}, {GermanCardinal, ParseGermanCardinal}} {
			s, _ := lang.format(n)
			if v, next, ok := lang.parse(s, 0); !ok || v != n || next != len(s) {
				t.Fatalf("parse(format(%d) = %q) = %d, %d, %v", n, s, v, next, ok)
			}
		}
	}
}
