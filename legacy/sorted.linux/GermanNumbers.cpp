#include "precomp.h"
#include "GermanNumbers.h"

#define skipws() while((*p == ' ')||(*p == '\t')) p++
#define skipand() skipws(); if( !strnicmp(p,"und",3) ) { p+=3; skipws(); }
#define skipth()                                                            \
    if( !strnicmp(p,"ste",3) && (p[3]==' ' || p[3]=='\t')) p+=3;            \
    else if( !strnicmp(p,"te",2) && (p[2]==' ' || p[2]=='\t')) p+=2;         \
    else if( !strnicmp(p,"n",1) && (p[1]==' ' || p[1]=='\t')) p+=1

static char* SingleDigitCardinal[10] = { 
"", "eins", "zwei", "drei", "vier", "fuenf", "sechs", "sieben", "acht", "neun" };

static char* SingleDigitCardinalCombined[10] = { 
"", "einund", "zweiund", "dreiund", "vierund", "fuenfund", "sechsund", "siebenund", "achtund", "neunund" };

static char* DecFactorCardinal[] = {
"", "", "zwanzig", "dreissig", "vierzig", "fuenfzig", "sechzig", "siebzig", "achtzig", "neunzig" };

static char* ZeroTillTwentyCardinal[20] = { 
"null", "eins", "zwei", "drei", "vier", "fuenf", "sechs", "sieben", "acht", "neun",
"zehn", "elf", "zwoelf", "dreizehn", "vierzehn", "fuenfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn" };

static char* HundredCardinals[10] = {
"", "einhundert","zweihundert","dreihundert","vierhundert","fuenfhundert","sechshundert","siebenhundert","achthundert","neunhundert"};

static char* SingleDigitOrdinal[10] = { 
"", "erste", "zweite", "dritte", "vierte", "fuenfte", "sechste", "siebente", "achte", "neunte" };

static char* DecFactorOrdinal[] = {
"", "", "zwanzigste", "dreissigste", "vierzigste", "fuenfzigste", "sechzigste", "siebzigste", "achtzigste", "neunzigste" };

static char* ZeroTillTwentyOrdinal[20] = { 
"", "erste", "zweite", "dritte", "vierte", "fuenfte", "sechste", "siebente", "achte", "neunte",
"zehnte", "elfte", "zwoelfte", "dreizehnte", "vierzehnte", "fuenfzehnte", "sechzehnte", "siebzehnte", "achtzehnte", "neunzehnte" };

static void AddNumericHalfCardinal( char* szResult, int n, char* pa )
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
                strcat( szResult, SingleDigitCardinalCombined[n % 10] );
                strcat( szResult, DecFactorCardinal[n / 10] );
            }
        }
        if( pa )
            strcat(szResult,pa);
    }
}

char* IntegerToGermanCardinal( int n )
{
    static char szResult[148];

    szResult[0] = 0;

    n %= 1000000000;
    if( n >= 1000000 )
    {
        AddNumericHalfCardinal(szResult,n / 1000000,"millionen");
        n %= 1000000;
    }
    AddNumericHalfCardinal(szResult,n / 1000,"tausend");
    AddNumericHalfCardinal(szResult,n % 1000,"");
    return szResult;
}

bool GermanCardinalToInteger( char** pp, int* pResult )
{
    int nResult = 0, i;
    int nTotalResult = 0;
    bool found;
    
    char* p = *pp;

    if( !strnicmp(p,"null",4) )
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

        for( i = 1; i < 10; i++ )
        {
            if( !strnicmp(p,SingleDigitCardinalCombined[i],strlen(SingleDigitCardinalCombined[i])) )
            {
                nResult += i;
                p+=strlen(SingleDigitCardinalCombined[i]);
                skipws();
                found = true;
                break;
            }
        }
        if( found )
        {
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
        }
        else
        {
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
        }    
        if( !strnicmp(p,"millionen",9) )
        {
            nTotalResult += nResult * 1000000;
            p+=9;
            skipand();
        }
        else if( !strnicmp(p,"tausend",7) )
        {
            nTotalResult += nResult * 1000;
            p+=7;
            skipand();
        }
        else if( !strnicmp(p,"hundert",7) )
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
    if( nTotalResult > 0 )
    {
        *pResult = nTotalResult;
        return true;
    }
    return false;
}

