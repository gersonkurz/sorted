#include "precomp.h"
#include "EnglishNumbers.h"
#include "GermanNumbers.h"
#include "SortedSyntax.h"

typedef bool (SortedSyntax::*PARSERFUNC)();

typedef struct
{
    PARSERFUNC func;
    char* error;
} PARSEQ;

PARSEQ funcs[14] = 
{
    { 0, "number declaration" },
    { 0, "declaration of jumps" },
    { 0, "declaration of output" },
    { 0, "declaration of input" },
    { 0, "sum declaration" },
    { 0, "declaration of conditions" },
    { 0, "label declaration" },
    { 0, "declaration of ordered differences" },
    { 0, "declaration of assignments" },
    { 0, "declaration of products" },
    { 0, "declaration of implementation" },
    { 0, "declaration of ratios" },
    { 0, "declaration of logical operations" },
    { 0, "coolness" },
};

bool SortedSyntax::Parse( CODEINFO* pCode, char* pszExpression )
{
    m_pszExpression = pszExpression;
    m_pCode = pCode;

    funcs[0].func = &SortedSyntax::NUMBER_DECLARATION;
    funcs[1].func = &SortedSyntax::JUMP_DECLARATION;
    funcs[2].func = &SortedSyntax::OUTPUT_DECLARATION;
    funcs[3].func = &SortedSyntax::INPUT_DECLARATION;
    funcs[4].func = &SortedSyntax::SUM_DECLARATION;
    funcs[5].func = &SortedSyntax::CONDITION_DECLARATION;
    funcs[6].func = &SortedSyntax::LABEL_DECLARATION;
    funcs[7].func = &SortedSyntax::DIFF_DECLARATION;
    funcs[8].func = &SortedSyntax::ASSIGN_DECLARATION;
    funcs[9].func = &SortedSyntax::PROD_DECLARATION;
    funcs[10].func = &SortedSyntax::IMPLEMENTATION_DECLARATION;
    funcs[11].func = &SortedSyntax::RATIO_DECLARATION;
    funcs[12].func = &SortedSyntax::NAND_DECLARATION;
    funcs[13].func = &SortedSyntax::COOL;
		
    for( int i = 0; i < 14; i++ )
    {
        if( !(this->*(funcs[i].func))() )
        {
            printf("ERROR, missing or invalid %s\n", funcs[i].error);
            return false;
        }
    }
    return true;
}

bool SortedSyntax::CARDINAL(int& n)
{
    if( EnglishCardinalToInteger( &m_pszExpression, &n ) )
        return true;

    if( GermanCardinalToInteger( &m_pszExpression, &n ) )
        return true;

    return false;
}

bool SortedSyntax::IS_KEYWORD(const char* pszKeyword)
{
    skipWhitespaces();

    int len = strlen(pszKeyword);
    if( isChar(m_pszExpression[len]) )
        return false;

    char c = m_pszExpression[len];
    m_pszExpression[len] = 0;
    bool success = !stricmp(m_pszExpression,pszKeyword);
    m_pszExpression[len] = c;
    if( !success )
        return false;

    m_pszExpression += len;
    skipWhitespaces();
    return true;
}

bool SortedSyntax::IS_KEYWORD_SEQUENCE(const char** pszKeyword)
{
    pushExpression();

    for( int i = 0; pszKeyword[i]; i++ )
        if( !IS_KEYWORD(pszKeyword[i]) )
        {
            popExpression();
            return false;
        }

    return true;
}

bool SortedSyntax::USES_NO_NUMBERS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","numbers", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","zahlen", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) ||
        IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_NUMBERS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","uses","the", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","die", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_NUMBER() || NUMBER_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::StoreSingleNumber(int n)
{
    long* psi = m_pCode->Data;

    for( int index = 0; index < m_pCode->nNumbersUsed; index++ )
        if( psi[index] == n )
            return false;

    psi[index] = n;
    m_pCode->nNumbersUsed++;
    return true;
}

