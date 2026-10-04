#include "precomp.h"
#include "SortedSyntax.h"
#include "EnglishNumbers.h"
#include "GermanNumbers.h"
#include "ctype.h"
#include "OPP.h"

bool ReadSortedSourcecode( CODEINFO* pCI, char* filename )
{
    PString strFilename(0,"%s.opp",filename);
    char* pExt = strrchr((char*)strFilename,'.');
    if(pExt)
    {
        *pExt = ',';

        OPP opp;
        if( !opp.ProcessFile(filename,(char*)strFilename) )
            return false;

        filename = strFilename;
    }

    FILE* fp = fopen(filename,"rt");
    if(fp)
    {
        char* buffer = (char*) calloc(1,291793), *write_pos,
            *read_pos, *line = (char*) calloc(1,1024);

        write_pos = buffer;

        while( !feof(fp) && fgets(line,1024,fp) )
        {
            read_pos = line;
            while( *read_pos )           
            {
                if( isValidChar(*read_pos) )
                    *(write_pos++) = *read_pos;
                else
                    *(write_pos++) = ' ';

                read_pos++;
            }
            *write_pos = 0;
        }
        fclose(fp);

        SortedSyntax syntax;

        bool success = syntax.Parse(pCI,buffer);
        free(buffer);
        return success;
    }
    return false;
}

void DumpCodeInfo( CODEINFO* pCI, char* szFilename )
{
    FILE* fp = fopen(szFilename,"wt");
    if(fp)
    {
        int i, j;

        fprintf( fp, "ELEMENTS=%d\n", pCI->nTypeCount );
        fprintf( fp, "LABELS=%d\n", pCI->nLabelsCount );

        fprintf( fp, "%d NUMBERS: ", pCI->nNumbersUsed );
        for( i = 0; i < pCI->nNumbersUsed; i++ )
        {
            if( i )
                fprintf(fp,",");

            fprintf(fp,"%d",pCI->Data[i] );
        }

        static char* SlideTypes[] = { "SUMS", "DIFFS", "PRODS", "RATIOS", "NANDS", "ASSIGNS", "WRITES", "READS", "CONDITIONS", "STATEMENTS", "JUMPS" };
        static int SlideUsage[] =   { 2     , 2      , 2      , 2       , 2      , 2        , 1       , 1      , 2           , 1           , 2 };
        static char* ObjectTypes[] = { "CELL","SUM","DIFF","PROD","RATIO","NAND","ASSIGN","WRITE","READ","COMPARE","JUMP","LABEL" };
        

        for( i = 0; i < SLIDE_INFO_MAX; i++ )
        {
            SLIDE_INFO* psi = &(pCI->Type[i]);
            SLIDE* ps = &(pCI->Code[psi->Index]);

            fprintf( fp, "\n%d %s: ", psi->Count, SlideTypes[i] );

            for( j = 0; j < psi->Count; j++ )
            {
                if( j )
                    fprintf(fp,", ");

                fprintf(fp,"{ " );

                if( ps->_[0].Type & SLID_TYPE_INDIRECT )
                {
                    fprintf( fp, "*(%d,%s)", ps->_[0].Index, ObjectTypes[ps->_[0].Type & 0xFF] );
                }
                else
                {
                    fprintf( fp, "(%d,%s)", ps->_[0].Index, ObjectTypes[ps->_[0].Type] );
                }
                
                if( SlideUsage[i] == 2 )
                {
                    if( ps->_[1].Type & SLID_TYPE_INDIRECT )
                    {
                        fprintf( fp, ",*(%d,%s)", ps->_[1].Index, ObjectTypes[ps->_[1].Type & 0xFF] );
                    }
                    else
                    {
                        fprintf( fp, ",(%d,%s)", ps->_[1].Index, ObjectTypes[ps->_[1].Type] );
                    }
                }
                fprintf( fp, ",%d}", ps->Flags );
                ps++;
            }
        }

        fprintf(fp,"\n");
        fclose(fp);
    }
}
                
char* SlideType( SLID* ps, char* lpszBuffer, long Flags, bool bClosingBracket = true )
{
    static char* ObjectTypes[] ={ "N", "S", "D", "P", "R", "L", "A", "W", "r", "C", "J", "L" };

    if( ps->Type & SLID_TYPE_INDIRECT )
    {
        sprintf(lpszBuffer, bClosingBracket ? "N(%s(%d))" : "N(%s(%d)", ObjectTypes[ps->Type & 0xFF], ps->Index+1 );
    }
    else
    {
        sprintf(lpszBuffer, bClosingBracket ? "%s(%d)" : "%s(%d", ObjectTypes[ps->Type], ps->Index+1 );
    }
    return lpszBuffer;
}

