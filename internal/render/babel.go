package render

import (
	"math/rand/v2"
	"strings"
	"unicode"

	"github.com/gersonkurz/sorted/internal/syntax"
)

// Babel mode (#34, Gerson's idea): a program written in several languages
// at once, each sentence in one of them, which Sorted! has always allowed
// (the 2000 parser mixes English and German per sentence). It is the only
// machine translator with 100% accuracy, as long as you only ever say one of
// fourteen things: the text still parses back into the same tables. The
// program is in the newest dialect any of the languages needs (Verys), and
// its numbers are written in their sentence's language.
type Babel struct {
	Langs []Lang
	Mix   Mix
	Seed  uint64 // for Random: the same seed, the same mix
}

// Mix is how Babel picks the language of each sentence.
type Mix int

const (
	Random    Mix = iota // at random, from Seed (the default: non-judgmental)
	Alternate            // in turn, in the order given
	Singable             // the one with the fewest syllables, the first given on a tie
)

// One is a single language: no mixing at all.
func One(l Lang) Babel { return Babel{Langs: []Lang{l}, Mix: Alternate} }

// Verys is the dialect the program needs: the newest any language needs.
func (b Babel) Verys() int {
	n := 0
	for _, l := range b.Langs {
		n = max(n, l.Verys())
	}
	return n
}

// rng is the random source of a Random mix, from its seed.
func (b Babel) rng() *rand.Rand { return rand.New(rand.NewPCG(b.Seed, b.Seed)) }

// candidates are the languages sentence i may be written in: the one the
// mix picks, or for Singable all of them, of which write keeps the one
// with the fewest syllables as it will stand (with its French filler), the
// first given on a tie.
func (b Babel) candidates(i int, rng *rand.Rand) []Lang {
	switch b.Mix {
	case Alternate:
		return b.Langs[i%len(b.Langs) : i%len(b.Langs)+1]
	case Random:
		j := rng.IntN(len(b.Langs))
		return b.Langs[j : j+1]
	}
	return b.Langs
}

// Syllables counts the syllables of a text, roughly but alike for every
// language (Gerson's choice on #34): a Han character is one, and otherwise
// a run of vowels is one ("cool" one, "chouette" two, "ichi" two, "shùzì"
// two).
func Syllables(text string) int {
	n, vowel := 0, false
	for _, r := range strings.ToLower(syntax.Unaccent(text)) {
		switch {
		case unicode.Is(unicode.Han, r):
			n++
			vowel = false
		case strings.ContainsRune("aeiouy", r):
			if !vowel {
				n++
			}
			vowel = true
		default:
			vowel = false
		}
	}
	return n
}
