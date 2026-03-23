import type * as monacoNs from 'monaco-editor';
import type { DbTableInfo } from '../../api/database.api';

/**
 * SQL keywords for autocomplete grouped by category.
 */
const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'BETWEEN', 'LIKE', 'IS',
  'NULL', 'AS', 'ON', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'CROSS', 'FULL',
  'ORDER BY', 'GROUP BY', 'HAVING', 'LIMIT', 'OFFSET', 'DISTINCT', 'ALL',
  'INSERT INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE FROM',
  'CREATE TABLE', 'ALTER TABLE', 'DROP TABLE', 'TRUNCATE TABLE',
  'CREATE INDEX', 'DROP INDEX', 'CREATE VIEW', 'DROP VIEW',
  'UNION', 'UNION ALL', 'EXCEPT', 'INTERSECT',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'EXISTS', 'ANY', 'SOME',
  'ASC', 'DESC', 'NULLS FIRST', 'NULLS LAST',
  'WITH', 'RECURSIVE', 'RETURNING',
  'BEGIN', 'COMMIT', 'ROLLBACK', 'SAVEPOINT',
  'GRANT', 'REVOKE',
  'TRUE', 'FALSE', 'DEFAULT',
];

/**
 * Common SQL functions across database dialects.
 */
const SQL_FUNCTIONS: Record<string, string[]> = {
  aggregate: ['COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'ARRAY_AGG', 'STRING_AGG', 'GROUP_CONCAT'],
  string: ['CONCAT', 'SUBSTRING', 'TRIM', 'UPPER', 'LOWER', 'LENGTH', 'REPLACE', 'POSITION', 'LEFT', 'RIGHT', 'LPAD', 'RPAD', 'REVERSE'],
  numeric: ['ABS', 'CEIL', 'FLOOR', 'ROUND', 'MOD', 'POWER', 'SQRT', 'RANDOM'],
  datetime: ['NOW', 'CURRENT_TIMESTAMP', 'CURRENT_DATE', 'CURRENT_TIME', 'DATE', 'EXTRACT', 'DATE_TRUNC', 'AGE', 'INTERVAL'],
  conditional: ['COALESCE', 'NULLIF', 'GREATEST', 'LEAST', 'IF', 'IIF'],
  type: ['CAST', 'CONVERT', 'TO_CHAR', 'TO_NUMBER', 'TO_DATE', 'TO_TIMESTAMP'],
  window: ['ROW_NUMBER', 'RANK', 'DENSE_RANK', 'NTILE', 'LAG', 'LEAD', 'FIRST_VALUE', 'LAST_VALUE', 'OVER', 'PARTITION BY'],
};

/**
 * Regex patterns for detecting context in SQL queries.
 * Determines what kind of completions to offer based on cursor position.
 */
// These patterns match SQL context before the cursor position in small user-typed input
const TABLE_CONTEXT_PATTERN = /\b(?:FROM|JOIN|INTO|UPDATE|TABLE)\s+[\w,.\s]*$/i;
const COLUMN_CONTEXT_PATTERN = /\b(?:SELECT|WHERE|ON|SET|BY|HAVING)\s+[\w,.\s]*$/i;
const SCHEMA_PREFIX_PATTERN = /(\w+)\.$/;

/**
 * Extract table names referenced in the current query for column scoping.
 */
function extractReferencedTables(text: string): string[] {
  const tables: string[] = [];
  // Match table names after FROM, JOIN, UPDATE, INTO
  const patterns = [
    /\bFROM\s+([\w.",\s]+)/gi,
    /\bJOIN\s+([\w."]+)/gi,
    /\bUPDATE\s+([\w."]+)/gi,
    /\bINTO\s+([\w."]+)/gi,
  ];

  for (const pattern of patterns) {
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(text)) !== null) {
      // Split by comma for FROM clauses with multiple tables
      const names = match[1].split(',').map((n) => n.trim().replace(/["']/g, ''));
      for (const name of names) {
        // Remove schema prefix if present, keep the table name
        const parts = name.split('.');
        tables.push(parts[parts.length - 1]);
      }
    }
  }

  return [...new Set(tables)];
}

/**
 * Creates a Monaco CompletionItemProvider for SQL with database-aware suggestions.
 */
export function createSqlCompletionProvider(
  monaco: typeof monacoNs,
  schemaTables: DbTableInfo[],
): monacoNs.languages.CompletionItemProvider {
  return {
    triggerCharacters: ['.', ' '],

    provideCompletionItems(
      model: monacoNs.editor.ITextModel,
      position: monacoNs.Position,
    ): monacoNs.languages.ProviderResult<monacoNs.languages.CompletionList> {
      const textUntilPosition = model.getValueInRange({
        startLineNumber: 1,
        startColumn: 1,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      });

      const word = model.getWordUntilPosition(position);
      const range: monacoNs.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      const suggestions: monacoNs.languages.CompletionItem[] = [];

      // Check if we're after a schema prefix (e.g., "public.")
      const schemaMatch = SCHEMA_PREFIX_PATTERN.exec(textUntilPosition);
      if (schemaMatch) {
        const schemaName = schemaMatch[1];
        const tablesInSchema = schemaTables.filter(
          (t) => t.schema.toLowerCase() === schemaName.toLowerCase(),
        );
        for (const table of tablesInSchema) {
          suggestions.push({
            label: table.name,
            kind: monaco.languages.CompletionItemKind.Class,
            detail: `Table (${table.schema})`,
            documentation: `Columns: ${table.columns.map((c) => c.name).join(', ')}`,
            insertText: table.name,
            range,
          });
        }
        return { suggestions };
      }

      // Check if we're in a table context (after FROM, JOIN, INTO, etc.)
      const isTableContext = TABLE_CONTEXT_PATTERN.test(textUntilPosition);
      if (isTableContext) {
        for (const table of schemaTables) {
          const qualifiedName = table.schema === 'public' ? table.name : `${table.schema}.${table.name}`;
          suggestions.push({
            label: qualifiedName,
            kind: monaco.languages.CompletionItemKind.Class,
            detail: `Table (${table.schema}) - ${table.columns.length} columns`,
            documentation: `Columns: ${table.columns.map((c) => `${c.name} (${c.dataType}${c.isPrimaryKey ? ', PK' : ''})`).join(', ')}`,
            insertText: qualifiedName,
            range,
          });
        }
      }

      // Check if we're in a column context (after SELECT, WHERE, ON, etc.)
      const isColumnContext = COLUMN_CONTEXT_PATTERN.test(textUntilPosition);
      if (isColumnContext) {
        const fullText = model.getValue();
        const referencedTables = extractReferencedTables(fullText);

        // If tables are referenced, scope column suggestions to those tables
        const relevantTables = referencedTables.length > 0
          ? schemaTables.filter((t) =>
            referencedTables.some((ref) => ref.toLowerCase() === t.name.toLowerCase()),
          )
          : schemaTables;

        for (const table of relevantTables) {
          for (const column of table.columns) {
            suggestions.push({
              label: column.name,
              kind: column.isPrimaryKey
                ? monaco.languages.CompletionItemKind.Field
                : monaco.languages.CompletionItemKind.Property,
              detail: `${column.dataType}${column.isPrimaryKey ? ' (PK)' : ''}${column.nullable ? ' nullable' : ''} - ${table.name}`,
              documentation: `Column from ${table.schema}.${table.name}`,
              insertText: column.name,
              range,
              sortText: `0_${column.name}`, // Columns first
            });
          }
        }
      }

      // Always add SQL keywords
      for (const keyword of SQL_KEYWORDS) {
        suggestions.push({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: keyword,
          range,
          sortText: `2_${keyword}`, // Keywords after columns and tables
        });
      }

      // Always add SQL functions
      for (const [category, funcs] of Object.entries(SQL_FUNCTIONS)) {
        for (const func of funcs) {
          suggestions.push({
            label: func,
            kind: monaco.languages.CompletionItemKind.Function,
            detail: `${category} function`,
            insertText: `${func}($0)`,
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
            sortText: `1_${func}`, // Functions between columns and keywords
          });
        }
      }

      // Add table names as general suggestions when not in specific context
      if (!isTableContext && !isColumnContext) {
        for (const table of schemaTables) {
          const qualifiedName = table.schema === 'public' ? table.name : `${table.schema}.${table.name}`;
          suggestions.push({
            label: qualifiedName,
            kind: monaco.languages.CompletionItemKind.Class,
            detail: `Table (${table.schema})`,
            insertText: qualifiedName,
            range,
            sortText: `1_${qualifiedName}`,
          });
        }
      }

      return { suggestions };
    },
  };
}
