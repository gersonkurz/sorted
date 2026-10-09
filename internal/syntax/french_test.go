package syntax

import (
	"reflect"
	"strings"
	"testing"
)

// french says what english (italian_test.go) says, in Vaudois French (#29).
const french = `Ce programme utilise les nombres zéro, un, vingt-trois et un-million.
Ce programme va toujours à la première étiquette et va parfois à la deuxième étiquette si la première condition est vraie.
Ce programme écrit le troisième nombre comme cardinal italien.
Ce programme lit la cellule indexée par le deuxième nombre comme caractère.
Ce programme utilise les sommes du premier nombre et du deuxième nombre, et de la huitième cellule et de la première somme.
Ce programme utilise la condition que le premier nombre soit égal au deuxième nombre et la condition que la première somme soit inférieure à la cellule indexée par le premier nombre.
Ce programme utilise deux étiquettes.
Ce programme utilise la différence ordonnée entre le premier nombre et le onzième nombre.
Ce programme affecte le premier nombre au deuxième nombre, la première somme à la cellule indexée par le premier nombre et le premier nombre au quatrième nombre.
Ce programme utilise les produits du premier nombre et du deuxième nombre, et du premier produit et du premier rapport.
Ce programme implémente la première affectation, la première étiquette, la première entrée, la première sortie, le deuxième saut et la deuxième étiquette.
Ce programme utilise le rapport du premier nombre au deuxième nombre.
Ce programme utilise les opérations logiques ni le premier nombre ni le deuxième nombre, et pas à la fois la première opération logique et le troisième nombre.
Ce programme est très très chouette.`

