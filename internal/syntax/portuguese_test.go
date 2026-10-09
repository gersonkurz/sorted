package syntax

import (
	"reflect"
	"strings"
	"testing"
)

// portuguese says what english (italian_test.go) says, in Brazilian
// Portuguese (#31).
const portuguese = `Este programa usa os números zero, um, vinte e três e um milhão.
Este programa sempre vai para o primeiro rótulo e às vezes vai para o segundo rótulo se a primeira condição for verdadeira.
Este programa escreve o terceiro número como cardinal italiano.
Este programa lê a célula indexada pelo segundo número como caractere.
Este programa usa as somas do primeiro número e do segundo número, e da oitava célula e da primeira soma.
Este programa usa a condição de que o primeiro número seja igual ao segundo número e a condição de que a primeira soma seja menor que a célula indexada pelo primeiro número.
Este programa usa dois rótulos.
Este programa usa a diferença ordenada entre o primeiro número e o décimo primeiro número.
Este programa atribui o primeiro número ao segundo número, a primeira soma à célula indexada pelo primeiro número e o primeiro número ao quarto número.
Este programa usa os produtos do primeiro número e do segundo número, e do primeiro produto e da primeira razão.
Este programa implementa a primeira atribuição, o primeiro rótulo, a primeira entrada, a primeira saída, o segundo salto e o segundo rótulo.
Este programa usa a razão entre o primeiro número e o segundo número.
Este programa usa as operações lógicas nem o primeiro número nem o segundo número, e não ambos a primeira operação lógica e o terceiro número.
Este programa é muito muito legal.`

