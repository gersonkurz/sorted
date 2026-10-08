/* A Brainfuck interpreter, in the C subset that sorted --from-c compiles,
   so that it runs as a Sorted! program, which is then as capable as
   Brainfuck: Turing complete, up to its memory.

   It reads a Brainfuck program from stdin, up to a '!' or the end of the
   input, and runs it; ',' reads what follows the '!', and 0 at the end of
   the input. A cell is a byte that wraps around; the tape has 30000 of
   them, and moving off either end of it is an error. */

#include <stdio.h>

#define TAPE 30000
#define CODE 5000

char code[CODE];
int match[CODE];
int open[CODE];
unsigned char tape[TAPE];

void say(char *s) {
	while (*s)
		putchar(*s++);
}

int main() {
	int n = 0, depth = 0, c;
	while ((c = getchar()) != -1 && c != '!') {
		if (c == '+' || c == '-' || c == '<' || c == '>' || c == '.' || c == ',' || c == '[' || c == ']') {
			if (n == CODE) {
				say("program too long\n");
				return 1;
			}
			code[n++] = c;
		}
	}
	/* each bracket learns where its partner is */
	for (int i = 0; i < n; i++) {
		if (code[i] == '[') {
			open[depth++] = i;
		} else if (code[i] == ']') {
			if (depth == 0) {
				say("unmatched ]\n");
				return 1;
			}
			int j = open[--depth];
			match[i] = j;
			match[j] = i;
		}
	}
	if (depth != 0) {
		say("unmatched [\n");
		return 1;
	}
	int p = 0;
	for (int pc = 0; pc < n; pc++) {
		switch (code[pc]) {
		case '+':
			tape[p]++;
			break;
		case '-':
			tape[p]--;
			break;
		case '>':
			if (++p == TAPE) {
				say("off the right end of the tape\n");
				return 1;
			}
			break;
		case '<':
			if (p-- == 0) {
				say("off the left end of the tape\n");
				return 1;
			}
			break;
		case '.':
			putchar(tape[p]);
			break;
		case ',':
			c = getchar();
			tape[p] = c == -1 ? 0 : c;
			break;
		case '[':
			if (!tape[p])
				pc = match[pc];
			break;
		case ']':
			if (tape[p])
				pc = match[pc];
			break;
		}
	}
	return 0;
}
