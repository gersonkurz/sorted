package syntax

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// hanzi and pinyin say what english (italian_test.go) says, in Mandarin
// (#33).
const (
	hanzi = `这个程序使用数字零、一、二十三和一百万。
这个程序总是跳到第一个标签，并在第一个条件为真时跳到第二个标签。
这个程序把第三个数字作为意大利语基数写出。
这个程序把第二个数字所指的单元作为字符读入。
这个程序使用第一个数字和第二个数字的和和第八个单元和第一个和的和。
这个程序使用第一个数字等于第二个数字的条件和第一个和小于第一个数字所指的单元的条件。
这个程序使用两个标签。
这个程序使用第一个数字和第十一个数字的差。
这个程序把第一个数字赋给第二个数字，把第一个和赋给第一个数字所指的单元，把第一个数字赋给第四个数字。
这个程序使用第一个数字和第二个数字的积和第一个积和第一个比的积。
这个程序实现第一个赋值、第一个标签、第一个输入、第一个输出、第二个跳转和第二个标签。
这个程序使用第一个数字和第二个数字的比。
这个程序使用既非第一个数字也非第二个数字的逻辑运算和并非第一个逻辑运算和第三个数字都成立的逻辑运算。
这个程序非常非常酷。`

	pinyin = `Zhège chéngxù shǐyòng shùzì líng, yī, èrshísān hé yībǎiwàn.
Zhège chéngxù zǒngshì tiàodào dì-yī gè biāoqiān, bìng zài dì-yī gè tiáojiàn wéi zhēn shí tiàodào dì-èr gè biāoqiān.
Zhège chéngxù bǎ dì-sān gè shùzì zuòwéi Yìdàlìyǔ jīshù xiěchū.
Zhège chéngxù bǎ dì-èr gè shùzì suǒ zhǐ de dānyuán zuòwéi zìfú dúrù.
Zhège chéngxù shǐyòng dì-yī gè shùzì hé dì-èr gè shùzì de hé hé dì-bā gè dānyuán hé dì-yī gè hé de hé.
Zhège chéngxù shǐyòng dì-yī gè shùzì děngyú dì-èr gè shùzì de tiáojiàn hé dì-yī gè hé xiǎoyú dì-yī gè shùzì suǒ zhǐ de dānyuán de tiáojiàn.
Zhège chéngxù shǐyòng liǎng gè biāoqiān.
Zhège chéngxù shǐyòng dì-yī gè shùzì hé dì-shíyī gè shùzì de chā.
Zhège chéngxù bǎ dì-yī gè shùzì fù gěi dì-èr gè shùzì, bǎ dì-yī gè hé fù gěi dì-yī gè shùzì suǒ zhǐ de dānyuán, bǎ dì-yī gè shùzì fù gěi dì-sì gè shùzì.
Zhège chéngxù shǐyòng dì-yī gè shùzì hé dì-èr gè shùzì de jī hé dì-yī gè jī hé dì-yī gè bǐ de jī.
Zhège chéngxù shíxiàn dì-yī gè fùzhí, dì-yī gè biāoqiān, dì-yī gè shūrù, dì-yī gè shūchū, dì-èr gè tiàozhuǎn hé dì-èr gè biāoqiān.
Zhège chéngxù shǐyòng dì-yī gè shùzì hé dì-èr gè shùzì de bǐ.
Zhège chéngxù shǐyòng jì fēi dì-yī gè shùzì yě fēi dì-èr gè shùzì de luójí yùnsuàn hé bìngfēi dì-yī gè luójí yùnsuàn hé dì-sān gè shùzì dōu chénglì de luójí yùnsuàn.
Zhège chéngxù fēicháng fēicháng kù.`
)

