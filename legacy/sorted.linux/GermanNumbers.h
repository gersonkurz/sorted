#ifndef GermanNumbers_H
#define GermanNumbers_H

char* IntegerToGermanCardinal( int n );
bool GermanCardinalToInteger( char** pp, int* pResult );

char* IntegerToGermanOrdinal( int n );
int GermanOrdinalToInteger( char** pp );

#endif

