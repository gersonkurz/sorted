// Command sorted runs programs written in Sorted!, the esoteric language from
// 2000.
//
//	sorted [--dump FILE] [--to-c FILE] [--lang NAME | --NAME] [--version] PROGRAM.s
//	sorted --from-c PROGRAM.c [--lang NAME | --NAME] [--dump FILE] [--to-c FILE]
//
// Before the program runs, --dump writes the parsed tables, as the
// original's /D does, and --to-c a C program that behaves exactly like it
// (unlike the original's /C, which is not ported), so C -> Sorted! -> C
// (--from-c with --to-c) turns a C program into an equivalent, thoroughly
// obfuscated one. --lang NAME, or just --NAME, prints the program in that
// language instead of running it, and NAME may name it in any language
// Sorted! speaks (--english, --deutsch, --italiano, --vaudois, --brasileiro, --nihongo, --中文, --pinyin, --lang anglais, --英語; see
// langs.go). --from-c compiles a C program into Sorted! and prints it;
// Sorted! does not prefer any language, so unless one is asked for, each
// run picks one at random.
//
// The flags are modern; what a program prints, and the diagnostics of the
// original (on stdout, byte for byte), are those of the Win32 Sorted.exe.
// Unlike the original, which always exits with 0, sorted exits with 1 when the
// program cannot be read or parsed, a requested file cannot be written (the
// original silently skips it), or the program fails at run time, and with 2
// on a usage error. Run-time errors have no counterpart in the original (it
// crashes), so they go to stderr.
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gersonkurz/sorted/internal/cc"
	"github.com/gersonkurz/sorted/internal/compile"
	"github.com/gersonkurz/sorted/internal/emit"
	"github.com/gersonkurz/sorted/internal/interp"
	"github.com/gersonkurz/sorted/internal/render"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// version is set at build time via -ldflags "-X main.version=...": what git
// describe says, or dev.
var version = "dev"

// versionLine is what --version prints (#39): the marker sentence of the
// newest dialect this binary reads, the dialect being the version, and, for
// a build that is not exactly on that dialect's release tag, the build in
// parentheses: "This code is very cool. (a105f41-dirty)".
func versionLine(build string) string {
	line := syntax.Marker(syntax.Newest)
	if build != syntax.Tag(syntax.Newest) {
		line += " (" + build + ")"
	}
	return line
}

// pickLang chooses the language of a compiled program that nobody chose a
// language for (defaultPick: any Sorted! speaks, at random). Tests replace
// it.
var pickLang = defaultPick

func defaultPick() render.Lang {
	var spoken []render.Lang
	for _, l := range languages {
		if l.spoken {
			spoken = append(spoken, l.lang)
		}
	}
	return spoken[rand.IntN(len(spoken))]
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command line and returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sorted", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print the version and exit")
	dumpFile := fs.String("dump", "", "write the parsed tables to `FILE` (legacy /D)")
	cFile := fs.String("to-c", "", "write a C program that behaves like this one to `FILE`")
	fs.String("lang", "", "print the program in the language `NAME` (see below) instead of running it") // its values, in order among the --NAME flags: languageFlags
	fromC := fs.String("from-c", "", "compile the C program `FILE` into Sorted! and print it")
	mix := fs.String("mix", "random", "how several languages mix: alternate, random[:SEED] or singable")
	var named []string
	fs.Usage = func() { usage(stderr, named) }
	args, named = languageFlags(fs, args)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, versionLine(version))
		return 0
	}
	positional := 1 // the Sorted! program, unless compiling from C
	if *fromC != "" {
		positional = 0
	}
	var chosen []render.Lang // the languages asked for, each once, in order: several make Babel (#34)
	for _, name := range slices.Concat(splitNames(named)...) {
		l, ok := findLanguage(name)
		if !ok {
			fs.Usage()
			return 2
		}
		if !l.spoken {
			fmt.Fprintf(stderr, "sorted: Sorted! does not speak %s yet\n", name)
			return 2
		}
		if !slices.Contains(chosen, l.lang) {
			chosen = append(chosen, l.lang)
		}
	}
	babel, ok := parseMix(*mix, chosen)
	if !ok || fs.NArg() != positional {
		fs.Usage()
		return 2
	}
	if *fromC != "" {
		if len(chosen) == 0 {
			babel = render.One(pickLang())
		}
		return translate(*fromC, babel, outputs{*cFile, *dumpFile}, stdout, stderr)
	}
	name := fs.Arg(0)

	raw, err := os.ReadFile(name)
	if err != nil {
		// The original's preprocessor reports the file without a newline,
		// so both messages end up on one line.
		fmt.Fprintf(stdout, "*** ERROR, unable to open file %s for reading", name)
		fmt.Fprintf(stdout, "%s is not intelligible.\n", name)
		return 1
	}
	p, err := syntax.Parse(raw)
	if err != nil {
		fmt.Fprintf(stdout, "%v\n", err)
		fmt.Fprintf(stdout, "%s is not intelligible.\n", name)
		return 1
	}
	// Like the original: the C translation first, then the dump, then run.
	code := writeFiles(p, outputs{*cFile, *dumpFile}, stderr)
	if len(chosen) > 0 {
		text, err := render.RenderBabel(p, babel)
		if err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			return 1
		}
		if _, err := fmt.Fprint(stdout, text); err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			return 1
		}
		return code
	}
	if err := interp.Run(p, stdin, stdout, 0); err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	return code
}

