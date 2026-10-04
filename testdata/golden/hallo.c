#include "stdio.h"
#include "stdlib.h"
#include "string.h"

long _[193719] = { 0,1,72,97,108,200,111,44,32,87,101,202,116,46,222};

#define N(x) _[x-1]
#define G(x) long x
#define B(x,y) case x: return y;
#define d(x) G(x)(G(n))
#define E(x) d(x){switch(n){
#define L(n) label##n:
#define UJ(n) goto label##n
#define CJ(n,c) if(C(c)) UJ(n)

#define H(x) putchar(x)
#define i(x) x = getchar()
#define F } return 0; }

d(S);d(A);d(W);d(C);
E(S)B(1,N(1)+N(2))F E(A)B(1,N(6)=N(5))B(2,N(15)=N(1))B(3,N(1)=S(1))
B(4,N(211)=N(7))B(5,N(12)=N(5))F E(W)B(1,H(N(N(1))))F E(C)B(1,N(16)==N(N(1))) 
F int main(int,char*[]) {A(1);A(2);A(3);A(4);
A(3);A(5);A(3);L(1);
CJ(2,1);W(1);A(3);UJ(1);
L(2);{ F
