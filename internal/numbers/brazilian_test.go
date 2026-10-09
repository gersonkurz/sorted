package numbers

import (
	"math"
	"testing"
)

// The expected words follow CLDR's Portuguese spellout rules
// (common/rbnf/pt.xml), apart from "zerésimo", where CLDR says "zero".
func TestBrazilianCardinal(t *testing.T) {
	for _, tt := range []struct {
		n    int32
		want string
	}{
		{0, "zero"}, {1, "um"}, {2, "dois"}, {3, "três"}, {10, "dez"}, {14, "catorze"}, {16, "dezesseis"}, {17, "dezessete"},
		{19, "dezenove"}, {20, "vinte"}, {21, "vinte e um"}, {99, "noventa e nove"}, {100, "cem"}, {101, "cento e um"},
		{110, "cento e dez"}, {200, "duzentos"}, {234, "duzentos e trinta e quatro"}, {500, "quinhentos"},
		{1000, "mil"}, {1001, "mil e um"}, {1100, "mil e cem"}, {1101, "mil cento e um"}, {1234, "mil duzentos e trinta e quatro"},
		{2000, "dois mil"}, {21000, "vinte e um mil"}, {200000, "duzentos mil"}, {1000000, "um milhão"},
		{1000100, "um milhão e cem"}, {1200000, "um milhão e duzentos mil"}, {1234000, "um milhão e duzentos e trinta e quatro mil"},
		{1234567, "um milhão duzentos e trinta e quatro mil quinhentos e sessenta e sete"},
		{2000000, "dois milhões"}, {2001234, "dois milhões mil duzentos e trinta e quatro"}, {1000000000, "um bilhão"},
		{math.MaxInt32, "dois bilhões cento e quarenta e sete milhões quatrocentos e oitenta e três mil seiscentos e quarenta e sete"},
		{-7, "menos sete"}, {math.MinInt32, "menos dois bilhões cento e quarenta e sete milhões quatrocentos e oitenta e três mil seiscentos e quarenta e oito"},
	} {
		if got := BrazilianCardinal(tt.n); got != tt.want {
			t.Errorf("BrazilianCardinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestBrazilianOrdinal(t *testing.T) {
	for _, tt := range []struct {
		n         int32
		want, fem string
	}{
		{0, "zerésimo", "zerésima"}, {1, "primeiro", "primeira"}, {2, "segundo", "segunda"}, {7, "sétimo", "sétima"},
		{10, "décimo", "décima"}, {11, "décimo primeiro", "décima primeira"}, {23, "vigésimo terceiro", "vigésima terceira"},
		{70, "septuagésimo", "septuagésima"}, {100, "centésimo", "centésima"}, {300, "tricentésimo", "tricentésima"},
		{999, "noningentésimo nonagésimo nono", "noningentésima nonagésima nona"},
		{1000, "milésimo", "milésima"}, {1001, "milésimo primeiro", "milésima primeira"},
		{2000, "dois milésimo", "duas milésima"}, {2345, "dois milésimo tricentésimo quadragésimo quinto", "duas milésima tricentésima quadragésima quinta"},
		{200000, "duzentos milésimo", "duzentas milésima"}, {1000000, "um milionésimo", "uma milionésima"},
		{2000000, "dois milionésimo", "duas milionésima"}, {1000000000, "um bilionésimo", "uma bilionésima"},
		{1234567, "um milionésimo duzentos e trinta e quatro milésimo quingentésimo sexagésimo sétimo",
			"uma milionésima duzentas e trinta e quatro milésima quingentésima sexagésima sétima"},
		{-3, "menos terceiro", "menos terceira"},
	} {
		if got := BrazilianOrdinal(tt.n, false); got != tt.want {
			t.Errorf("BrazilianOrdinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
		if got := BrazilianOrdinal(tt.n, true); got != tt.fem {
			t.Errorf("BrazilianOrdinal(%d, feminine) = %q, want %q", tt.n, got, tt.fem)
		}
	}
}

// The parser reads what the formatter writes, as Very Very Sorted! sees it:
// without accents.
func TestBrazilianRoundTrip(t *testing.T) {
	for _, n := range italianSamples() {
		w := plain(BrazilianCardinal(n))
		if v, next, ok := ParseBrazilianCardinal(w+" rotulos", 0); !ok || v != n || next != len(w) {
			t.Fatalf("ParseBrazilianCardinal(%q) = %d, %d, %v", w, v, next, ok)
		}
		if n < 1 {
			continue
		}
		for _, fem := range []bool{false, true} {
			o := plain(BrazilianOrdinal(n, fem))
			if v, next := ParseBrazilianOrdinal(" "+o+" numero", 0); v != n || next != len(o)+1 {
				t.Fatalf("ParseBrazilianOrdinal(%q) = %d, %d", o, v, next)
			}
		}
	}
}

func TestParseBrazilian(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int32
		ok   bool
		rest string // what is left
	}{
		{"uma", 1, true, ""}, {"duas", 2, true, ""}, {"quatorze", 14, true, ""}, {"duzentas e uma", 201, true, ""},
		{"dois milhao", 2000000, true, ""}, {"um milhoes", 1000000, true, ""}, {"mil e duzentos e trinta", 1230, true, ""},
		{"zero", 0, true, ""}, {"vinte e um, dois", 21, true, ", dois"},
		// greedy, but "e" joins only what can follow
		{"dois e tres", 2, true, " e tres"}, {"cem e um", 100, true, " e um"}, {"vinte e trinta", 20, true, " e trinta"},
		{"duzentos e um", 201, true, ""}, {"mil e mil", 1000, true, " e mil"}, {"vinte e", 20, true, " e"},
		// Portugal's words are not numbers here
		{"dezasseis", 0, false, "dezasseis"}, {"dezanove", 0, false, "dezanove"}, {"mil milhoes", 1000, true, " milhoes"},
		{"um mil", 1, true, " mil"}, {"cento", 0, false, "cento"}, {"milhao", 0, false, "milhao"}, {"numero", 0, false, "numero"},
		{"", 0, false, ""}, {"tres bilhoes", 0, false, "tres bilhoes"},
	} {
		v, next, ok := ParseBrazilianCardinal(tt.s, 0)
		if ok != tt.ok || v != tt.want || tt.s[next:] != tt.rest {
			t.Errorf("ParseBrazilianCardinal(%q) = %d, %q, %v", tt.s, v, tt.s[next:], ok)
		}
	}
	for _, tt := range []struct {
		s    string
		want int32
	}{
		{"primeira", 1}, {"vigesima terceira", 23}, {"milionesimo", 1000000}, {"duas milesima", 2000}, {"um milesimo", 0},
		{"zeresimo", 0}, {"numero", 0}, {"dois", 0}, {"segundo numero", 2}, {"decimo e primeiro", 10},
	} {
		if v, _ := ParseBrazilianOrdinal(tt.s, 0); v != tt.want {
			t.Errorf("ParseBrazilianOrdinal(%q) = %d, want %d", tt.s, v, tt.want)
		}
	}
}