void GenerateSourceInC( CODEINFO* pCI, char* szFilename )
{
    FILE* fp = fopen(szFilename,"wt");
    if(fp)
    {
        int i, j;

        fprintf( fp, "#include \"stdio.h\"\n" );
        fprintf( fp, "#include \"stdlib.h\"\n" );
        fprintf( fp, "#include \"string.h\"\n\n" );

        fprintf( fp, "long _[%d]", sizeof(pCI->Data)/sizeof(pCI->Data[0]) );
        if( !pCI->nNumbersUsed )
        {
            fprintf( fp, ";\n" );
        }
        else
        {
            fprintf( fp, " = { " );
            for( i = 0; i < pCI->nNumbersUsed; i++ )
            {
                if( i )
                    fprintf(fp,",");

                fprintf(fp,"%d",pCI->Data[i] );
            }
            fprintf( fp, "};\n\n");
        }
        fprintf( fp, "#define N(x) _[x-1]\n" );
        fprintf( fp, "#define G(x) long x\n" );
        fprintf( fp, "#define B(x,y) case x: return y;\n" );
        fprintf( fp, "#define d(x) G(x)(G(n))\n" );
        fprintf( fp, "#define E(x) d(x){switch(n){\n" );
        fprintf( fp, "#define L(n) label##n:\n" );
        fprintf( fp, "#define UJ(n) goto label##n\n" );
        fprintf( fp, "#define CJ(n,c) if(C(c)) UJ(n)\n\n" );
        fprintf( fp, "#define H(x) putchar(x)\n" );
        fprintf( fp, "#define i(x) x = getchar()\n" );
        fprintf( fp, "#define F } return 0; }\n" );
        fprintf( fp, "\n" );

        SLIDE_INFO* psi;
        char szTemp1[80], szTemp2[80];

        static char* SlideTypes[] = { "S", "D", "P", "R", "L", "A", "W", "R", "C" };
        static char* SlideTypeFormat[] = { "%s+%s", "%s-%s", "%s*%s", "%s/%s", "~(%s) & ~(%s)", "%s=%s", "H(%s)", "i(%s)", "C", "L", "G" };

        int q = 0;

        for( i = 0; i <= SLIDE_INFO_CONDITIONS; i++ )
        {
            psi = &(pCI->Type[i]);
            if( !psi->Count )
                continue;
            
            fprintf( fp, "d(%s);", SlideTypes[i] );

        }
        fprintf( fp, "\n" );

        for( i = 0; i <= SLIDE_INFO_CONDITIONS; i++ )
        {
            psi = &(pCI->Type[i]);
            if( !psi->Count )
                continue;
            
            fprintf( fp, "E(%s)", SlideTypes[i] );

            if( i == SLIDE_INFO_CONDITIONS )
            {
                SLIDE* ps = &(pCI->Code[psi->Index]);
                for( j = 0; j < psi->Count; j++ )
                {
                    const char* lpszFormat = (ps->Flags == SLIDE_COMPARE_EQUAL) ? "%s==%s" : "%s<%s";
                    fprintf( fp, "B(%d,", j+1);
                    fprintf( fp, lpszFormat, SlideType( &(ps->_[0]), szTemp1, ps->Flags ), SlideType( &(ps->_[1]), szTemp2, ps->Flags ) );
                    fprintf( fp, ") " );
                    ps++;
                    if( !(++q % 4) )
                        fprintf(fp,"\n");
                }
            }
            else
            {
                SLIDE* ps = &(pCI->Code[psi->Index]);
                for( j = 0; j < psi->Count; j++ )
                {
                    fprintf( fp, "B(%d,", j+1);
                    if( i == SLIDE_INFO_ASSIGNS )
                        fprintf( fp, SlideTypeFormat[i], SlideType( &(ps->_[1]), szTemp1, ps->Flags ), SlideType( &(ps->_[0]), szTemp2, ps->Flags ) );
                    else
                        fprintf( fp, SlideTypeFormat[i], SlideType( &(ps->_[0]), szTemp1, ps->Flags ), SlideType( &(ps->_[1]), szTemp2, ps->Flags ) );
                    fprintf( fp, ")" );
                    ps++;
                    if( !(++q % 4) )
                        fprintf(fp,"\n");
                }
            }

            fprintf( fp, "F " );
        }

        fprintf( fp, "int main(int,char*[]) {" );

        psi = &(pCI->Type[SLIDE_INFO_STATEMENTS]);
        SLIDE_INFO* psiJ = &(pCI->Type[SLIDE_INFO_JUMPS]);
        SLIDE* ps = &(pCI->Code[psi->Index]);
        SLIDE* psJ = &(pCI->Code[psiJ->Index]);
        for( j = 0; j < psi->Count; j++ )
        {
            if( ps->_[0].Type == SLID_TYPE_JUMP )
            {
                if( psJ[ps->_[0].Index].Flags == SLIDE_UNCONDITIONAL_JUMP )
                {
                    fprintf( fp, "UJ(%d);", psJ[ps->_[0].Index]._[0].Index+1 );
                }
                else
                {
                    fprintf( fp, "CJ(%d,%d);", psJ[ps->_[0].Index]._[0].Index+1, psJ[ps->_[0].Index]._[1].Index+1 );
                }
            }
            else
            {
                fprintf( fp, "%s);", SlideType( &(ps->_[0]), szTemp1, ps->Flags, false ) );
            }
            if( !(++q % 4) )
                fprintf(fp,"\n");
            ps++;
        }
        fprintf( fp, "{ F\n" );
        fclose(fp);
    }
}       

