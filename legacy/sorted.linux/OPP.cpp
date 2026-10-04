#include "precomp.h"
#include "OPP.h"
#include "ctype.h"
#include "time.h"

bool OPPM::Fits(char** qq,PList* pArguments)
{
    char* p = *qq;
    int n = strlen((char*)m_strName);
    if( strncmp(p,(char*)m_strName,n) == 0 )
    {
        p += n;
        if( *p == '(' )
        {
            p++;
            char* r = p;
            int level = 0;
            PList arguments;
            char* s = r;
            pArguments->DeleteContents();
            while( *r && (*r != ')' || level) )
            {
                if( *r == '(' )
                    level++;
                else if( *r == ')' && level )
                    level--;
                else if( !level && (*r == ',') )
                {
                    *r = 0;
                    pArguments->AddTail( new PString(s) );
                    *r = ',';
                    s = r+1;
                }
                r++;
            }
            if( s )
            {
                char c = *r;
                *r = 0;
                pArguments->AddTail( new PString(s) );
                *r = c;
                *qq = r+1;
            }
            return true;
        }
    }
    return false;
}

static char* ExpandArgument(char* q, char* r,int n)
{
    if( n==1 )
        *q++ = '\"';
    else if( n==2 )
        *q++ = '\'';

    strcpy(q,r);
    q += strlen(q);

    if( n==1 )
        *q++ = '\"';
    else if( n==2 )
        *q++ = '\'';

    return q;
}

bool OPPM::Parse(char** _0,PList* _1)
{
    char* _2 = __D, *_3 = *_0;
    int _4 = 0, _5;
    PString* _6;

    while(*_2)
    {
        if( !strnicmp(_2,"#\"",2) )
        {
            _2 += 2;
            _4 = 1;
        }
        else if( !strnicmp(_2,"#'",2) )
        {
            _2 += 2;
            _4 = 2;
        }
        else if( !strnicmp(_2,"##,#",4) )
        {
            _2+=4;
            *_3++ = '#';

            if( *_2 == '#') 
            {
                *_3++ = '#';
                _2++;
            }
            if( isdigit(*_2) )
            {
                while( isdigit(*_2) )
                    *_3++ = *_2++;
            }
            else *_3++ = *_2++;;
        }
        else if( !(*_2-'#') && isdigit(_2[1]) )
        {
            _2++;
            _5 = strtol(_2,&_2,10);
            _6 = (PString*) _1->Find(_5);
            if( !_6 )
            {
                printf( "*** ERROR, not enough arguments for macro %s", (char*) __D);
                return false;
            }
            if( strncmp(_2,"..n",3) == 0 )
            {
                _2+=3;
                while(true)
                {
                    _3 = ExpandArgument(_3,*_6,_4);
                    _6 = (PString*) _6->m_pNext;
                    if( _6 )
                        *_3++ = ',';
                    else break;
                }
            }
            else _3 = ExpandArgument(_3,*_6,_4);
            _4 = 0;
        }
        else
            *_3++ = *_2++;
    }
    *_0 = _3;
    return true;
}

OPPM::OPPM( const char* pszName, const char* pszDefinition )
    :   m_strName( pszName ), __D( pszDefinition )
{
}

OPP::OPP()
{
    m_szInput = new char[MAX_COLUMNS_PER_LINE*2];
    m_szOutput = m_szInput ? m_szInput + MAX_COLUMNS_PER_LINE : 0;
}

OPP::~OPP()
{
    delete [] m_szInput;
}

bool OPP::AddMacroDefinition( char** pp )
{
    char* q = *pp;
    while( *q && *q != ' ' )
        q++;

    if( !*q )
        return false;

    *q++ = 0;     

    m_Macros.AddTail( new OPPM(*pp,q) );
    *pp = q + strlen(q);
    return true;
}