// Mandarin parses into exactly the tables of its English twin, layout and
// scratch slots included, in hanzi and in pinyin.
func TestMandarin(t *testing.T) {
	want, err := parse(english)
	if err != nil {
		t.Fatalf("english: %v", err)
	}
	traditional := strings.NewReplacer("这个程序", "這個程式", "数字", "數字", "标签", "標籤", "条件", "條件", "两", "兩", "赋给", "賦給",
		"读入", "讀入", "写出", "寫出", "意大利语", "意大利語", "为真时", "為真時", "输入", "輸入", "输出", "輸出", "单元", "單元",
		"积", "積", "逻辑运算", "邏輯運算", "赋值", "賦值", "跳转", "跳轉", "等于", "等於", "实现", "實現", "并", "並")
	for _, tt := range []struct{ name, src string }{
		{"hanzi", hanzi},
		{"pinyin", pinyin},
		{"traditional", traditional.Replace(hanzi)},
		{"ASCII punctuation", strings.NewReplacer("，", ",", "、", ",", "。", ".").Replace(hanzi)},
		{"spaced", strings.NewReplacer("这个程序使用", "这个程序 使用 ", "第八个单元和", "第八个单元 和 ").Replace(hanzi)},
		{"without 个", regexp.MustCompile(`第([一二三四五六七八九十]+)个`).ReplaceAllString(hanzi, "第$1")},
		{"without 并", strings.Replace(hanzi, "，并在", "，在", 1)},
		{"两", strings.Replace(hanzi, "二十三", "两十三", 1)},
		{"toneless pinyin", strings.NewReplacer("ǐ", "i", "ò", "o", "ù", "u", "ì", "i", "è", "e", "í", "i", "é", "e", "ā", "a", "ǎ", "a").Replace(pinyin)},
		{"pinyin forms", strings.NewReplacer("dì-yī gè biāoqiān, bìng", "dìyī biāoqiān, bìng", "fù gěi dì-èr", "fùgěi dì èr", "bìngfēi", "bìng fēi").Replace(pinyin)},
		{"upper case pinyin", strings.ToUpper(pinyin)},
		{"short marker", strings.Replace(hanzi, "这个程序非常非常酷。", "非常非常酷。", 1)},
		{"pinyin marker", strings.Replace(hanzi, "这个程序非常非常酷。", "Fēicháng fēicháng kù.", 1)},
		{"Japanese marker", strings.Replace(hanzi, "这个程序非常非常酷。", "Totemo totemo kakkoii.", 1)},
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
func TestMandarinNone(t *testing.T) {
	src := `这个程序使用数字七。
这个程序哪儿也不去。
这个程序不能写。
这个程序不能读。
这个程序使用第一个数字和第一个数字的和。
这个程序不使用条件。
这个程序使用一个标签。
这个程序不使用差。
这个程序把第一个和赋给第一个数字。
这个程序不使用积。
这个程序实现第一个赋值。
这个程序不使用比。
这个程序不合逻辑。
非常非常酷。`
	p, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.Data[0] != 7 || p.LabelsCount != 1 || p.Tables[Sums].Count != 1 || p.Tables[Assigns].Count != 1 || p.Tables[Statements].Count != 1 {
		t.Errorf("%+v", p)
	}
	for _, tt := range []struct{ from, to string }{
		{"使用数字七", "不使用数字"},
		{"使用第一个数字和第一个数字的和", "不使用和"},
		{"使用一个标签", "不使用标签"},
		{"使用一个标签", "使用二十二个标签"},
		{"把第一个和赋给第一个数字", "不赋值"},
		{"不合逻辑", "不使用逻辑运算"},
		{"不使用条件", "使用第一个数字小于第一个数字的条件"},
		{"哪儿也不去", "总是跳到第一个标签"},
		{"不使用差", "使用第一个数字和第一个数字的差和第一个和和第一个和的差"},
		{"不使用积", "使用第一个数字和第一个数字的积"},
		{"不使用比", "使用第一个数字和第一个数字的比、第一个数字和第一个数字的比和第一个数字和第一个数字的比"},
		{"不合逻辑", "使用并非第一个数字和第一个数字都成立的逻辑运算"},
		{"不能写", "把第一个数字作为巴西序数写出"},
		{"不能写", "把第一个数字作為日語基數写出"},
		{"不能读", "把第一个数字作为字符读入"},
	} {
		if _, err := parse(strings.Replace(src, tt.from, tt.to, 1)); err != nil {
			t.Errorf("%s: %v", tt.to, err)
		}
	}
}

// Mandarin is Very Very Sorted!, mixes with the other languages and its
// two scripts with each other, and broken Mandarin fails.
func TestMandarinDialect(t *testing.T) {
	for _, end := range []string{"Cool.", "This code is very cool.", "这个程序非常酷。"} {
		if _, err := parse(strings.Replace(hanzi, "这个程序非常非常酷。", end, 1)); err == nil || err.Error() != "ERROR, missing or invalid number declaration" {
			t.Errorf("%s: %v", end, err)
		}
	}
	lines := strings.Split(hanzi, "\n")
	for _, other := range []string{english, italian, french, portuguese, japanese, pinyin} {
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
		{"as a chinese cardinal", FormatChineseCardinal}, {"as a chinese ordinal", FormatChineseOrdinal},
		{"als ein chinesischer Kardinal", FormatChineseCardinal}, {"als eine chinesische Ordinalzahl", FormatChineseOrdinal},
		{"come cardinale cinese", FormatChineseCardinal}, {"come ordinale cinese", FormatChineseOrdinal},
		{"comme cardinal chinois", FormatChineseCardinal}, {"comme ordinal chinois", FormatChineseOrdinal},
		{"como cardinal chinês", FormatChineseCardinal}, {"como ordinal chinês", FormatChineseOrdinal},
		{"o chūgokugo no kisū to shite kakimasu", FormatChineseCardinal},
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
	for _, f := range zhFormats {
		for _, w := range []string{f.p.hz, f.p.py} {
			src := strings.Replace(hanzi, "作为意大利语基数写出", "作为 "+w+" 写出", 1)
			if p, err := parse(src); err != nil || p.Entries(Writes)[0].Flags != f.format {
				t.Errorf("%s: %v", w, err)
			}
		}
	}
	for _, tt := range []struct{ from, to string }{
		{"二十三和一百万", "二十三和二十三"},
		{"总是跳到第一个标签", "总是跳到第一个条件"},
		{"作为意大利语基数写出", "作为意大利语基数"},
		{"和第十一个数字的差", "和第十一个的差"},
		{"赋给第四个数字", "赋给第四个和"},
		{"等于第二个数字的条件", "大于第二个数字的条件"},
		{"使用两个标签", "使用两标签"},
		{"一百万", "十亿"},  // declared numbers stop below 1000000000
		{"二十三", "二百五"}, // colloquial 250 is not a number here
		{"都成立", "成立"},
		{"既非第一个数字也非", "既非第一个数字非"},
	} {
		if !strings.Contains(hanzi, tt.from) {
			t.Fatalf("%q is not in the program", tt.from)
		}
		if _, err := parse(strings.Replace(hanzi, tt.from, tt.to, 1)); err == nil {
			t.Errorf("%s parses", tt.to)
		}
	}
}

// Very Very Sorted! reads Mandarin's punctuation and traditional
// characters.
func TestFilterMandarin(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"一、二和三。", "一, 二和三. "},
		{"這個程式，並", "这个程序, 并"},
		{"一,二", "一, 二"},
		{"ドイツ語", "ドイツ語"},
		{"un, deux.", "un, deux."},
	} {
		if got := FilterVeryVery([]byte(tt.in)); got != tt.want {
			t.Errorf("FilterVeryVery(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
