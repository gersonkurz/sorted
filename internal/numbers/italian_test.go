package numbers

import (
	"math"
	"strings"
	"testing"
)

func TestItalianCardinal(t *testing.T) {
	for _, tt := range []struct {
		n    int32
		want string
	}{
		{0, "zero"}, {1, "uno"}, {3, "tre"}, {8, "otto"}, {11, "undici"}, {17, "diciassette"},
		{20, "venti"}, {21, "ventuno"}, {23, "ventitré"}, {28, "ventotto"}, {80, "ottanta"}, {99, "novantanove"},
		{100, "cento"}, {101, "centouno"}, {103, "centotré"}, {108, "centootto"}, {180, "centottanta"},
		{188, "centottantotto"}, {200, "duecento"}, {883, "ottocentottantatré"},
		{1000, "mille"}, {1001, "milleuno"}, {2000, "duemila"}, {21000, "ventunomila"}, {23000, "ventitremila"},
		{1000000, "unmilione"}, {2000000, "duemilioni"}, {1234567, "unmilioneduecentotrentaquattromilacinquecentosessantasette"},
		{1000000000, "unmiliardo"}, {math.MaxInt32, "duemiliardicentoquarantasettemilioniquattrocentottantatremilaseicentoquarantasette"},
		{-7, "meno sette"}, {math.MinInt32, "meno duemiliardicentoquarantasettemilioniquattrocentottantatremilaseicentoquarantotto"},
	} {
		if got := ItalianCardinal(tt.n); got != tt.want {
			t.Errorf("ItalianCardinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestItalianOrdinal(t *testing.T) {
	for _, tt := range []struct {
		n    int32
		want string
	}{
		{0, "zeresimo"}, {1, "primo"}, {3, "terzo"}, {8, "ottavo"}, {10, "decimo"}, {11, "undicesimo"},
		{20, "ventesimo"}, {21, "ventunesimo"}, {23, "ventitreesimo"}, {26, "ventiseiesimo"}, {28, "ventottesimo"},
		{100, "centesimo"}, {101, "centunesimo"}, {103, "centotreesimo"}, {108, "centottesimo"}, {110, "centodiecesimo"},
		{180, "centottantesimo"}, {1000, "millesimo"}, {1001, "milleunesimo"}, {2000, "duemillesimo"},
		{1001000, "unmilionemillesimo"}, {1000000, "milionesimo"}, {2000000, "duemilionesimo"},
		{1000000000, "miliardesimo"}, {-7, "meno settimo"},
	} {
		if got := ItalianOrdinal(tt.n, false); got != tt.want {
			t.Errorf("ItalianOrdinal(%d) = %q, want %q", tt.n, got, tt.want)
		}
		if got, want := ItalianOrdinal(tt.n, true), tt.want[:len(tt.want)-1]+"a"; got != want {
			t.Errorf("ItalianOrdinal(%d, feminine) = %q, want %q", tt.n, got, want)
		}
	}
}

// italianSamples are 0 to 30000, and from there to math.MaxInt32 in steps
// that hit every digit position.
func italianSamples() []int32 {
	var ns []int32
	for n := int32(0); n <= 30000; n++ {
		ns = append(ns, n)
	}
	for n := int64(30001); n <= math.MaxInt32; n = n*13/10 + 7 {
		ns = append(ns, int32(n), int32(n/1000*1000), int32(n/1000000*1000000+n%1000))
	}
	return append(ns, math.MaxInt32)
}

// The parser reads what the formatter writes, as the Very Very Sorted!
// parser sees it: without accents.
func TestItalianRoundTrip(t *testing.T) {
	plain := strings.NewReplacer("é", "e")
	for _, n := range italianSamples() {
		w := plain.Replace(ItalianCardinal(n))
		if v, next, ok := ParseItalianCardinal(w+" etichette", 0); !ok || v != n || next != len(w) {
			t.Fatalf("ParseItalianCardinal(%q) = %d, %d, %v", w, v, next, ok)
		}
		if n < 1 {
			continue
		}
		for _, fem := range []bool{false, true} {
			o := ItalianOrdinal(n, fem)
			if v, next := ParseItalianOrdinal(" "+o+" numero", 0); v != n || next != len(o)+1 {
				t.Fatalf("ParseItalianOrdinal(%q) = %d, %d", o, v, next)
			}
		}
	}
}

func TestParseItalian(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int32
		next int // -1: fails
	}{
		{"un etichetta", 1, 2},
		{"una", 1, 3},
		{"un milione duecentomila, due", 1200000, 23},
		{"due milioni", 2000000, 11},
		{"duemilioni tre", 2000003, 14},
		{"un miliardo un milione", 1001000000, 22},
		{"centuno", 101, 7},
		{"centotto", 108, 8},
		{"ventiotto", 28, 9},
		{"ventitre e", 23, 8},
		{"zero", 0, 4},
		{"centdue", 0, -1},
		{"duemille", 0, -1},
		{"milione", 0, -1},
		{"unmilioni", 0, -1},
		{"unomila", 0, -1},
		{"tremiliardi", 0, -1}, // above math.MaxInt32
		{"numero", 0, -1},
		{"", 0, -1},
	} {
		v, next, ok := ParseItalianCardinal(tt.s, 0)
		if tt.next < 0 {
			if ok || next != 0 {
				t.Errorf("ParseItalianCardinal(%q) = %d, %d, %v; want failure", tt.s, v, next, ok)
			}
		} else if !ok || v != tt.want || next != tt.next {
			t.Errorf("ParseItalianCardinal(%q) = %d, %d, %v; want %d, %d", tt.s, v, next, ok, tt.want, tt.next)
		}
	}
	for _, tt := range []struct {
		s    string
		want int32
	}{
		{"prima", 1}, {"decima", 10}, {"centounesimo", 101}, {"duemillesima", 2000}, {"unmilionesimo", 1000000},
		{"zeresimo", 0}, {"tresimo", 0}, {"primi", 0}, {"esimo", 0}, {"numero", 0}, {"uno", 0},
	} {
		if v, _ := ParseItalianOrdinal(tt.s, 0); v != tt.want {
			t.Errorf("ParseItalianOrdinal(%q) = %d, want %d", tt.s, v, tt.want)
		}
	}
}
