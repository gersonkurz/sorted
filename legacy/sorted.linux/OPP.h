#ifndef OPP_H
#define OPP_H

#include "gtools.h"

#ifdef _WIN32 
#define OPEN_FILE_FOR_READING "rt"
#define OPEN_FILE_FOR_WRITING "wt"
#define NEWLINE_STRING        "\r\n"
#else
#define OPEN_FILE_FOR_READING "r"
#define OPEN_FILE_FOR_WRITING "w"
#define NEWLINE_STRING        "\n"
#endif

#define MAX_COLUMNS_PER_LINE    10240
#define MAX_CC_NEST_DEPTH       27

class OPPCCE : public PList
    {
    public:
        bool Evaluate();

        PString m_strVariable;
    };

class OPPM : public PNode
    {
    public:
        OPPM( const char* pszName, const char* pszDefinition );
        bool Fits(char** qq,PList* pArguments);
        bool Parse(char** pszWritePos,PList* pArguments);

        PString m_strName, __D;
    };

class OPP
    {
    public:
        OPP();
        virtual ~OPP();
        bool ProcessFile( const char* pszSource, const char* pszTarget );

    protected:
        bool IProcessFile( const char* pszSource );
        bool ProcessSingleLine();
        const char* ExtractIncludeFileName( const char* p );
        bool AddMacroDefinition( char** pp );
        OPPCCE* CheckCC(char** ppReadPos);
        OPPCCE* CCObject(char** ppReadPos);
        bool IsAMacro(char** pszWritePos,char** qq);

        PList m_Macros;
        FILE* m_fpTarget;
        char* m_szInput;
        char* m_szOutput;
        int m_nLineNumber, m_nCurlyBracketsClose, m_nCurlyBracketsOpen;
        int m_iSkipIndex;
    };

#endif // OPP_H
