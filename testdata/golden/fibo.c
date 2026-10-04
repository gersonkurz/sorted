#include "stdio.h"
#include "stdlib.h"
#include "string.h"

long _[193719] = { 20,71,3,2,1};

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
E(S)B(1,N(10)+N(11))B(2,N(2)+N(5))F E(A)B(1,N(10)=N(4))B(2,N(11)=N(5))
B(3,N(2)=N(5))B(4,N(2)=S(2))B(5,N(4)=N(11))B(6,N(10)=N(11))
B(7,N(11)=S(1))F E(W)B(1,H(N(11)))F E(C)B(1,N(2)==N(1)) B(2,N(2)<N(3)) 
F int main(int,char*[]) {A(3);L(1);CJ(2,1);CJ(4,2);
UJ(3);L(4);A(2);A(6);
UJ(5);L(3);A(5);A(7);
A(1);L(5);W(1);A(4);
UJ(1);L(2);{ F