const char* OPP::ExtractIncludeFileName( const char* p )
{
    static char szFileName[317];

    char* q = szFileName;

    while(true)
    {
        if((p[0]=='\\')&&(p[1]=='\\'))
        {
            *q++ = '.';
            *q++ = '.';
            p+=2;
        }
        else if((p[0]=='/')&&(p[1]=='/'))
        {
            *q++ = '\\';
            *q++ = '\\';
            p+=2;
        }
        else if((p[0]=='.')&&(p[1]=='.'))
        {
            *q++ = '\\';
            p+=2;
        }
        else if((p[0]=='\\')&&(p[1]=='.'))
        {
            *q++ = '.';
            p+=2;
        }
        else if( *p == '.' )
        {
            *q = 0;
            break;
        }
        else
        {
            *q++ = *p++;
        }
    }
    return szFileName;
}

bool OPP::ProcessSingleLine()
{
restart_processing:
    char* pszInputLine = m_szInput;
    char* pszWritePos = m_szOutput;
    bool bAtLeastOneMacroExpanded = false;

    while( *pszInputLine )
    {
        if( (pszInputLine[0] == '#') && (pszInputLine[1] == '#') )
        {
            switch(pszInputLine[2])
            {
            case 'i':
                strcpy(pszWritePos,"complex(0,1)");
                pszWritePos += strlen(pszWritePos);               
                pszInputLine += 3;
                break;
            case '_':
                sprintf(pszWritePos,"%d",m_nLineNumber-5);
                pszWritePos += strlen(pszWritePos);
                pszInputLine += 3;
                break;
            case '$':
                sprintf(pszWritePos,"%d",rand());
                pszWritePos += strlen(pszWritePos);
                pszInputLine += 3;
                break;
            case '{':
                sprintf(pszWritePos,"%d",m_nCurlyBracketsOpen);
                pszWritePos += strlen(pszWritePos);
                pszInputLine += 3;
                break;
            case '}':
                sprintf(pszWritePos,"%d",m_nCurlyBracketsClose % 5 );
                pszWritePos += strlen(pszWritePos);
                pszInputLine += 3;
                break;
            case '.':
                if( m_iSkipIndex )
                    m_iSkipIndex--;
                pszInputLine += 3;
                break;
            case ':':
                pszInputLine += 3;
                if( !AddMacroDefinition(&pszInputLine) )
                {
                    printf( "*** ERROR, invalid syntax for macro %s", pszInputLine );
                    return false;
                }
                break;
            case '@':
                {
                    if( m_iSkipIndex )
                        m_iSkipIndex--;
                    pszInputLine += 3;
                    char* pReadPos = (char*) pszInputLine ;
                    OPPCCE* pC = CheckCC(&pReadPos);
                    if( pC )
                    {
                        pszInputLine = pReadPos;
                        if( !pC->Evaluate() )
                            m_iSkipIndex++;
                        delete pC;
                    }
                    else
                    {
                        printf( "*** ERROR, invalid syntax for conditional compilation statement %s", pszInputLine );
                        return false;
                    }
                }
                break;
            case '~':   
                {
                    pszInputLine += 2;
                    char* pReadPos = (char*) pszInputLine ;
                    OPPCCE* pC = CheckCC(&pReadPos);
                    if( pC )
                    {
                        pszInputLine = pReadPos;
                        if( !pC->Evaluate() )
                            m_iSkipIndex++;
                        delete pC;
                    }
                    else
                    {
                        printf( "*** ERROR, invalid syntax for conditional compilation statement %s", pszInputLine );
                        return false;
                    }
                }
                break;
            case '<':
                #ifdef _WIN32
                *(pszWritePos++) = '\r';
                #endif
                *(pszWritePos++) = '\n';
                return IProcessFile(ExtractIncludeFileName(pszInputLine+3));
            default:
                printf( "*** ERROR, unknown preprocessor directive \"%s\" in line %ld.",pszInputLine,m_nLineNumber );
                return false;
            }
        }
        else if( !m_iSkipIndex )
        {
            if( !IsAMacro(&pszWritePos,&pszInputLine) )
            {
                char c = *(pszInputLine++);

                if( c == '{' )
                    m_nCurlyBracketsOpen++;
                else if( c == '}' )
                    m_nCurlyBracketsOpen++;

                *(pszWritePos++) = c;
            }
            else 
            {
                bAtLeastOneMacroExpanded = true;
            }
        }
        else pszInputLine++;
    }
    if( bAtLeastOneMacroExpanded )
    {
        *pszWritePos = 0;
        strcpy(m_szInput,m_szOutput);
        goto restart_processing;
    }

    *(pszWritePos++) = '\n';
    *pszWritePos = 0;
    return true;
}

