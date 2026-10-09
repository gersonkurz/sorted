package numbers

import (
	"math"
	"strings"
	"testing"
)

func TestVaudoisCardinal(t *testing.T) {
	for _, tt := range []struct {
		n    int32
		want string
	}{
		{0, "zéro"}, {1, "un"}, {10, "dix"}, {16, "seize"}, {17, "dix-sept"}, {19, "dix-neuf"}, {20, "vingt"},
		{21, "vingt-et-un"}, {22, "vingt-deux"}, {61, "soixante-et-un"}, {70, "septante"}, {71, "septante-et-un"},
		{80, "huitante"}, {88, "huitante-huit"}, {91, "nonante-et-un"}, {99, "nonante-neuf"},
		{100, "cent"}, {101, "cent-un"}, {200, "deux-cents"}, {201, "deux-cent-un"}, {1000, "mille"}, {1001, "mille-un"},
		{2000, "deux-mille"}, {200000, "deux-cent-mille"}, {1000000, "un-million"}, {2000000, "deux-millions"},
		{200000000, "deux-cents-millions"}, {-200000000, "moins deux-cents-millions"}, {1000000000, "un-milliard"},
		{math.MaxInt32, "deux-milliards-cent-quarante-sept-millions-quatre-cent-huitante-trois-mille-six-cent-quarante-sept"},
		{-7, "moins sept"}, {math.MinInt32, "moins deux-milliards-cent-quarante-sept-millions-quatre-cent-huitante-trois-mille-six-cent-quarante-huit"},
	} {
		if got := VaudoisCardinal(tt.n); got != tt.want {
			t.Errorf("VaudoisCardinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestVaudoisOrdinal(t *testing.T) {
	for _, tt := range []struct {
		n         int32
		want, fem string
	}{
		{0, "zéroième", ""}, {1, "premier", "première"}, {2, "deuxième", ""}, {3, "troisième", ""}, {4, "quatrième", ""},
		{5, "cinquième", ""}, {9, "neuvième", ""}, {11, "onzième", ""}, {21, "vingt-et-unième", ""}, {25, "vingt-cinquième", ""},
		{80, "huitantième", ""}, {100, "centième", ""}, {200, "deux-centième", ""}, {1000, "millième", ""},
		{2000, "deux-millième", ""}, {1000000, "millionième", ""}, {2000000, "deux-millionième", ""},
		{1000000000, "milliardième", ""}, {-3, "moins troisième", ""},
		{200000000, "deux-cent-millionième", ""}, {-200000000, "moins deux-cent-millionième", ""},
		{200000003, "deux-cents-millions-troisième", ""}, {300000000, "trois-cent-millionième", ""},
		{2000000000, "deux-milliardième", ""}, {200001000, "deux-cents-millions-millième", ""},
	} {
		if got := VaudoisOrdinal(tt.n, false); got != tt.want {
			t.Errorf("VaudoisOrdinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
		fem := tt.fem
		if fem == "" {
			fem = tt.want
		}
		if got := VaudoisOrdinal(tt.n, true); got != fem {
			t.Errorf("VaudoisOrdinal(%d, feminine) = %q, want %q", tt.n, got, fem)
		}
	}
}

// The parser reads what the formatter writes, as Very Very Sorted! sees it:
// without accents.
func TestVaudoisRoundTrip(t *testing.T) {
	plain := strings.NewReplacer("é", "e", "è", "e")
	for _, n := range italianSamples() {
		w := plain.Replace(VaudoisCardinal(n))
		if v, next, ok := ParseVaudoisCardinal(w+" etiquettes", 0); !ok || v != n || next != len(w) {
			t.Fatalf("ParseVaudoisCardinal(%q) = %d, %d, %v", w, v, next, ok)
		}
		if n < 1 {
			continue
		}
		for _, fem := range []bool{false, true} {
			o := plain.Replace(VaudoisOrdinal(n, fem))
			if v, next := ParseVaudoisOrdinal(" "+o+" nombre", 0); v != n || next != len(o)+1 {
				t.Fatalf("ParseVaudoisOrdinal(%q) = %d, %d", o, v, next)
			}
		}
	}
}

func TestParseVaudois(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int32
		ok   bool
	}{
		{"une", 1, true}, {"vingt-et-une", 21, true}, {"deux-cent", 200, true}, {"deux-cents-mille", 200000, true},
		{"deux-million", 2000000, true}, {"vingt-un", 21, true}, {"zero", 0, true},
		{"un-mille", 0, false}, {"dix-un", 0, false}, {"vingt-et-deux", 0, false}, {"million", 0, false},
		{"trois-milliards", 0, false}, {"deux deux", 2, true}, {"nombre", 0, false}, {"", 0, false}, {"-", 0, false},
	} {
		v, next, ok := ParseVaudoisCardinal(tt.s, 0)
		if ok != tt.ok || ok && v != tt.want || !ok && next != 0 {
			t.Errorf("ParseVaudoisCardinal(%q) = %d, %d, %v", tt.s, v, next, ok)
		}
	}
	for _, tt := range []struct {
		s    string
		want int32
	}{{"premiere", 1}, {"seconde", 2}, {"deux-centsieme", 200}, {"unieme", 0}, {"zeroieme", 0}, {"ieme", 0}, {"nombre", 0}} {
		if v, _ := ParseVaudoisOrdinal(tt.s, 0); v != tt.want {
			t.Errorf("ParseVaudoisOrdinal(%q) = %d, want %d", tt.s, v, tt.want)
		}
	}
}
