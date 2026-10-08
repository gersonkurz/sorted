package syntax

import (
	"reflect"
	"strings"
	"testing"
)

// italian uses every Italian sentence form, and english says the same in
// English; both are Very Very Sorted! (#30).
const (
	italian = `Questo programma usa i numeri zero, uno, ventitré e un milione.
Questo programma va sempre alla prima etichetta e va talvolta alla seconda etichetta se la prima condizione è vera.
Questo programma scrive il terzo numero come cardinale italiano.
Questo programma legge la cella indicizzata dal secondo numero come carattere.
Questo programma usa le somme del primo numero e del secondo numero, e dell'ottava cella e della prima somma.
Questo programma usa la condizione che il primo numero sia uguale al secondo numero e la condizione che la prima somma sia minore della cella indicizzata dal primo numero.
Questo programma usa due etichette.
Questo programma usa la differenza ordinata tra il primo numero e l'undicesimo numero.
Questo programma assegna il primo numero al secondo numero, la prima somma alla cella indicizzata dal primo numero e il primo numero al quarto numero.
Questo programma usa i prodotti del primo numero e del secondo numero, e del primo prodotto e del primo rapporto.
Questo programma implementa il primo assegnamento, la prima etichetta, il primo ingresso, la prima uscita, il secondo salto e la seconda etichetta.
Questo programma usa il rapporto tra il primo numero e il secondo numero.
Questo programma usa le operazioni logiche né il primo numero né il secondo numero, e non entrambi la prima operazione logica e il terzo numero.
Questo programma è molto molto figo.`

	english = `This code uses the numbers zero, one, twentythree, and onemillion.
This code always goes to the first label, and sometimes goes to the second label if the first condition is true.
This code writes the third number as an italian cardinal.
This code reads the cell indexed by the second number as a character.
This code uses the sums of the first number and the second number, and of the eight cell and the first sum.
This code uses the condition that the first number is equal to the second number, and the condition that the first sum is less than the cell indexed by the first number.
This code uses two labels.
This code uses the ordered difference between the first number and the eleventh number.
This code assigns the first number to the second number, the first sum to the cell indexed by the first number, and the first number to the fourth number.
This code uses the products of the first number and the second number, and of the first product and the first ratio.
This code implements the first assignment, the first label, the first input, the first output, the second jump, and the second label.
This code uses the ratio of the first number to the second number.
This code uses the logical operations of not the first number and not the second number, and of not both the first logical operation and the third number.
This code is very very cool.`
)