bool SortedSyntax::SINGLE_NUMBER()
{
    pushExpression();
    if( IS_KEYWORD("number") || IS_KEYWORD("zahl") )
    {
        int n;

        if( CARDINAL(n) )
        {
            return StoreSingleNumber(n);
        }
    }
    popExpression();
    return false;
}

bool SortedSyntax::NUMBER_SEQUENCE()
{
    pushExpression();
    
    int n;
    
    if( !IS_KEYWORD("numbers") && !IS_KEYWORD("zahlen") )
        return false;

loop:
    if( !CARDINAL(n) )
        goto failed;

    if( !StoreSingleNumber(n) )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !CARDINAL(n) )
                goto failed;

            return StoreSingleNumber(n);
        }
        goto loop;
    }
failed:
    popExpression();
    return false;
}

bool SortedSyntax::NUMBER_DECLARATION()
{
    if( USES_NUMBERS() || USES_NO_NUMBERS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::IDENTIFIER( PSLID ps )
{
    return INDIRECT_USE(ps) || DIRECT_USE(ps);
}

bool SortedSyntax::INDIRECT_USE( PSLID ps )
{
    pushExpression();
    static const char* szSequenceE[] = { "the","cell","indexed", "by", NULL };
    static const char* szSequenceG[] = { "diejenige","zelle","die","indiziert","wird","durch",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) ||
        IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( DIRECT_USE(ps) )
        {
            ps->Type |= SLID_TYPE_INDIRECT;
            return true;
        }

    popExpression();
    return false;
}

bool SortedSyntax::DIRECT_USE( PSLID ps )
{
    pushExpression();
    if( IS_KEYWORD( "the" ) || IS_KEYWORD( "die" ) || IS_KEYWORD( "das" ) || IS_KEYWORD( "der" )  || IS_KEYWORD( "den" ) )
    {
        int n = EnglishOrdinalToInteger(&m_pszExpression);
        if( !n )
            n = GermanOrdinalToInteger(&m_pszExpression);
        if( n ) 
        {
            if( IS_KEYWORD("sum") || IS_KEYWORD("summe") )
            {
                ps->Type = SLID_TYPE_SUM;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("number") || IS_KEYWORD("zahl") )
            {
                ps->Type = SLID_TYPE_NUMBER;
                ps->Index = n-1;
                return true;
            }
            else if( (IS_KEYWORD("ordered") && IS_KEYWORD("difference")) ||
                     (IS_KEYWORD("geordnete") && IS_KEYWORD("differenz")) )
            {
                ps->Type = SLID_TYPE_DIFF;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("product") || IS_KEYWORD("produkt") )
            {
                ps->Type = SLID_TYPE_PROD;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("ratio") || IS_KEYWORD("verhaeltnis") )
            {
                ps->Type = SLID_TYPE_RATIO;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("cell") || IS_KEYWORD("zelle") )
            {
                ps->Type = SLID_TYPE_CELL;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("assignment") || IS_KEYWORD("zuweisung") )
            {
                ps->Type = SLID_TYPE_ASSIGN;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("jump") || IS_KEYWORD("sprungbefehl") )
            {
                ps->Type = SLID_TYPE_JUMP;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("label") || IS_KEYWORD("sprungziel") )
            {
                ps->Type = SLID_TYPE_LABEL;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("condition") || IS_KEYWORD("bedingung") )
            {
                ps->Type = SLID_TYPE_CONDITION;
                ps->Index = n-1;
                return true;
            }
            else if( IS_KEYWORD("output") || IS_KEYWORD("ausgabe") )
            {
                ps->Type = SLID_TYPE_WRITE;
                ps->Index = n-1;
                return true;
            }
        }
    }
    popExpression();
    return false;
}

bool SortedSyntax::SUM_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_SUMS]);
    SLIDE* pCell = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IS_KEYWORD("of") || IS_KEYWORD("von") || IS_KEYWORD("aus") )
        if( IDENTIFIER(&(pCell->_[0])) )
            if( IS_KEYWORD("and") || IS_KEYWORD("und") )
                if( IDENTIFIER(&(pCell->_[1])) )
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_SUMS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","sums", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","summen", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG))
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_SUMS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","uses","the", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","die", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_SUM() || SUM_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_SUM()
{
    pushExpression();
    if( IS_KEYWORD("sum") || IS_KEYWORD("summe") )
        if( SUM_SPEC() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SUM_SEQUENCE()
{
    pushExpression();
    
    if( !IS_KEYWORD("sums") && !IS_KEYWORD("summen") )
        return false;

loop:
    if( !SUM_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !SUM_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

void SortedSyntax::PrepareCodeInfo( int nType )
{
    m_pCode->Type[nType].Count = 0;
    m_pCode->Type[nType].Index = m_pCode->nTypeCount;
}

bool SortedSyntax::SUM_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_SUMS);
    
    if( USES_SUMS() || USES_NO_SUMS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::COOL()
{
    static const char* szSequenceE[] = { "this","code", "is","cool",".",NULL };
    static const char* szSequenceEs[] = { "cool",".",NULL };
    static const char* szSequenceG[] = { "dieses","programm","ist","hervorragend",".",NULL };
    static const char* szSequenceGs[] = { "hervorragend",".",NULL };

    pushExpression();
    return  IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceEs) ||
            IS_KEYWORD_SEQUENCE(szSequenceG) || IS_KEYWORD_SEQUENCE(szSequenceGs);
}

bool SortedSyntax::DIFF_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_DIFFS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IS_KEYWORD("between") || IS_KEYWORD("zwischen") )
        if( IDENTIFIER(&(ps->_[0])) )
            if( IS_KEYWORD("and") || IS_KEYWORD("und"))
                if( IDENTIFIER(&(ps->_[1])) )
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_DIFFS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","ordered", "differences", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","geordneten","differenzen", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_DIFFS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","uses","the", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","die", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_DIFF() || DIFF_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_DIFF()
{
    pushExpression();
    if( (IS_KEYWORD("ordered") && IS_KEYWORD("difference")) ||
        (IS_KEYWORD("geordnete") && IS_KEYWORD("differenz")) )
        if( DIFF_SPEC() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::DIFF_SEQUENCE()
{
    pushExpression();
    
    if( !(IS_KEYWORD("ordered") && IS_KEYWORD("differences")) && 
        !(IS_KEYWORD("geordnete") && IS_KEYWORD("differenz")))
        return false;

loop:
    if( !DIFF_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !DIFF_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::DIFF_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_DIFFS);

    if( USES_DIFFS() || USES_NO_DIFFS() )
        return IS_KEYWORD(".");

    return false;
}


bool SortedSyntax::PROD_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_PRODS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IS_KEYWORD("of") || IS_KEYWORD("von") )
        if( IDENTIFIER(&(ps->_[0])) )
            if( IS_KEYWORD("and") || IS_KEYWORD("und") )
                if( IDENTIFIER(&(ps->_[1])) )
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_PRODS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","products", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","produkte", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_PRODS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","uses","the", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_PROD() || PROD_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_PROD()
{
    pushExpression();
    if( IS_KEYWORD("product") || (IS_KEYWORD("das") && IS_KEYWORD("produkt")) )
        if( PROD_SPEC() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::PROD_SEQUENCE()
{
    pushExpression();
    
    if( !IS_KEYWORD("products") &&
        !(IS_KEYWORD("die") && IS_KEYWORD("produkte")) )
        return false;

loop:
    if( !PROD_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !PROD_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::PROD_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_PRODS);

    if( USES_PRODS() || USES_NO_PRODS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::RATIO_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_RATIOS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IS_KEYWORD("of") )
        if( IDENTIFIER(&(ps->_[0])) )
            if( IS_KEYWORD("to") )
                if( IDENTIFIER(&(ps->_[1])) )
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_RATIOS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","ratios", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","verhaeltnisse", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_RATIOS()
{
    pushExpression();
    static const char* szSequence[] = { "this","code","uses","the", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequence) )
        if( SINGLE_RATIO() || RATIO_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_RATIO()
{
    pushExpression();
    if( IS_KEYWORD("ratio") )
        if( RATIO_SPEC() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::RATIO_SEQUENCE()
{
    pushExpression();
    
    if( !IS_KEYWORD("ratios") )
        return false;

loop:
    if( !RATIO_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and")|| IS_KEYWORD("und") )
        {
            if( !RATIO_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::RATIO_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_RATIOS);

    if( USES_RATIOS() || USES_NO_RATIOS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::NAND_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_NANDS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IS_KEYWORD("of") && IS_KEYWORD("not") )
        if( IDENTIFIER(&(ps->_[0])) )
            if( IS_KEYWORD("and") && IS_KEYWORD("not") )
                if( IDENTIFIER(&(ps->_[1])) )
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_NANDS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","logical","operations",NULL };
    static const char* szSequenceG[] = { "dieses","programm","ist","unlogisch",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_NANDS()
{
    pushExpression();
    static const char* szSequence[] = { "this","code","uses","the", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequence) )
        if( SINGLE_NAND() || NAND_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_NAND()
{
    pushExpression();
    if( IS_KEYWORD("logical") && IS_KEYWORD("operation") )
        if( NAND_SPEC() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::NAND_SEQUENCE()
{
    pushExpression();
    
    if( !(IS_KEYWORD("logical") && IS_KEYWORD("operations")) )
        return false;

loop:
    if( !NAND_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and")|| IS_KEYWORD("und") )
        {
            if( !NAND_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::NAND_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_NANDS);

    if( USES_NANDS() || USES_NO_NANDS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::USES_NO_LABELS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","labels",NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","sprungziele",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_LABELS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","uses", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
    {
        int n;
        if( EnglishCardinalToInteger(&m_pszExpression,&n) ||
            GermanCardinalToInteger(&m_pszExpression,&n) ) 
        {
            if( ( ( n == 1 ) && IS_KEYWORD("label") ) ||
                IS_KEYWORD("labels") )
            {
                m_pCode->nLabelsCount = n;
                return true;
            }
            if( ( ( n == 1 ) && IS_KEYWORD("sprungziel") ) ||
                IS_KEYWORD("sprungziele") )
            {
                m_pCode->nLabelsCount = n;
                return true;
            }
        }
    }

    popExpression();
    return false;
}

bool SortedSyntax::LABEL_DECLARATION()
{
    if( USES_LABELS() || USES_NO_LABELS() )
        return IS_KEYWORD(".");

    return false;
}


bool SortedSyntax::ASSIGN_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_ASSIGNS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IDENTIFIER(&(ps->_[0])) )
        if( IS_KEYWORD("to") || IS_KEYWORD("an") )
            if( IDENTIFIER(&(ps->_[1])) )
                if( (ps->_[1].Type & 0xFF) == SLID_TYPE_NUMBER)
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_ASSIGNS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","assignments",NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","zuweisungen",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_ASSIGNS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","assigns", NULL };
    static const char* szSequenceG[] = { "dieses","programm","weisst","zu", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_ASSIGN() || ASSIGN_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

void SortedSyntax::CancelSpec( int n )
{
    SLIDE_INFO* psi = &(m_pCode->Type[n]);
    if( psi->Count > 0 )
        psi->Count--;
}

bool SortedSyntax::SINGLE_ASSIGN()
{
    pushExpression();

    if( ASSIGN_SPEC() )
        if( *m_pszExpression == '.' )
            return true;
        else 
            CancelSpec(SLIDE_INFO_ASSIGNS);
   
    popExpression();
    return false;
}

bool SortedSyntax::ASSIGN_SEQUENCE()
{
    pushExpression();
    
loop:
    if( !ASSIGN_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !ASSIGN_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::ASSIGN_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_ASSIGNS);

    if( USES_ASSIGNS() || USES_NO_ASSIGNS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::OUTPUT_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_WRITES]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IDENTIFIER(&(ps->_[0])) )
    {
        if( (IS_KEYWORD("as") && IS_KEYWORD("a")) || IS_KEYWORD("als") )
        {
            if( IS_KEYWORD("character") || (IS_KEYWORD("ein") && IS_KEYWORD("zeichen")) )
            {
                ps->Flags = SLIDE_FORMAT_CHARACTER;
            }
            else if( (IS_KEYWORD("english") && IS_KEYWORD("cardinal"))  
                    || (IS_KEYWORD("ein") && IS_KEYWORD("englischer") && IS_KEYWORD("Kardinal")) )
            {
                ps->Flags = SLIDE_FORMAT_ENGLISH_CARDINAL;
            }
            else if( (IS_KEYWORD("english") && IS_KEYWORD("ordinal"))  
                    || (IS_KEYWORD("ein") && IS_KEYWORD("englische") && IS_KEYWORD("Ordinalzahl")) )
            {
                ps->Flags = SLIDE_FORMAT_ENGLISH_ORDINAL;
            }
            else if( (IS_KEYWORD("german") && IS_KEYWORD("cardinal"))  
                    || (IS_KEYWORD("ein") && IS_KEYWORD("deutscher") && IS_KEYWORD("Kardinal")) )
            {
                ps->Flags = SLIDE_FORMAT_GERMAN_CARDINAL;
            }
            else if( (IS_KEYWORD("german") && IS_KEYWORD("ordinal"))  
                    || (IS_KEYWORD("eine") && IS_KEYWORD("deutsche") && IS_KEYWORD("Ordinalzahl")) )
            {
                ps->Flags = SLIDE_FORMAT_GERMAN_ORDINAL;
            }
            else goto failed;

            psi->Count++;
            m_pCode->nTypeCount++;
            return true;
        }
    }
failed:
    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_OUTPUT()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","produce","any","output",NULL };
    static const char* szSequenceEs[] = { "this","code","cannot","write",NULL };
    static const char* szSequenceG[] = { "dieses","programm","erzeugt","keine","ausgaben",NULL };
    static const char* szSequenceGs[] = { "dieses","programm","kann","nicht","schreiben",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) ||
        IS_KEYWORD_SEQUENCE(szSequenceEs) ||
        IS_KEYWORD_SEQUENCE(szSequenceG) || 
        IS_KEYWORD_SEQUENCE(szSequenceGs) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_OUTPUT()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","writes", NULL };
    static const char* szSequenceG[] = { "Dieses","programm","schreibt", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_OUTPUT() || OUTPUT_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_OUTPUT()
{
    pushExpression();

    if( OUTPUT_SPEC() )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::OUTPUT_SEQUENCE()
{
    pushExpression();
    
loop:
    if( !OUTPUT_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !OUTPUT_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::OUTPUT_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_WRITES);

    if( USES_OUTPUT() || USES_NO_OUTPUT() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::INPUT_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_READS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IDENTIFIER(&(ps->_[0])) )
    {
        if( (IS_KEYWORD("as") && IS_KEYWORD("a")) ||
			(IS_KEYWORD("als") ) )
        {
            if( IS_KEYWORD("character") || (IS_KEYWORD("ein") && IS_KEYWORD("zeichen")) )
            {
                ps->Flags = SLIDE_FORMAT_CHARACTER;
            }
            else goto failed;

            psi->Count++;
            m_pCode->nTypeCount++;
            return true;
        }
    }
failed:
    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_INPUT()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","produce","any","input",NULL };
    static const char* szSequenceEs[] = { "this","code","cannot","read",NULL };
    static const char* szSequenceG[] = { "dieses","programm","liest","keine","eingaben",NULL };
    static const char* szSequenceGs[] = { "dieses","programm","kann","nicht","lesen",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) ||
        IS_KEYWORD_SEQUENCE(szSequenceEs) ||
        IS_KEYWORD_SEQUENCE(szSequenceG) || 
        IS_KEYWORD_SEQUENCE(szSequenceGs) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_INPUT()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","reads", NULL };
    static const char* szSequenceG[] = { "dieses","programm","liest", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_INPUT() || INPUT_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_INPUT()
{
    pushExpression();

    if( INPUT_SPEC() )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::INPUT_SEQUENCE()
{
    pushExpression();
    
loop:
    if( !INPUT_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !INPUT_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::INPUT_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_READS);

    if( USES_INPUT() || USES_NO_INPUT() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::COMPARISON( PSLIDE ps )
{
    static const char* szSequence1E[] = { "is","equal","to", NULL };
    static const char* szSequence2E[] = { "is","less","than", NULL };
    static const char* szSequence1G[] = { "ist", "gleich", NULL };
    static const char* szSequence2G[] = { "ist","kleiner","als", NULL };

    pushExpression();
    if( IS_KEYWORD_SEQUENCE(szSequence1E) ||  IS_KEYWORD_SEQUENCE(szSequence1G) )
    {
        ps->Flags = SLIDE_COMPARE_EQUAL;
        return true;
    }
    popExpression();
    if( IS_KEYWORD_SEQUENCE(szSequence2E) || IS_KEYWORD_SEQUENCE(szSequence2G) )
    {
        ps->Flags = SLIDE_COMPARE_LESS;
        return true;
    }
    return false;
}

bool SortedSyntax::CONDITION_SPEC()
{
    static const char* szSequenceE[] = { "the","condition","that", NULL };
    static const char* szSequenceG[] = { "die","bedingung","dass", NULL };

    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_CONDITIONS]);
    SLIDE* pCell = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IS_KEYWORD_SEQUENCE(szSequenceE) ||IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( IDENTIFIER(&(pCell->_[0])) )
            if( COMPARISON(pCell) )
                if( IDENTIFIER(&(pCell->_[1])) )
                {
                    psi->Count++;
                    m_pCode->nTypeCount++;
                    return true;
                }

    popExpression();
    return false;
}


bool SortedSyntax::USES_NO_CONDITIONS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","not","use","any","conditions",NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt","keine","bedingungen",NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_CONDITIONS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","uses", NULL };
    static const char* szSequenceG[] = { "dieses","programm","benutzt", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_CONDITION() || CONDITION_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_CONDITION()
{
    pushExpression();

    if( CONDITION_SPEC() )
        if( *m_pszExpression == '.' )
            return true;
        else CancelSpec(SLIDE_INFO_CONDITIONS);

    popExpression();
    return false;
}

bool SortedSyntax::CONDITION_SEQUENCE()
{
    pushExpression();
    
loop:
    if( !CONDITION_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !CONDITION_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::CONDITION_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_CONDITIONS);

    if( USES_CONDITIONS() || USES_NO_CONDITIONS() )
        return IS_KEYWORD(".");

    return false;
}

bool SortedSyntax::STATEMENT_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_STATEMENTS]);
    SLIDE* ps = &(m_pCode->Code[psi->Index+psi->Count]);

    if( IDENTIFIER(&(ps->_[0])) )
    {
        psi->Count++;
        m_pCode->nTypeCount++;
        return true;
    }

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_STATEMENT()
{
    pushExpression();

    if( STATEMENT_SPEC() )
        if( *m_pszExpression == '.' )
            return true;
        else 
            CancelSpec(SLIDE_INFO_STATEMENTS);
   
    popExpression();
    return false;
}

bool SortedSyntax::STATEMENT_SEQUENCE()
{
    pushExpression();
    
loop:
    if( !STATEMENT_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !STATEMENT_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::IMPLEMENTATION_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_STATEMENTS);

    pushExpression();
    static const char* szSequenceE[] = { "this","code","implements", NULL };
    static const char* szSequenceG[] = { "dieses","programm","implementiert", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_STATEMENT() || STATEMENT_SEQUENCE() )
            return IS_KEYWORD(".");

    popExpression();
    return false;
}


bool SortedSyntax::JUMP_SPEC()
{
    pushExpression();

    SLIDE_INFO* psi = &(m_pCode->Type[SLIDE_INFO_JUMPS]);
    SLIDE* pCell = &(m_pCode->Code[psi->Index+psi->Count]);

    static const char* szSequence1E[] = { "always","goes", "to", NULL };
    static const char* szSequence2E[] = { "sometimes","goes", "to", NULL };
    static const char* szSequence3E[] = { "is","true", NULL };
    static const char* szSequence1G[] = { "springt","immer", "an", NULL };
    static const char* szSequence2G[] = { "springt","manchmal", "an", NULL };
    static const char* szSequence3G[] = { "wahr","ist", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequence1E) || IS_KEYWORD_SEQUENCE(szSequence1G) )
    {
        if( IDENTIFIER(&(pCell->_[0])) )
            if( pCell->_[0].Type == SLID_TYPE_LABEL )
            {
                pCell->Flags = SLIDE_UNCONDITIONAL_JUMP;
                psi->Count++;
                m_pCode->nTypeCount++;
                return true;
            }
    }
    else if( IS_KEYWORD_SEQUENCE(szSequence2E) || IS_KEYWORD_SEQUENCE(szSequence2G) )
    {
        if( IDENTIFIER(&(pCell->_[0])) )
            if( pCell->_[0].Type == SLID_TYPE_LABEL )
                if( IS_KEYWORD("if") || IS_KEYWORD("wenn") || IS_KEYWORD("falls") )
                    if( IDENTIFIER(&(pCell->_[1])) )
                        if( pCell->_[1].Type == SLID_TYPE_CONDITION )
                            if( IS_KEYWORD_SEQUENCE(szSequence3E) || IS_KEYWORD_SEQUENCE(szSequence3G) )
                            {
                                pCell->Flags = SLIDE_CONDITIONAL_JUMP;
                                psi->Count++;  
                                m_pCode->nTypeCount++;
                                return true;
                            }
    }

    popExpression();
    return false;
}

bool SortedSyntax::USES_NO_JUMPS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code","does","never","go","anywhere", NULL };
    static const char* szSequenceG[] = { "dieses","programm","geht","nirgendwo","hin", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        return true;

    popExpression();
    return false;
}

bool SortedSyntax::USES_JUMPS()
{
    pushExpression();
    static const char* szSequenceE[] = { "this","code", NULL };
    static const char* szSequenceG[] = { "dieses","programm", NULL };

    if( IS_KEYWORD_SEQUENCE(szSequenceE) || IS_KEYWORD_SEQUENCE(szSequenceG) )
        if( SINGLE_JUMP() || JUMP_SEQUENCE() )
            return true;

    popExpression();
    return false;
}

bool SortedSyntax::SINGLE_JUMP()
{
    pushExpression();

    if( JUMP_SPEC() )
        if( *m_pszExpression == '.' )
            return true;
        else CancelSpec(SLIDE_INFO_JUMPS);


    popExpression();
    return false;
}

bool SortedSyntax::JUMP_SEQUENCE()
{
    pushExpression();
    
loop:
    if( !JUMP_SPEC() )
        goto failed;

    if( IS_KEYWORD(",") )
    {
        if( IS_KEYWORD("and") || IS_KEYWORD("und") )
        {
            if( !JUMP_SPEC() )
                goto failed;

            return true;
        }
        goto loop;
    }

failed:
    popExpression();
    return false;
}

bool SortedSyntax::JUMP_DECLARATION()
{
    PrepareCodeInfo(SLIDE_INFO_JUMPS);
    
    if( USES_JUMPS() || USES_NO_JUMPS() )
        return IS_KEYWORD(".");

    return false;
}