bool OPP::IsAMacro(char** pszWritePos,char** qq)
{
    PList args;
    ENUMERATE( &m_Macros, OPPM, m )
        if( m->Fits(qq,&args) )
            return m->Parse(pszWritePos,&args);

    return false;
}

bool OPP::IProcessFile( const char* pszSource )
{
    FILE* fpSource = fopen(pszSource,OPEN_FILE_FOR_READING);
    if( !fpSource )
    {
        printf( "*** ERROR, unable to open file %s for reading", pszSource );
        return false;
    }
    bool bSuccess = true;
    m_nLineNumber = m_nCurlyBracketsClose = m_nCurlyBracketsOpen = 0;
    while( !feof(fpSource) )
    {
        if( fgets(m_szInput,MAX_COLUMNS_PER_LINE,fpSource) )
        {
            m_nLineNumber++;

            if( m_szInput[strlen(m_szInput)-1] == '\n' )
                m_szInput[strlen(m_szInput)-1] = 0;
            if( !ProcessSingleLine() )
            {
                bSuccess = false;
                break;
            }
            fputs(m_szOutput,m_fpTarget);
        }
        else break;
    }

    fclose(fpSource);
    return bSuccess; 
}

bool OPP::ProcessFile( const char* pszSource, const char* pszTarget )
{
    srand(time(0));
    m_iSkipIndex = 0;

    m_fpTarget = fopen(pszTarget,OPEN_FILE_FOR_WRITING);
    if( !m_fpTarget )
    {
        printf( "*** ERROR, unable to open file %s for writing", pszTarget );
        return false;
    }
    
    bool bSuccess = IProcessFile( pszSource );
    fclose( m_fpTarget );
    m_fpTarget = 0;
    return bSuccess;
}

bool OPPCCE::Evaluate()
{
    if( m_lCount == 2 )
    {
        bool a = ((OPPCCE*)m_pHead)->Evaluate() ? false : true;
        bool b = ((OPPCCE*)m_pTail)->Evaluate() ? false : true;

        return a && b;
    }
    else if( !IsEmptyString((char*)m_strVariable) )
    {
        char* p = getenv((char*)m_strVariable);
        if( p && *p != '0' && stricmp(p,"FALSE") )
        {
            return true;
        }
    }
    return false;
}

OPPCCE* OPP::CCObject(char** ppReadPos)
{
    char* pReadPos = *ppReadPos;
    if( *pReadPos == '(' )
    {
        pReadPos++;
        OPPCCE* q = CheckCC(&pReadPos);
        if( q )
        {
            if( *pReadPos == ')' )
            {
                pReadPos++;
                *ppReadPos = pReadPos;
                return q;
            }
        }
    }
    else
    {
        while( (*pReadPos != '.') && (*pReadPos != ')') && *pReadPos )
            pReadPos++;

        OPPCCE* q = new OPPCCE;
        if( q ) 
        {
            char c = *pReadPos;
            *pReadPos = 0;
            q->m_strVariable = *ppReadPos;
            *pReadPos = c;
            *ppReadPos = pReadPos;
            return q;
        }
    }
    return 0;
}

OPPCCE* OPP::CheckCC( char** ppReadPos )
{
    char* pReadPos = *ppReadPos;
    if( *pReadPos == '~' )
    {
        pReadPos++;

        OPPCCE* pA = CCObject(&pReadPos);
        if( pA )
        {
            if( (pReadPos[0] == '.') && (pReadPos[1] == '~') )
            {
                pReadPos += 2;
                OPPCCE* pB = CCObject(&pReadPos);
                if( pB )
                {
                    OPPCCE* pC = new OPPCCE;
                    if( pC ) 
                    {
                        pC->AddTail(pA);
                        pC->AddTail(pB);
                        *ppReadPos = pReadPos;
                        return pC;
                    }
                }
            }
        }
    }
    return 0;
}