long* GetDataPointer( CODEINFO* pci, SLID* ps )
{
    if( ps->Type & SLID_TYPE_INDIRECT )
    {
        return 0;
    }
    else
    {
        return &(pci->Data[ps->Index]);
    }
}

class SortedInterpreter 
    {
    public:
        void Interpret( CODEINFO* pCI );

    protected:
        long* GetDataPointer( SLID* ps );
        long GetDataValue( SLID* ps );

        int m_iPC;
        int* m_pKnownLabel;
        CODEINFO* m_pCI;
    };

long SortedInterpreter::GetDataValue( SLID* ps )
{
    long Result = 0;
    SLIDE_INFO* psi;
    SLIDE* psA;
    
    switch( ps->Type & 0xFF )
    {
    case SLID_TYPE_NUMBER:
        Result = m_pCI->Data[ps->Index];
        break;
    case SLID_TYPE_SUM:
        psi = &(m_pCI->Type[ps->Type-1]);
        psA = &(m_pCI->Code[psi->Index + ps->Index]);
        Result = GetDataValue(&psA->_[0])+GetDataValue(&psA->_[1]);
        break;
    case SLID_TYPE_DIFF:
        psi = &(m_pCI->Type[ps->Type-1]);
        psA = &(m_pCI->Code[psi->Index + ps->Index]);
        Result = GetDataValue(&psA->_[0])-GetDataValue(&psA->_[1]);
        break;
    case SLID_TYPE_PROD:
        psi = &(m_pCI->Type[ps->Type-1]);
        psA = &(m_pCI->Code[psi->Index + ps->Index]);
        Result = GetDataValue(&psA->_[0])*GetDataValue(&psA->_[1]);
        break;
    case SLID_TYPE_RATIO:
        psi = &(m_pCI->Type[ps->Type-1]);
        psA = &(m_pCI->Code[psi->Index + ps->Index]);
        Result = GetDataValue(&psA->_[0])/GetDataValue(&psA->_[1]);
        break;
    case SLID_TYPE_NAND:
        psi = &(m_pCI->Type[ps->Type-1]);
        psA = &(m_pCI->Code[psi->Index + ps->Index]);
        Result = ~GetDataValue(&psA->_[0]) & ~GetDataValue(&psA->_[1]);
        break;
    case SLID_TYPE_CONDITION:
        psi = &(m_pCI->Type[ps->Type-1]);
        psA = &(m_pCI->Code[psi->Index + ps->Index]);
        if( psA->Flags == SLIDE_COMPARE_EQUAL )
            Result = GetDataValue(&psA->_[0]) == GetDataValue(&psA->_[1]);
        else
            Result = GetDataValue(&psA->_[0]) < GetDataValue(&psA->_[1]);
        break;
    case SLID_TYPE_LABEL:
        Result = m_pKnownLabel[ps->Index];
        break;
    }
    if( ps->Type & SLID_TYPE_INDIRECT )
        Result = m_pCI->Data[Result-1];

    return Result;
}