// French parses into exactly the tables of its English twin, layout and
// scratch slots included.
func TestFrench(t *testing.T) {
	want, err := parse(english)
	if err != nil {
		t.Fatalf("english: %v", err)
	}
	fillers := strings.NewReplacer(
		"Ce programme utilise les nombres", "Ce programme, eh, utilise les nombres",
		"Ce programme va toujours", "Hein, ce programme va toujours",
		"la deuxième étiquette.\n", "la deuxième étiquette, voilà, quoi.\n",
		"soit égal au", "soit, quoi, égal au")
	for _, tt := range []struct{ name, src string }{
		{"as written", french},
		{"without accents", strings.NewReplacer("é", "e", "è", "e", "à", "a", "ç", "c").Replace(french)},
		{"upper case", strings.ToUpper(french)},
		{"fillers", fillers.Replace(french)},
		{"lists with commas", strings.NewReplacer("un, vingt-trois et", "un, vingt-trois, et", "étiquette et va", "étiquette, et va").Replace(french)},
		{"lists without commas", strings.NewReplacer("nombre, et de la", "nombre et de la", "nombre et la condition", "nombre, et la condition").Replace(french)},
		{"est, égale, une, millions", strings.NewReplacer("soit égal", "est égale", "comme caractère", "comme un caractère", "un-million", "un-millions").Replace(french)},
		{"short marker", strings.Replace(french, "Ce programme est très très chouette.", "Très très chouette.", 1)},
		{"Italian marker", strings.Replace(french, "Ce programme est très très chouette.", "Molto molto figo.", 1)},
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
func TestFrenchNone(t *testing.T) {
	src := `Ce programme utilise le nombre sept.
Y'a pas le feu au lac.
Ce programme ne peut pas écrire.
Ce programme ne peut pas lire.
Ce programme utilise la somme du premier nombre et du premier nombre.
Ce programme n'utilise aucune condition.
Ce programme utilise une étiquette.
Ce programme n'utilise aucune différence ordonnée.
Ce programme affecte la première somme au premier nombre.
Ce programme n'utilise aucun produit.
Ce programme implémente la première affectation.
Ce programme n'utilise aucun rapport.
Ce programme est illogique.
Très très chouette.`
	p, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.Data[0] != 7 || p.LabelsCount != 1 || p.Tables[Sums].Count != 1 || p.Tables[Assigns].Count != 1 || p.Tables[Statements].Count != 1 {
		t.Errorf("%+v", p)
	}
	for _, tt := range []struct{ from, to string }{
		{"utilise le nombre sept", "n'utilise aucun nombre"},
		{"Y'a pas le feu au lac", "Ce programme ne va nulle part"},
		{"ne peut pas écrire", "ne produit aucune sortie"},
		{"ne peut pas lire", "ne reçoit aucune entrée"},
		{"utilise la somme du premier nombre et du premier nombre", "n'utilise aucune somme"},
		{"utilise une étiquette", "n'utilise aucune étiquette"},
		{"affecte la première somme au premier nombre", "n'utilise aucune affectation"},
		{"est illogique", "n'utilise aucune opération logique"},
		{"n'utilise aucune condition", "utilise la condition que le premier nombre soit inférieur au premier nombre"},
		{"Y'a pas le feu au lac", "Ce programme va toujours à la première étiquette"},
		{"n'utilise aucune différence ordonnée", "utilise les différences ordonnées entre le premier nombre et le premier nombre et entre la première somme et la première somme"},
		{"n'utilise aucun produit", "utilise le produit du premier nombre et du premier nombre"},
		{"n'utilise aucun rapport", "utilise les rapports du premier nombre au premier nombre, du premier nombre au premier nombre et du premier nombre au premier nombre"},
		{"est illogique", "utilise l'opération logique pas à la fois le premier nombre et le premier nombre"},
		{"ne peut pas écrire", "écrit le premier nombre comme ordinal vaudois"},
		{"utilise une étiquette", "utilise vingt-et-une étiquettes"},
	} {
		if _, err := parse(strings.Replace(src, tt.from, tt.to, 1)); err != nil {
			t.Errorf("%s: %v", tt.to, err)
		}
	}
}

// French is Very Very Sorted!, mixes with the other languages, and broken
// French fails.
func TestFrenchDialect(t *testing.T) {
	for _, end := range []string{"Cool.", "This code is very cool.", "Ce programme est très chouette."} {
		if _, err := parse(strings.Replace(french, "Ce programme est très très chouette.", end, 1)); err == nil || err.Error() != "ERROR, missing or invalid number declaration" {
			t.Errorf("%s: %v", end, err)
		}
	}
	lines := strings.Split(french, "\n")
	it := strings.Split(italian, "\n")
	for i := range lines {
		mixed := append(append(append([]string{}, lines[:i]...), it[i]), lines[i+1:]...)
		if _, err := parse(strings.Join(mixed, "\n")); err != nil {
			t.Errorf("Italian sentence %d: %v", i+1, err)
		}
	}
	for _, f := range []struct {
		phrase string
		format int32
	}{
		{"as a vaudois cardinal", FormatVaudoisCardinal}, {"as a vaudois ordinal", FormatVaudoisOrdinal},
		{"als ein waadtländischer Kardinal", FormatVaudoisCardinal}, {"als eine waadtländische Ordinalzahl", FormatVaudoisOrdinal},
		{"come cardinale vodese", FormatVaudoisCardinal}, {"come ordinale vodese", FormatVaudoisOrdinal},
		{"comme cardinal anglais", FormatEnglishCardinal}, {"comme ordinal allemand", FormatGermanOrdinal},
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
		{"vingt-trois et un-million", "vingt-trois et vingt-trois"},
		{"à la première étiquette et va", "à la première condition et va"},
		{"comme cardinal italien", "comme cardinal"},
		{"le onzième nombre", "le onzième"},
		{"au quatrième nombre", "à la quatrième somme"},
		{"soit égal au", "soit supérieur au"},
		{"utilise deux étiquettes", "utilise deux étiquette"},
		{"vingt-trois", "vingt trois"}, // a number is one word
		{"un-million", "un-milliard"},  // declared numbers stop below 1000000000
	} {
		if !strings.Contains(french, tt.from) {
			t.Fatalf("%q is not in the program", tt.from)
		}
		if _, err := parse(strings.Replace(french, tt.from, tt.to, 1)); err == nil {
			t.Errorf("%s parses", tt.to)
		}
	}
}

// Very Very Sorted! keeps hyphens inside words and drops the fillers with
// their commas.
func TestFillersAndHyphens(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"Ce programme, eh, utilise", "ce programme utilise"},
		{"étiquette, voilà, quoi.", "etiquette ."},
		{"Hein, ce", " ce"},
		{"vingt-et-un, deux - trois -x", "vingt-et-un, deux   trois  x"},
		{"heine eh-bien quoique", "heine eh-bien quoique"},
	} {
		if got := FilterVeryVery([]byte(tt.in)); got != tt.want {
			t.Errorf("FilterVeryVery(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := FilterVery([]byte("vingt-et-un")); got != "vingt et un" {
		t.Errorf("FilterVery keeps hyphens: %q", got)
	}
}