char* IntegerToGermanOrdinal( int n )
{
    static char szResult[148];

    if( n < 1 )
        return "ERROR, ORDINALS ARE POSITIVE INTEGERS";

    char* p = IntegerToGermanCardinal( n );
    if( p )
    {
        char* q = p + strlen(p);

        bool bIsSpecialCase = false;
        for( int i = 1; i < 20; i++ )
        {
            // check if its one
            if( !stricmp(q-strlen(ZeroTillTwentyCardinal[i]),ZeroTillTwentyCardinal[i]) )
            {
                bIsSpecialCase = true;
                strcpy( q-strlen(ZeroTillTwentyCardinal[i]), ZeroTillTwentyOrdinal[i] );
                break;
            }
        }
        if( !bIsSpecialCase )
        {
                strcpy(q,"ste");
        }
    }
    return p;
}

int GermanOrdinalToInteger( char** pp )
{
    int nResult = 0, i;
    int nTotalResult = 0;
    bool found;
    
    char* p = *pp;

    skipws();
    while( *p )
    {
        skipws();
        found = false;
        
        for( i = 1; i < 10; i++ )
        {
            if( !strnicmp(p,HundredCardinals[i],strlen(HundredCardinals[i])) )
            {
                nResult += i*100;
                p+=strlen(HundredCardinals[i]);
                skipand();
                found = true;
                break;
            }
        }

        if( found && !strnicmp(p,"ste",3) )
        {
            p += 3;
            nTotalResult += nResult;
            skipws();
            break;
        }

        found = false;

        for( i = 1; i < 10; i++ )
        {
            if( !strnicmp(p,SingleDigitCardinalCombined[i],strlen(SingleDigitCardinalCombined[i])) )
            {
                nResult += i;
                p+=strlen(SingleDigitCardinalCombined[i]);
                skipws();
                found = true;
                break;
            }
        }
        if( found )
        {
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
            if( found && !strnicmp(p,"ste",3) )
            {
                p += 3;
                nTotalResult += nResult;
                skipws();
                break;
            }
        }
        else
        {
            if( !found )
            {
                for( i = 19; i >= 1; i-- )
                {
                    if( !strnicmp(p,ZeroTillTwentyOrdinal[i],strlen(ZeroTillTwentyOrdinal[i])) )
                    {
                        nResult += i;
                        p+=strlen(ZeroTillTwentyOrdinal[i]);
                        skipth();
                        break;
                    }
                }            
            }
            if( !found )
            {
                for( i = 2; i < 10; i++ )
                {
                    if( !strnicmp(p,DecFactorCardinal[i],strlen(DecFactorCardinal[i])) )
                    {
                        nResult += i*10;
                        p+=strlen(DecFactorCardinal[i]);               
                        skipth();
                        found = true;
                        break;
                    }
                }
            }
            if( !found )
            {
                for( i = 19; i >= 1; i-- )
                {
                    if( !strnicmp(p,ZeroTillTwentyCardinal[i],strlen(ZeroTillTwentyCardinal[i])) )
                    {
                        nResult += i;
                        p+=strlen(ZeroTillTwentyCardinal[i]);
                        skipth();

                        break;
                    }
                }            
            }


        }    
        if( !strnicmp(p,"millionen",9) )
        {
            nTotalResult += nResult * 1000000;
            p+=9;
            skipand();
        }
        if( !strnicmp(p,"millionste",10) )
        {
            nTotalResult += nResult * 1000000;
            p+=10;
            skipws();
            break;
        }
        else if( !strnicmp(p,"tausend",7) )
        {
            nTotalResult += nResult * 1000;
            p+=7;
            if( !strnicmp(p,"ste",3) )
            {
                p += 3;
                skipws();
                break;
            }
            skipand();
        }
        else if( !strnicmp(p,"hundert",7) )
        {
            nTotalResult += nResult * 100;
            p+=7;
            if( !strnicmp(p,"ste",3) )
            {
                p += 3;
                skipws();
                break;
            }
            skipand();
        }
        else
        {
            nTotalResult += nResult;
            if( !strnicmp(p,"ste",3) )
            {
                p += 3;
                skipws();
                break;
            }
            skipws();
            break;
        }
        nResult = 0;

    }

    *pp = p;
    return nTotalResult;
}
