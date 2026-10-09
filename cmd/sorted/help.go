package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/gersonkurz/sorted/internal/render"
)

// helps is the help text in every language Sorted! speaks (#43, wording on
// the issue): the usage, the flags, and the line before the table of names.
// The flags are syntax and stay as they are; the placeholders are
// translated, and Vaudois French keeps its fillers. Which language says it
// is as non-judgmental as the language of --from-c: the one named, or one
// at random (usage).
var helps = map[render.Lang]string{
	render.English: `usage: sorted [--dump FILE] [--to-c FILE] [--lang NAME | --NAME] [--version] PROGRAM.s
       sorted --from-c PROGRAM.c [--lang NAME | --NAME] [--dump FILE] [--to-c FILE]
  --dump FILE     write the parsed tables to FILE (legacy /D)
  --from-c FILE   compile the C program FILE into Sorted! and print it
  --lang NAME     print the program in the language NAME (see below) instead of running it
  --mix RULE      how several languages mix: alternate, random[:SEED] (the default) or singable
  --to-c FILE     write a C program that behaves like this one to FILE
  --version       print the version and exit
NAME is any language Sorted! speaks, named in any language it speaks:
`,
	render.German: `Aufruf: sorted [--dump DATEI] [--to-c DATEI] [--lang NAME | --NAME] [--version] PROGRAMM.s
        sorted --from-c PROGRAMM.c [--lang NAME | --NAME] [--dump DATEI] [--to-c DATEI]
  --dump DATEI    schreibt die geparsten Tabellen in DATEI (das alte /D)
  --from-c DATEI  übersetzt das C-Programm DATEI in Sorted! und gibt es aus
  --lang NAME     gibt das Programm in der Sprache NAME (siehe unten) aus, statt es auszuführen
  --mix REGEL     wie sich mehrere Sprachen mischen: alternate, random[:SAAT] (der Standard) oder singable
  --to-c DATEI    schreibt ein C-Programm, das sich wie dieses verhält, in DATEI
  --version       gibt die Version aus und beendet sich
NAME ist jede Sprache, die Sorted! spricht, benannt in jeder Sprache, die es spricht:
`,
	render.Italian: `uso: sorted [--dump FILE] [--to-c FILE] [--lang NOME | --NOME] [--version] PROGRAMMA.s
     sorted --from-c PROGRAMMA.c [--lang NOME | --NOME] [--dump FILE] [--to-c FILE]
  --dump FILE     scrive le tabelle analizzate in FILE (il vecchio /D)
  --from-c FILE   compila il programma C FILE in Sorted! e lo stampa
  --lang NOME     stampa il programma nella lingua NOME (vedi sotto) invece di eseguirlo
  --mix REGOLA    come si mescolano più lingue: alternate, random[:SEME] (predefinito) o singable
  --to-c FILE     scrive in FILE un programma C che si comporta come questo
  --version       stampa la versione ed esce
NOME è qualsiasi lingua che Sorted! parla, chiamata in qualsiasi lingua che parla:
`,
	render.French: `usage : sorted [--dump FICHIER] [--to-c FICHIER] [--lang NOM | --NOM] [--version] PROGRAMME.s
        sorted --from-c PROGRAMME.c [--lang NOM | --NOM] [--dump FICHIER] [--to-c FICHIER]
  --dump FICHIER    écrit les tables analysées dans FICHIER (l'ancien /D)
  --from-c FICHIER  compile le programme C FICHIER en Sorted! et l'affiche
  --lang NOM        affiche le programme dans la langue NOM (voir plus bas) au lieu de l'exécuter
  --mix RÈGLE       comment plusieurs langues se mélangent : alternate, random[:GRAINE] (par défaut) ou singable, quoi
  --to-c FICHIER    écrit dans FICHIER un programme C qui se comporte comme celui-ci
  --version         affiche la version et s'en va, voilà
NOM est n'importe quelle langue que Sorted! parle, nommée dans n'importe laquelle, hein :
`,
	render.Portuguese: `uso: sorted [--dump ARQUIVO] [--to-c ARQUIVO] [--lang NOME | --NOME] [--version] PROGRAMA.s
     sorted --from-c PROGRAMA.c [--lang NOME | --NOME] [--dump ARQUIVO] [--to-c ARQUIVO]
  --dump ARQUIVO    escreve as tabelas analisadas em ARQUIVO (o antigo /D)
  --from-c ARQUIVO  compila o programa C ARQUIVO em Sorted! e o imprime
  --lang NOME       imprime o programa no idioma NOME (veja abaixo) em vez de executá-lo
  --mix REGRA       como vários idiomas se misturam: alternate, random[:SEMENTE] (o padrão) ou singable
  --to-c ARQUIVO    escreve em ARQUIVO um programa C que se comporta como este
  --version         imprime a versão e sai
NOME é qualquer idioma que o Sorted! fala, chamado em qualquer idioma que ele fala:
`,
	render.Japanese: `tsukaikata: sorted [--dump FAIRU] [--to-c FAIRU] [--lang NAMAE | --NAMAE] [--version] PUROGURAMU.s
            sorted --from-c PUROGURAMU.c [--lang NAMAE | --NAMAE] [--dump FAIRU] [--to-c FAIRU]
  --dump FAIRU    kaiseki shita tēburu o FAIRU ni kakimasu (mukashi no /D)
  --from-c FAIRU  C no puroguramu FAIRU o Sorted! ni konpairu shite hyōji shimasu
  --lang NAMAE    puroguramu o jikkō suru kawari ni, gengo NAMAE de hyōji shimasu (shita o mite kudasai)
  --mix RŪRU      fukusū no gengo no mazekata: alternate, random[:SHĪDO] (kitei) mata wa singable
  --to-c FAIRU    kore to onaji yō ni ugoku C no puroguramu o FAIRU ni kakimasu
  --version       bājon o hyōji shite shūryō shimasu
NAMAE wa Sorted! ga hanasu gengo desu. Sorted! ga hanasu dono gengo de yonde mo ii desu:
`,
	render.Mandarin: `用法：sorted [--dump 文件] [--to-c 文件] [--lang 名称 | --名称] [--version] 程序.s
      sorted --from-c 程序.c [--lang 名称 | --名称] [--dump 文件] [--to-c 文件]
  --dump 文件     把解析出的表写入文件（旧版的 /D）
  --from-c 文件   把 C 程序文件编译成 Sorted! 并输出
  --lang 名称     不运行程序，而是用名称所指的语言（见下文）输出程序
  --mix 规则      多种语言的混合方式：alternate、random[:种子]（默认）或 singable
  --to-c 文件     把一个行为与此程序相同的 C 程序写入文件
  --version       输出版本并退出
名称是 Sorted! 会说的任何语言，可以用它会说的任何语言来称呼：
`,
	render.Pinyin: `yòngfǎ: sorted [--dump WÉNJIÀN] [--to-c WÉNJIÀN] [--lang MÍNGCHĒNG | --MÍNGCHĒNG] [--version] CHÉNGXÙ.s
        sorted --from-c CHÉNGXÙ.c [--lang MÍNGCHĒNG | --MÍNGCHĒNG] [--dump WÉNJIÀN] [--to-c WÉNJIÀN]
  --dump WÉNJIÀN    bǎ jiěxī chū de biǎo xiěrù wénjiàn (jiùbǎn de /D)
  --from-c WÉNJIÀN  bǎ C chéngxù wénjiàn biānyì chéng Sorted! bìng shūchū
  --lang MÍNGCHĒNG  bù yùnxíng chéngxù, ér shì yòng míngchēng suǒ zhǐ de yǔyán (jiàn xiàwén) shūchū chéngxù
  --mix GUĪZÉ       duō zhǒng yǔyán de hùnhé fāngshì: alternate, random[:ZHǑNGZI] (mòrèn) huò singable
  --to-c WÉNJIÀN    bǎ yī gè xíngwéi yǔ cǐ chéngxù xiāngtóng de C chéngxù xiěrù wénjiàn
  --version         shūchū bǎnběn bìng tuìchū
míngchēng shì Sorted! huì shuō de rènhé yǔyán, kěyǐ yòng tā huì shuō de rènhé yǔyán lái chēnghu:
`,
}

// usage writes the help in the language named on the command line, or in
// one picked at random when none is, or when the names disagree (being
// non-judgmental, it does not side with either).
func usage(w io.Writer, named []string) {
	langs := map[render.Lang]bool{}
	for _, name := range named {
		for _, name := range strings.Split(name, ",") { // --lang zh,fr
			if l, ok := findLanguage(name); ok && l.spoken {
				langs[l.lang] = true
			}
		}
	}
	lang := pickLang()
	for l := range langs {
		if len(langs) == 1 {
			lang = l
		}
	}
	fmt.Fprint(w, helps[lang]+languageTable())
}
