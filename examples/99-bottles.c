/* 99 Bottles of Beer, the song, for Sorted!: the language you should be
   able to sing. Compile it with sorted --from-c 99-bottles.c (add --lang de
   for the German Sorted! version), or with any C compiler. */
#include <stdio.h>

void say(char *s) {
	while (*s)
		putchar(*s++);
}

void number(int n) {
	if (n >= 10)
		number(n / 10);
	putchar('0' + n % 10);
}

void bottles(int n, int capital) {
	if (n == 0) {
		if (capital)
			say("No more");
		else
			say("no more");
	} else
		number(n);
	say(" bottle");
	if (n != 1)
		putchar('s');
	say(" of beer");
}

int main() {
	for (int n = 99; n > 0; n--) {
		bottles(n, 1);
		say(" on the wall, ");
		bottles(n, 0);
		say(".\nTake one down and pass it around, ");
		bottles(n - 1, 0);
		say(" on the wall.\n\n");
	}
	bottles(0, 1);
	say(" on the wall, ");
	bottles(0, 0);
	say(".\nGo to the store and buy some more, ");
	bottles(99, 0);
	say(" on the wall.\n");
	return 0;
}