// Italian parses into exactly the tables its English twin does, table
// layout and scratch slots included, so a program translates either way.
func TestItalian(t *testing.T) {
	want, err := parse(english)
	if err != nil {
		t.Fatalf("english: %v", err)
	}
	if want.Verys != 2 || want.Entries(Writes)[0].Flags != FormatItalianCardinal || want.Entries(Nands)[1].Flags != LogicalNand {
		t.Fatalf("english: verys %d, writes %v, nands %v", want.Verys, want.Entries(Writes), want.Entries(Nands))
	}
	for _, tt := range []struct{ name, src string }{
		{"as written", italian},
		{"without accents", strings.NewReplacer("é", "e", "è", "e", "à", "a").Replace(italian)},
		{"upper case", strings.ToUpper(italian)},
		{"typographic apostrophes", strings.ReplaceAll(italian, "'", "’")},
		{"lists with commas", strings.NewReplacer("uno, ventitré e", "uno, ventitré, e", "etichetta e va", "etichetta, e va").Replace(italian)},
		{"lists without commas", strings.NewReplacer("numero, e dell'", "numero e dell'", "numero e la condizione", "numero, e la condizione").Replace(italian)},
		{"è, fra, ed, un", strings.NewReplacer("sia uguale", "è uguale", "tra il primo numero e l'", "fra il primo numero ed l'", "come carattere", "come un carattere").Replace(italian)},
		{"elided and spaced numbers", strings.NewReplacer("ventitré", "ventitre", "un milione", "unmilione").Replace(italian)},
		{"short marker", strings.Replace(italian, "Questo programma è molto molto figo.", "Molto molto figo.", 1)},
		{"English marker", strings.Replace(italian, "Questo programma è molto molto figo.", "Very very cool.", 1)},
		{"German marker", strings.Replace(italian, "Questo programma è molto molto figo.", "Dieses Programm ist ganz ganz hervorragend.", 1)},
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
func TestItalianNone(t *testing.T) {
	src := `Questo programma usa il numero sette.
Questo programma non va mai da nessuna parte.
Questo programma non può scrivere.
Questo programma non può leggere.
Questo programma usa la somma del primo numero e del primo numero.
Questo programma non usa condizioni.
Questo programma usa un'etichetta.
Questo programma non usa differenze ordinate.
Questo programma assegna la prima somma al primo numero.
Questo programma non usa prodotti.
Questo programma implementa il primo assegnamento.
Questo programma non usa rapporti.
Questo programma è illogico.
Molto molto figo.`
	p, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.Data[0] != 7 || p.LabelsCount != 1 || p.Tables[Sums].Count != 1 || p.Tables[Assigns].Count != 1 || p.Tables[Statements].Count != 1 {
		t.Errorf("%+v", p)
	}
	for _, tt := range []struct{ from, to string }{
		{"usa il numero sette", "non usa numeri"},
		{"non può scrivere", "non produce uscite"},
		{"non può leggere", "non riceve ingressi"},
		{"usa la somma del primo numero e del primo numero", "non usa somme"},
		{"usa un'etichetta", "non usa etichette"},
		{"assegna la prima somma al primo numero", "non usa assegnamenti"},
		{"è illogico", "non usa operazioni logiche"},
		{"non usa condizioni", "usa la condizione che il primo numero sia minore del primo numero"},
		{"non va mai da nessuna parte", "va sempre alla prima etichetta"},
		{"non usa differenze ordinate", "usa le differenze ordinate tra il primo numero e il primo numero e tra la prima somma e la prima somma"},
		{"non usa prodotti", "usa il prodotto del primo numero e del primo numero"},
		{"non usa rapporti", "usa i rapporti tra il primo numero e il primo numero, tra il primo numero e il primo numero, e fra il primo numero e il primo numero"},
		{"è illogico", "usa l'operazione logica non entrambi il primo numero e il primo numero"},
		{"non può scrivere", "scrive il primo numero come ordinale tedesco"},
	} {
		if _, err := parse(strings.Replace(src, tt.from, tt.to, 1)); err != nil {
			t.Errorf("%s: %v", tt.to, err)
		}
	}
}

// Italian is Very Very Sorted!: without that ending, the original's error
// stands; and English and German sentences mix with Italian ones, as they
// do with each other.
func TestItalianDialect(t *testing.T) {
	for _, end := range []string{"Cool.", "This code is very cool.", "Questo programma è molto figo.", "Questo programma è molto molto molto figo."} {
		if _, err := parse(strings.Replace(italian, "Questo programma è molto molto figo.", end, 1)); err == nil || err.Error() != "ERROR, missing or invalid number declaration" {
			t.Errorf("%s: %v", end, err)
		}
	}
	lines := strings.Split(italian, "\n")
	en := strings.Split(english, "\n")
	for i := range lines {
		mixed := append(append(append([]string{}, lines[:i]...), en[i]), lines[i+1:]...)
		if _, err := parse(strings.Join(mixed, "\n")); err != nil {
			t.Errorf("English sentence %d: %v", i+1, err)
		}
	}
	// The Italian output formats in German, and only in Very Very Sorted!.
	for _, f := range []struct {
		de     string
		format int32
	}{{"als ein italienischer Kardinal", FormatItalianCardinal}, {"als eine italienische Ordinalzahl", FormatItalianOrdinal}} {
		src := strings.Replace(minimal, "This code writes the first number as a character.", "Dieses Programm schreibt die erste Zahl "+f.de+".", 1)
		if _, err := parse(strings.Replace(src, "Cool.", "This code is very cool.", 1)); err == nil {
			t.Errorf("%s in Very Sorted!", f.de)
		}
		p, err := parse(strings.Replace(src, "Cool.", "Ganz ganz hervorragend.", 1))
		if err != nil || p.Entries(Writes)[0].Flags != f.format {
			t.Errorf("%s: %v", f.de, err)
		}
	}
	// Broken Italian fails.
	for _, tt := range []struct{ from, to string }{
		{"ventitré e un milione", "ventitré e ventitré"},                             // a number twice
		{"alla prima etichetta e va", "alla prima condizione e va"},                  // a jump to a condition
		{"il terzo numero come", "il terzo numero come cardinale"},                   // no language
		{"l'undicesimo numero", "l'undicesimo"},                                      // no noun
		{"al quarto numero", "alla quarta somma"},                                    // a store into a sum
		{"sia uguale al", "sia maggiore del"},                                        // no such comparison
		{"Questo programma usa due etichette", "Questo programma usa due etichetta"}, // two of one
		{"la cella indicizzata dal secondo", "la cella indicizzata dalla cella indicizzata dal secondo"},
	} {
		if !strings.Contains(italian, tt.from) {
			t.Fatalf("%q is not in the program", tt.from)
		}
		if _, err := parse(strings.Replace(italian, tt.from, tt.to, 1)); err == nil {
			t.Errorf("%s parses", tt.to)
		}
	}
}

// Declared numbers and label counts stop below 1000000000, as in English and
// German.
func TestItalianDeclarationLimit(t *testing.T) {
	const most = "novecentonovantanovemilioninovecentonovantanovemilanovecentonovantanove"
	p, err := parse(strings.Replace(italian, "un milione", most, 1))
	if err != nil || p.Data[3] != 999999999 {
		t.Errorf("999999999: %v", err)
	}
	for _, too := range []string{"unmiliardo", "un miliardo", "duemiliardi"} {
		if _, err := parse(strings.Replace(italian, "un milione", too, 1)); err == nil {
			t.Errorf("%s declared", too)
		}
	}
	if _, err := parse(strings.Replace(italian, "usa due etichette", "usa "+most+" etichette", 1)); err != nil {
		t.Errorf("999999999 labels: %v", err)
	}
	if _, err := parse(strings.Replace(italian, "usa due etichette", "usa unmiliardo etichette", 1)); err == nil {
		t.Error("1000000000 labels declared")
	}
}

func TestFilterVeryVery(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"È così: può", "e cosi  puo"},
		{"Fünf, ZWÖLF.", "fuenf, zwoelf."},
		{"dell'ottavo", "dell ottavo"},
		{"ドイツ語", "ドイツ語"},
	} {
		if got := FilterVeryVery([]byte(tt.in)); got != tt.want {
			t.Errorf("FilterVeryVery(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