long* SortedInterpreter::GetDataPointer( SLID* ps )
{
    long* lpResult = 0;
    switch( ps->Type & 0xFF )
    {
    case SLID_TYPE_NUMBER:
        lpResult = &(m_pCI->Data[ps->Index]);
        break;
    }
    if( lpResult && (ps->Type & SLID_TYPE_INDIRECT) )
        lpResult = &(m_pCI->Data[*lpResult]);

    return lpResult;
}

void SortedInterpreter::Interpret( CODEINFO* pCI )
{
    int i,j;

    m_pCI = pCI;
    m_pKnownLabel = new int[pCI->nLabelsCount+2];

    SLIDE_INFO* psi = &(pCI->Type[SLIDE_INFO_STATEMENTS]), *psiA;
    SLIDE* ps = &(pCI->Code[psi->Index]), *psA;

    for( i = 0; i < pCI->nLabelsCount; m_pKnownLabel[i++] = 0 ) {}

    for( j = 0; j < psi->Count; j++, ps++ )
        if( ps->_[0].Type == SLID_TYPE_LABEL )
            m_pKnownLabel[ps->_[0].Index] = j;
    
    psi = &(pCI->Type[SLIDE_INFO_STATEMENTS]);
    for( m_iPC = 0; m_iPC < psi->Count; m_iPC++ )
    {
        ps = &(pCI->Code[psi->Index+m_iPC]);
        psiA = &(m_pCI->Type[ps->_[0].Type-1]);
        psA = &(pCI->Code[psiA->Index + ps->_[0].Index]);
        switch( ps->_[0].Type )
        {
        case SLID_TYPE_ASSIGN:
            *GetDataPointer(&psA->_[1]) = GetDataValue(&psA->_[0]);
            break;
        case SLID_TYPE_READ:
            *GetDataPointer(&psA->_[1]) = getchar();
            break;
        case SLID_TYPE_WRITE:
            switch(psA->Flags)
            {
            case SLIDE_FORMAT_ENGLISH_CARDINAL:
                printf( "%s\n", IntegerToEnglishCardinal( (int) GetDataValue(&psA->_[0]) ));
                break;

            case SLIDE_FORMAT_ENGLISH_ORDINAL:
                printf( "%s\n", IntegerToEnglishOrdinal( (int) GetDataValue(&psA->_[0]) ));
                break;

            case SLIDE_FORMAT_GERMAN_CARDINAL:
                printf( "%s\n", IntegerToGermanCardinal( (int) GetDataValue(&psA->_[0]) ));
                break;

            case SLIDE_FORMAT_GERMAN_ORDINAL:
                printf( "%s\n", IntegerToGermanOrdinal( (int) GetDataValue(&psA->_[0]) ));
                break;

            default:
                putchar( (int) GetDataValue(&psA->_[0]) );
            }
            
            break;
        case SLID_TYPE_JUMP:
            psiA = &(m_pCI->Type[SLIDE_INFO_JUMPS]);
            psA = &(pCI->Code[psiA->Index + ps->_[0].Index]);
            if( psA->Flags == SLIDE_UNCONDITIONAL_JUMP )
                m_iPC = GetDataValue(&psA->_[0]);
            else if( GetDataValue(&psA->_[1]) )
                m_iPC = GetDataValue(&psA->_[0]);
            break;
        }
    }
    delete [] m_pKnownLabel;
}

int main( int argc, char* argv[] )
{
    char* pCreateCSource = 0;
    char* pDumpInstructions = 0;
    char* pSourcecode = 0;

    for(int i=1;i<argc;i++ )
    {
        char* a = argv[i];
        if( (*a == '-') || (*a == '/') )
        {
            switch(tolower(a[1]))
            {
            case 's':
                pSourcecode = a+2;
                break;
            case 'c':
                pCreateCSource = a+2;
                break;
            case 'd':
                pDumpInstructions = a+2;
                break;
            }
        }
    }
    if( pSourcecode )
    {
        static CODEINFO ci;
        memset( &ci, 0, sizeof(ci) );

        if( ReadSortedSourcecode( &ci, pSourcecode ) )
        {
            if( pCreateCSource )
                GenerateSourceInC(&ci, pCreateCSource );

            if( pDumpInstructions )
                DumpCodeInfo( &ci, pDumpInstructions );

            SortedInterpreter si;
            si.Interpret(&ci);    
        }
        else printf( "%s is not intelligible.\n", pSourcecode );
    }
    else printf( "USAGE: Sorted /S<Sourcecode> [/D<Dump>] [/C<C-Sourcecode>]\n" );
    return 0;
    
}




