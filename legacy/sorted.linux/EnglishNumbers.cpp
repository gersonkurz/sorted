#include "precomp.h"
#include "EnglishNumbers.h"

#define skipws() while((*p == ' ')||(*p == '\t')) p++
#define skipand() skipws(); if( !strnicmp(p,"and",3) ) { p+=3; skipws(); }
#define skipth() if( !strnicmp(p,"th",2) && (p[2]==' ' || p[2]=='\t')) p+=2; else if( !stricmp(p,"h") ) p+=1;

static char* SingleDigitCardinal[10] = { 
"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine" };

static char* DecFactorCardinal[] = {
"", "", "twenty", "thirty", "fourty", "fifty", "sixty", "seventy", "eighty", "ninety" };

static char* ZeroTillTwentyCardinal[20] = { 
"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
"ten", "eleven", "twelve", "thirteen", "fourteen", "fiveteen", "sixteen", "seventeen", "eighteen", "nineteen" };

static char* HundredCardinals[10] = {
"", "onehundred","twohundred","threehundred","fourhundred","fivehundred","sixhundred","sevenhundred","eighthundred","ninehundred"};

static char* SingleDigitOrdinal[10] = { 
"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "nineth" };

static char* DecFactorOrdinal[] = {
"", "", "twentieth", "thirtieth", "fourtieth", "fiftieth", "sixtieth", "seventieth", "eightieth", "ninetieth" };

static char* ZeroTillTwentyOrdinal[20] = { 
"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth",
"tenth", "eleventh", "twelveth", "thirteenth", "fourteenth", "fifteenth", "sixteenth", "seventeenth", "eighteenth", "nineteenth" };

static char* HundredOrdinals[10] = {
"", "onehundred","twohundred","threehundred","fourhundred","fivehundred","sixhundred","sevenhundred","eighthundred","ninehundred"};

void AddNumericHalfCardinal( char* szResult, int n, char* pa )
{
    if( n )
    {
        strcat( szResult, HundredCardinals[n / 100] );
        n %= 100;
        if( n )
        {
            if( n < 20 )
            {
                strcat( szResult, ZeroTillTwentyCardinal[n] );
            }
            else
            {
                strcat( szResult, DecFactorCardinal[n / 10] );
                strcat( szResult, SingleDigitCardinal[n % 10] );
            }
        }
        if( pa )
            strcat(szResult,pa);
    }
}

char* IntegerToEnglishCardinal( int n )
{
    static char szResult[148];

    szResult[0] = 0;

    n %= 1000000000;
    if( n >= 1000000 )
    {
        AddNumericHalfCardinal(szResult,n / 1000000,"million");
        n %= 1000000;
    }
    AddNumericHalfCardinal(szResult,n / 1000,"thousand");
    AddNumericHalfCardinal(szResult,n % 1000,"");
    return szResult;
}

bool EnglishCardinalToInteger( char** pp, int* pResult )
{
    int nResult = 0, i;
    int nTotalResult = 0;
    bool found;
    
    char* p = *pp;

    if( !strnicmp(p,"zero",4) )
    {
        p+=4;
        skipws();
        *pp = p;
        *pResult = 0;
        return true;
    }

    skipws();
    while( *p )
    {
        skipws();
        for( i = 1; i < 10; i++ )
        {
            if( !strnicmp(p,HundredCardinals[i],strlen(HundredCardinals[i])) )
            {
                nResult += i*100;
                p+=strlen(HundredCardinals[i]);
                skipand();
                break;
            }
        }

        found = false;
        for( i = 2; i < 10; i++ )
        {
            if( !strnicmp(p,DecFactorCardinal[i],strlen(DecFactorCardinal[i])) )
            {
                nResult += i*10;
                p+=strlen(DecFactorCardinal[i]);               
                skipws();
                found = true;
                break;
            }
        }
        if( found )
        {
            for( i = 1; i < 10; i++ )
            {
                if( !strnicmp(p,SingleDigitCardinal[i],strlen(SingleDigitCardinal[i])) )
                {
                    nResult += i;
                    p+=strlen(SingleDigitCardinal[i]);
                    skipws();
                    break;
                }
            }
        }
        else
        {
            
            for( i = 19; i >= 1; i-- )
            {
                if( !strnicmp(p,ZeroTillTwentyCardinal[i],strlen(ZeroTillTwentyCardinal[i])) )
                {
                    nResult += i;
                    p+=strlen(ZeroTillTwentyCardinal[i]);
                    skipws();
                    break;
                }
            }            
        }
    
        if( !strnicmp(p,"million",7) )
        {
            nTotalResult += nResult * 1000000;
            p+=7;
            skipand();
        }
        else if( !strnicmp(p,"thousand",8) )
        {
            nTotalResult += nResult * 1000;
            p+=8;
            skipand();
        }
        else if( !strnicmp(p,"hundred",7) )
        {
            nTotalResult += nResult * 100;
            p+=7;
            skipand();
        }
        else
        {
            nTotalResult += nResult;
            skipws();
            break;
        }
        nResult = 0;

    }

    *pp = p;
    if( nTotalResult )
    {
        *pResult = nTotalResult;
        return true;
    }
    return false;
}

char* IntegerToEnglishOrdinal( int n )
{
    static char szResult[148];

    if( n < 1 )
        return "ERROR, ORDINALS ARE POSITIVE INTEGERS";

    char* p = IntegerToEnglishCardinal( n );
    if( p )
    {
        char* q = p + strlen(p)-1;
        if( *q == 'y' )
            strcpy(q,"ieth");
        else
        {
            q++;

            bool bIsSpecialCase = false;
            for( int i = 1; i < 10; i++ )
            {
                if( !stricmp(q-strlen(SingleDigitCardinal[i]),SingleDigitCardinal[i]) )
                {
                    bIsSpecialCase = true;
                    strcpy( q-strlen(SingleDigitCardinal[i]), SingleDigitOrdinal[i] );
                    break;
                }
            }
            if( !bIsSpecialCase )
            {
                if( q[-1] == 't' )
                    strcpy(q,"h");
                else
                    strcpy(q,"th");
            }
        }
    }
    return p;
}

int EnglishOrdinalToInteger( char** pp )
{
    int nResult = 0, i;
    int nTotalResult = 0;
    bool found;
    char* p = *pp;

    skipws();
    while( *p )
    {
        skipws();
        for( i = 1; i < 10; i++ )
        {
            if( !strnicmp(p,HundredCardinals[i],strlen(HundredCardinals[i])) )
            {
                nResult += i*100;
                p+=strlen(HundredCardinals[i]);
                skipand();
                break;
            }
        }

        found = false;
        for( i = 2; i < 10; i++ )
        {
            if( !strnicmp(p,DecFactorCardinal[i],strlen(DecFactorCardinal[i])) )
            {
                nResult += i*10;
                p+=strlen(DecFactorCardinal[i]);               
                skipws();
                found = true;
                break;
            }
        }
        if( !found )
        {
            for( i = 2; i < 10; i++ )
            {
                if( !strnicmp(p,DecFactorOrdinal[i],strlen(DecFactorOrdinal[i])) )
                {
                    nResult += i*10;
                    p+=strlen(DecFactorOrdinal[i]);               
                    skipws();
                    found = true;
                    break;
                }
            }
        }
 
        if( found )
        {
            found = false;
            for( i = 1; i < 10; i++ )
            {
                if( !strnicmp(p,SingleDigitCardinal[i],strlen(SingleDigitCardinal[i])) )
                {
                    nResult += i;
                    p+=strlen(SingleDigitCardinal[i]);
                    skipws();
                    found = true;
                    break;
                }
            }
            if( !found )
            {
                for( i = 1; i < 10; i++ )
                {
                    if( !strnicmp(p,SingleDigitOrdinal[i],strlen(SingleDigitOrdinal[i])) )
                    {
                        nResult += i;
                        p+=strlen(SingleDigitOrdinal[i]);
                        skipws();
                        found = true;
                        break;
                    }
                }
            }
        }
        else
        {
            for( i = 19; i >= 1; i-- )
            {
                if( !strnicmp(p,ZeroTillTwentyCardinal[i],strlen(ZeroTillTwentyCardinal[i])) )
                {
                    nResult += i;
                    p+=strlen(ZeroTillTwentyCardinal[i]);
                    skipws();
                    found = true;
                    break;
                }
            }
            if( !found )
            {
                for( i = 19; i >= 1; i-- )
                {
                    if( !strnicmp(p,ZeroTillTwentyOrdinal[i],strlen(ZeroTillTwentyOrdinal[i])) )
                    {
                        nResult += i;
                        p+=strlen(ZeroTillTwentyOrdinal[i]);
                        skipws();
                        found = true;
                        break;
                    }
                }
            }
        }
        if( !strnicmp(p,"million",7) )
        {
            nTotalResult += nResult * 1000000;
            p+=7;
            skipand();
            skipth();
        }
        else if( !strnicmp(p,"thousand",8) )
        {
            nTotalResult += nResult * 1000;
            p+=8;
            skipand();
            skipth();
        }
        else if( !strnicmp(p,"hundred",7) )
        {
            nTotalResult += nResult * 100;
            p+=7;
            skipand();
            skipth();
        }
        else 
        {
            skipth();
            nTotalResult += nResult;
            skipws();
            break;
        }
        nResult = 0;
    }
    *pp = p;
    return nTotalResult;
}
