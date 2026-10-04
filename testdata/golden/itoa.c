#include "stdio.h"
#include "stdlib.h"
#include "string.h"

long _[193719] = { 41281927,10,11,48,80,79,1,0};

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

d(S);d(D);d(P);d(R);d(A);d(W);d(C);
E(S)B(1,N(3)+N(4))B(2,N(5)+N(7))F E(D)B(1,N(5)-N(7))B(2,N(1)-N(3))
F E(P)B(1,N(2)*N(3))F E(R)B(1,N(1)/N(2))F E(A)B(1,N(3)=R(1))B(2,N(5)=D(1))
B(3,N(6)=N(5))B(4,N(1)=R(1))B(5,N(N(5))=S(1))B(6,N(3)=P(1))
B(7,N(N(5))=N(8))B(8,N(3)=D(2))B(9,N(5)=S(2))F E(W)B(1,H(N(N(5))))
F E(C)B(1,N(1)==N(8)) B(2,N(8)<N(1)) B(3,N(5)<N(6)) F int main(int,char*[]) {A(7);
A(3);A(2);L(1);A(1);
A(6);A(8);A(4);A(2);
A(5);CJ(2,1);CJ(1,2);L(2);
W(1);A(9);CJ(2,3);{ F
