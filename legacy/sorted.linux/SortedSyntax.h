#ifndef SortedSyntax_H
#define SortedSyntax_H

#define SLID_TYPE_NUMBER        ((long)0)
#define SLID_TYPE_CELL          SLID_TYPE_NUMBER

#define SLID_TYPE_SUM           ((long)1)
#define SLID_TYPE_DIFF          ((long)2)
#define SLID_TYPE_PROD          ((long)3)
#define SLID_TYPE_RATIO         ((long)4)
#define SLID_TYPE_NAND          ((long)5)
#define SLID_TYPE_ASSIGN        ((long)6)
#define SLID_TYPE_WRITE         ((long)7)
#define SLID_TYPE_READ          ((long)8)
#define SLID_TYPE_CONDITION     ((long)9)
#define SLID_TYPE_JUMP          ((long)10)
#define SLID_TYPE_LABEL         ((long)11)

#define SLID_TYPE_INDIRECT      0xF00000L

#define SLIDE_INFO_SUMS         0
#define SLIDE_INFO_DIFFS        1
#define SLIDE_INFO_PRODS        2
#define SLIDE_INFO_RATIOS       3
#define SLIDE_INFO_NANDS        4
#define SLIDE_INFO_ASSIGNS      5
#define SLIDE_INFO_WRITES       6
#define SLIDE_INFO_READS        7
#define SLIDE_INFO_CONDITIONS   8
#define SLIDE_INFO_STATEMENTS   9
#define SLIDE_INFO_JUMPS        10
#define SLIDE_INFO_MAX          11

#define MAX_INSTRUCTIONS_PER_PROGRAM    128192
#define MAX_DATA_PER_PROGRAM            193719

typedef struct slid
{
    long Type,Index;
} SLID, *PSLID;

typedef struct slide
{
    SLID _[2];
    long Flags;
} SLIDE, *PSLIDE;

typedef struct
{
    int Count, Index;
} SLIDE_INFO;

typedef struct
{
    SLIDE_INFO Type[32];
    int nTypeCount;
    int nNumbersUsed;
    int nLabelsCount;
    SLIDE Code[MAX_INSTRUCTIONS_PER_PROGRAM];
    long Data[MAX_DATA_PER_PROGRAM];
} CODEINFO;

#define SLIDE_FORMAT_CHARACTER              ((long)0)
#define SLIDE_FORMAT_ENGLISH_CARDINAL       ((long)1)
#define SLIDE_FORMAT_ENGLISH_ORDINAL        ((long)2)
#define SLIDE_FORMAT_GERMAN_CARDINAL        ((long)3)
#define SLIDE_FORMAT_GERMAN_ORDINAL         ((long)4)

#define SLIDE_COMPARE_EQUAL         ((long)0)
#define SLIDE_COMPARE_LESS          ((long)1)

#define SLIDE_UNCONDITIONAL_JUMP    ((long)0)
#define SLIDE_CONDITIONAL_JUMP      ((long)1)

#define isChar(c)           ((c>='A'&&c<='Z')||(c>='a'&&c<='z'))
#define pushExpression()    char* backup = m_pszExpression
#define popExpression()     m_pszExpression = backup
#define isValidChar(c)      (isChar(c)||(c=='.')||(c==','))
#define isWhitespace()      (*m_pszExpression && !isValidChar(*m_pszExpression))
#define skipWhitespaces()   while( isWhitespace() ) m_pszExpression++;

class SortedSyntax
    {
    public:
        bool Parse( CODEINFO* pCode, char* pszExpression );

    protected:
        CODEINFO* m_pCode;

        bool NUMBER_DECLARATION();
        bool USES_NO_NUMBERS();
        bool USES_NUMBERS();
        bool SINGLE_NUMBER();        
        bool NUMBER_SEQUENCE();
        bool IS_KEYWORD(const char* pszKeyword);
        bool IS_KEYWORD_SEQUENCE(const char** pszKeyword);
        bool CARDINAL(int& n);
        bool SUM_DECLARATION();
        bool USES_NO_SUMS();
        bool USES_SUMS();
        bool SINGLE_SUM();
        bool SUM_SEQUENCE();
        bool SUM_SPEC();
        bool DIFF_DECLARATION();
        bool USES_NO_DIFFS();
        bool USES_DIFFS();
        bool SINGLE_DIFF();
        bool DIFF_SEQUENCE();
        bool DIFF_SPEC();
        bool PROD_DECLARATION();
        bool USES_NO_PRODS();
        bool USES_PRODS();
        bool SINGLE_PROD();
        bool PROD_SEQUENCE();
        bool PROD_SPEC();
        bool RATIO_DECLARATION();
        bool USES_NO_RATIOS();
        bool USES_RATIOS();
        bool SINGLE_RATIO();
        bool RATIO_SEQUENCE();
        bool RATIO_SPEC();
        bool NAND_DECLARATION();
        bool USES_NO_NANDS();
        bool USES_NANDS();
        bool SINGLE_NAND();
        bool NAND_SEQUENCE();
        bool NAND_SPEC();
        bool IDENTIFIER( PSLID ps );
        bool DIRECT_USE( PSLID ps );
        bool INDIRECT_USE( PSLID ps );
        bool COOL();
        bool LABEL_DECLARATION();
        bool USES_NO_LABELS();
        bool USES_LABELS();
        bool ASSIGN_DECLARATION();
        bool USES_NO_ASSIGNS();
        bool USES_ASSIGNS();
        bool SINGLE_ASSIGN();
        bool ASSIGN_SEQUENCE();
        bool ASSIGN_SPEC();
        bool OUTPUT_DECLARATION();
        bool USES_NO_OUTPUT();
        bool USES_OUTPUT();
        bool SINGLE_OUTPUT();
        bool OUTPUT_SEQUENCE();
        bool OUTPUT_SPEC();
        bool INPUT_DECLARATION();
        bool USES_NO_INPUT();
        bool USES_INPUT();
        bool SINGLE_INPUT();
        bool INPUT_SEQUENCE();
        bool INPUT_SPEC();
        bool CONDITION_DECLARATION();
        bool USES_NO_CONDITIONS();
        bool USES_CONDITIONS();
        bool SINGLE_CONDITION();
        bool CONDITION_SEQUENCE();
        bool CONDITION_SPEC();
        bool COMPARISON( PSLIDE ps );
        bool IMPLEMENTATION_DECLARATION();
        bool SINGLE_STATEMENT();
        bool STATEMENT_SEQUENCE();
        bool STATEMENT_SPEC();
        bool JUMP_DECLARATION();
        bool USES_NO_JUMPS();
        bool USES_JUMPS();
        bool SINGLE_JUMP();
        bool JUMP_SEQUENCE();
        bool JUMP_SPEC();
        bool UNCONDITIONAL_JUMP();
        bool CONDITIONAL_JUMP();

        bool StoreSingleNumber( int n );
        void PrepareCodeInfo( int nType );
        void CancelSpec( int n );

        char* m_pszExpression;
    };

#endif 

