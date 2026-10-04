#ifndef EnglishNumbers_H
#define EnglishNumbers_H

char* IntegerToEnglishCardinal( int n );
bool EnglishCardinalToInteger( char** pp, int* pResult );

char* IntegerToEnglishOrdinal( int n );
int EnglishOrdinalToInteger( char** pp );

#endif