// splitNames splits each name at its commas (--lang zh,fr).
func splitNames(names []string) [][]string {
	var parts [][]string
	for _, n := range names {
		parts = append(parts, strings.Split(n, ","))
	}
	return parts
}

// parseMix reads --mix for the languages chosen: "alternate", "singable",
// "random" or "random:SEED" (Babel mode, #34). Without a seed, the mix is
// different every time. A single language is just that language.
func parseMix(mix string, langs []render.Lang) (render.Babel, bool) {
	b := render.Babel{Langs: langs, Seed: rand.Uint64()}
	switch kind, seed, seeded := strings.Cut(mix, ":"); {
	case kind == "alternate" && !seeded:
		b.Mix = render.Alternate
	case kind == "singable" && !seeded:
		b.Mix = render.Singable
	case kind == "random" && !seeded:
		b.Mix = render.Random
	case kind == "random":
		n, err := strconv.ParseUint(seed, 10, 64)
		if err != nil {
			return b, false
		}
		b.Mix, b.Seed = render.Random, n
	default:
		return b, false
	}
	if len(langs) == 1 {
		return render.One(langs[0]), true
	}
	return b, true
}

// outputs are the files a run writes besides the program's output.
type outputs struct {
	c, dump string // --to-c, --dump
}

// writeFiles writes the requested outputs, the C translation before the
// dump as the original does, and returns 1 if one cannot be written.
func writeFiles(p *syntax.Program, files outputs, stderr io.Writer) int {
	code := 0
	for _, out := range []struct {
		file   string
		render func(*syntax.Program) string
	}{{files.c, emit.Exact}, {files.dump, emit.Dump}} {
		if out.file == "" {
			continue
		}
		if err := os.WriteFile(out.file, []byte(out.render(p)), 0o644); err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			code = 1
		}
	}
	return code
}

// translate compiles a C program into Sorted! and prints it. --dump and
// --to-c describe the compiled program as any Sorted! interpreter sees it.
func translate(name string, babel render.Babel, files outputs, stdout, stderr io.Writer) int {
	src, err := os.ReadFile(name)
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	prog, err := cc.Parse(string(src))
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %s:%v\n", name, err)
		return 1
	}
	compiled, err := compile.Compile(prog, babel.Verys())
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %s:%v\n", name, err)
		return 1
	}
	text, p, err := render.ComposeBabel(compiled, babel)
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %s: %v\n", name, err)
		return 1
	}
	code := writeFiles(p, files, stderr)
	if _, err := fmt.Fprint(stdout, text); err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	return code
}