// Portuguese parses into exactly the tables of its English twin, layout and
// scratch slots included.
func TestPortuguese(t *testing.T) {
	want, err := parse(english)
	if err != nil {
		t.Fatalf("english: %v", err)
	}
	for _, tt := range []struct{ name, src string }{
		{"as written", portuguese},
		{"without accents", strings.NewReplacer("é", "e", "ê", "e", "à", "a", "ã", "a", "õ", "o", "ç", "c", "ú", "u", "ó", "o", "í", "i").Replace(portuguese)},
		{"upper case", strings.ToUpper(portuguese)},
		{"lists with commas", strings.NewReplacer("vinte e três e", "vinte e três, e", "rótulo e às", "rótulo, e às").Replace(portuguese)},
		{"lists without commas", strings.NewReplacer("segundo número, e da oitava", "segundo número e da oitava", "número e a condição", "número, e a condição").Replace(portuguese)},
		{"é, do que, um, milhões, por o", strings.NewReplacer("seja igual", "é igual", "menor que", "menor do que", "for verdadeira", "é verdadeira",
			"como caractere", "como um caractere", "um milhão", "um milhões", "condição de que o", "condição que o", "pelo segundo", "por o segundo").Replace(portuguese)},
		{"either gender", strings.NewReplacer("o primeiro número ao segundo", "o primeira número ao segunda", "da oitava célula", "do oitavo célula").Replace(portuguese)},
		{"short marker", strings.Replace(portuguese, "Este programa é muito muito legal.", "Muito muito legal.", 1)},
		{"French marker", strings.Replace(portuguese, "Este programa é muito muito legal.", "Très très chouette.", 1)},
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
func TestPortugueseNone(t *testing.T) {
	src := `Este programa usa o número sete.
Este programa não vai a lugar nenhum.
Este programa não pode escrever.
Este programa não pode ler.
Este programa usa a soma do primeiro número e do primeiro número.
Este programa não usa nenhuma condição.
Este programa usa um rótulo.
Este programa não usa nenhuma diferença ordenada.
Este programa atribui a primeira soma ao primeiro número.
Este programa não usa nenhum produto.
Este programa implementa a primeira atribuição.
Este programa não usa nenhuma razão.
Este programa é ilógico.
Muito muito legal.`
	p, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.Data[0] != 7 || p.LabelsCount != 1 || p.Tables[Sums].Count != 1 || p.Tables[Assigns].Count != 1 || p.Tables[Statements].Count != 1 {
		t.Errorf("%+v", p)
	}
	for _, tt := range []struct{ from, to string }{
		{"usa o número sete", "não usa nenhum número"},
		{"usa a soma do primeiro número e do primeiro número", "não usa nenhuma soma"},
		{"usa um rótulo", "não usa nenhum rótulo"},
		{"atribui a primeira soma ao primeiro número", "não faz nenhuma atribuição"},
		{"atribui a primeira soma ao primeiro número", "não usa nenhuma atribuição"},
		{"é ilógico", "não usa nenhuma operação lógica"},
		{"não usa nenhuma condição", "usa a condição de que o primeiro número seja menor que o primeiro número"},
		{"não vai a lugar nenhum", "sempre vai para o primeiro rótulo"},
		{"não usa nenhuma diferença ordenada", "usa as diferenças ordenadas entre o primeiro número e o primeiro número e entre a primeira soma e a primeira soma"},
		{"não usa nenhum produto", "usa o produto do primeiro número e do primeiro número"},
		{"não usa nenhuma razão", "usa as razões entre o primeiro número e o primeiro número, entre o primeiro número e o primeiro número e entre o primeiro número e o primeiro número"},
		{"é ilógico", "usa a operação lógica não ambos o primeiro número e o primeiro número"},
		{"não pode escrever", "escreve o primeiro número como ordinal brasileiro"},
		{"não pode ler", "lê o primeiro número como caractere"},
		{"usa um rótulo", "usa vinte e um rótulos"},
	} {
		if _, err := parse(strings.Replace(src, tt.from, tt.to, 1)); err != nil {
			t.Errorf("%s: %v", tt.to, err)
		}
	}
}

// Numbers are read greedily: "vinte e um" is 21, and a list says 20 and 1
// with a comma before its "e".
func TestPortugueseNumberLists(t *testing.T) {
	src := strings.Replace(portuguese, "Este programa usa os números zero, um, vinte e três e um milhão.", "Este programa usa os números %s.", 1)
	src = strings.Replace(src, "o décimo primeiro número", "o primeiro número", 1)
	src = strings.Replace(src, "o quarto número", "o primeiro número", 1)
	for _, tt := range []struct {
		list string
		want []int32
	}{
		{"vinte e um", []int32{21}},
		{"vinte, e um", []int32{20, 1}},
		{"dois e três", []int32{2, 3}},
		{"cem e um", []int32{100, 1}},
		{"cento e um", []int32{101}},
		{"mil e cem", []int32{1100}},
		{"mil, e cem", []int32{1000, 100}},
		{"um, vinte e um e dois", []int32{1, 21, 2}},
		{"duzentos e trinta e quatro mil quinhentos e sessenta e sete e oito", []int32{234567, 8}},
	} {
		p, err := parse(strings.Replace(src, "%s", tt.list, 1))
		if err != nil {
			t.Errorf("%s: %v", tt.list, err)
			continue
		}
		if !reflect.DeepEqual(p.Data, tt.want) {
			t.Errorf("%s: %v, want %v", tt.list, p.Data, tt.want)
		}
	}
}

// Portuguese is Very Very Sorted!, mixes with the other languages, and
// broken Portuguese fails.
func TestPortugueseDialect(t *testing.T) {
	for _, end := range []string{"Cool.", "This code is very cool.", "Este programa é muito legal."} {
		if _, err := parse(strings.Replace(portuguese, "Este programa é muito muito legal.", end, 1)); err == nil || err.Error() != "ERROR, missing or invalid number declaration" {
			t.Errorf("%s: %v", end, err)
		}
	}
	lines := strings.Split(portuguese, "\n")
	for _, other := range []string{italian, french} {
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
		{"as a brazilian cardinal", FormatBrazilianCardinal}, {"as a brazilian ordinal", FormatBrazilianOrdinal},
		{"als ein brasilianischer Kardinal", FormatBrazilianCardinal}, {"als eine brasilianische Ordinalzahl", FormatBrazilianOrdinal},
		{"come cardinale brasiliano", FormatBrazilianCardinal}, {"come ordinale brasiliano", FormatBrazilianOrdinal},
		{"comme cardinal brésilien", FormatBrazilianCardinal}, {"comme ordinal brésilien", FormatBrazilianOrdinal},
		{"como cardinal brasileiro", FormatBrazilianCardinal}, {"como ordinal brasileiro", FormatBrazilianOrdinal},
		{"como cardinal inglês", FormatEnglishCardinal}, {"como ordinal alemão", FormatGermanOrdinal},
		{"como cardinal valdense", FormatVaudoisCardinal}, {"como um caractere", FormatCharacter},
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
	for _, tt := range []struct{ from, to string }{
		{"vinte e três e um milhão", "vinte e três e vinte e três"},
		{"para o primeiro rótulo e às", "para a primeira condição e às"},
		{"como cardinal italiano", "como cardinal"},
		{"o décimo primeiro número", "o décimo primeiro"},
		{"ao quarto número", "à quarta soma"},
		{"seja igual ao", "seja maior que o"},
		{"usa dois rótulos", "usa dois rótulo"},
		{"um milhão", "um bilhão"},                 // declared numbers stop below 1000000000
		{"vinte e três", "vinte e dezanove"},       // Portugal's 19 is not Brazil's
		{"às vezes vai", "às vezes vai a"},         // "para" it is
		{"não ambos a primeira", "não a primeira"}, // NAND needs "ambos"
	} {
		if !strings.Contains(portuguese, tt.from) {
			t.Fatalf("%q is not in the program", tt.from)
		}
		if _, err := parse(strings.Replace(portuguese, tt.from, tt.to, 1)); err == nil {
			t.Errorf("%s parses", tt.to)
		}
	}
}
