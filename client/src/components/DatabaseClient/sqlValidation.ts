import type * as monacoNs from 'monaco-editor';

interface ValidationMarker {
  message: string;
  severity: 'error' | 'warning' | 'info';
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
}

/**
 * Check for unclosed single-quoted strings.
 */
function checkUnclosedQuotes(text: string, lines: string[]): ValidationMarker[] {
  const markers: ValidationMarker[] = [];
  let inSingleQuote = false;
  let quoteStartLine = 0;
  let quoteStartCol = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    for (let j = 0; j < line.length; j++) {
      if (line[j] === "'" && (j === 0 || line[j - 1] !== "'")) {
        // Handle escaped quotes ('') by checking next char
        if (inSingleQuote && j + 1 < line.length && line[j + 1] === "'") {
          j++; // Skip escaped quote
          continue;
        }
        if (!inSingleQuote) {
          inSingleQuote = true;
          quoteStartLine = i;
          quoteStartCol = j;
        } else {
          inSingleQuote = false;
        }
      }
    }
  }

  if (inSingleQuote) {
    markers.push({
      message: 'Unclosed string literal',
      severity: 'error',
      startLineNumber: quoteStartLine + 1,
      startColumn: quoteStartCol + 1,
      endLineNumber: quoteStartLine + 1,
      endColumn: quoteStartCol + 2,
    });
  }

  // Suppress unused var
  void text;

  return markers;
}

/**
 * Check for mismatched parentheses.
 */
function checkMismatchedParentheses(lines: string[]): ValidationMarker[] {
  const markers: ValidationMarker[] = [];
  const stack: { line: number; col: number }[] = [];
  let inString = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    for (let j = 0; j < line.length; j++) {
      // Skip string contents
      if (line[j] === "'" && (j === 0 || line[j - 1] !== "'")) {
        inString = !inString;
        continue;
      }
      if (inString) continue;

      // Skip line comments
      if (line[j] === '-' && j + 1 < line.length && line[j + 1] === '-') break;

      if (line[j] === '(') {
        stack.push({ line: i, col: j });
      } else if (line[j] === ')') {
        if (stack.length === 0) {
          markers.push({
            message: 'Unmatched closing parenthesis',
            severity: 'error',
            startLineNumber: i + 1,
            startColumn: j + 1,
            endLineNumber: i + 1,
            endColumn: j + 2,
          });
        } else {
          stack.pop();
        }
      }
    }
  }

  for (const open of stack) {
    markers.push({
      message: 'Unmatched opening parenthesis',
      severity: 'error',
      startLineNumber: open.line + 1,
      startColumn: open.col + 1,
      endLineNumber: open.line + 1,
      endColumn: open.col + 2,
    });
  }

  return markers;
}

/**
 * Check for common SQL mistakes.
 */
function checkCommonMistakes(lines: string[]): ValidationMarker[] {
  const markers: ValidationMarker[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    // Skip comments
    if (trimmed.startsWith('--')) continue;

    // Warn about SELECT * in multi-statement queries
    if (/\bSELECT\s+\*/i.test(trimmed)) {
      const match = trimmed.match(/\bSELECT\s+\*/i);
      if (match && match.index !== undefined) {
        const col = line.indexOf(match[0]);
        markers.push({
          message: 'Consider specifying explicit column names instead of SELECT *',
          severity: 'info',
          startLineNumber: i + 1,
          startColumn: col + 1,
          endLineNumber: i + 1,
          endColumn: col + match[0].length + 1,
        });
      }
    }

    // Warn about DELETE without WHERE
    if (/\bDELETE\s+FROM\b/i.test(trimmed) && !/\bWHERE\b/i.test(trimmed)) {
      // Check if WHERE is on subsequent lines before next semicolon
      let hasWhere = false;
      for (let k = i + 1; k < lines.length; k++) {
        if (/\bWHERE\b/i.test(lines[k])) { hasWhere = true; break; }
        if (lines[k].includes(';')) break;
      }
      if (!hasWhere) {
        const col = line.search(/\bDELETE/i);
        markers.push({
          message: 'DELETE without WHERE clause will remove all rows',
          severity: 'warning',
          startLineNumber: i + 1,
          startColumn: col + 1,
          endLineNumber: i + 1,
          endColumn: col + 12,
        });
      }
    }

    // Warn about UPDATE without WHERE
    if (/\bUPDATE\b/i.test(trimmed) && /\bSET\b/i.test(trimmed) && !/\bWHERE\b/i.test(trimmed)) {
      let hasWhere = false;
      for (let k = i + 1; k < lines.length; k++) {
        if (/\bWHERE\b/i.test(lines[k])) { hasWhere = true; break; }
        if (lines[k].includes(';')) break;
      }
      if (!hasWhere) {
        const col = line.search(/\bUPDATE\b/i);
        markers.push({
          message: 'UPDATE without WHERE clause will modify all rows',
          severity: 'warning',
          startLineNumber: i + 1,
          startColumn: col + 1,
          endLineNumber: i + 1,
          endColumn: col + 7,
        });
      }
    }
  }

  return markers;
}

/**
 * Validate SQL content and set Monaco editor markers.
 * Call this on debounced content change.
 */
export function validateSql(
  monaco: typeof monacoNs,
  model: monacoNs.editor.ITextModel,
): void {
  const text = model.getValue();
  if (!text.trim()) {
    monaco.editor.setModelMarkers(model, 'sql-validation', []);
    return;
  }

  const lines = text.split('\n');
  const markers: ValidationMarker[] = [
    ...checkUnclosedQuotes(text, lines),
    ...checkMismatchedParentheses(lines),
    ...checkCommonMistakes(lines),
  ];

  const monacoMarkers: monacoNs.editor.IMarkerData[] = markers.map((m) => ({
    message: m.message,
    severity:
      m.severity === 'error'
        ? monaco.MarkerSeverity.Error
        : m.severity === 'warning'
          ? monaco.MarkerSeverity.Warning
          : monaco.MarkerSeverity.Info,
    startLineNumber: m.startLineNumber,
    startColumn: m.startColumn,
    endLineNumber: m.endLineNumber,
    endColumn: m.endColumn,
  }));

  monaco.editor.setModelMarkers(model, 'sql-validation', monacoMarkers);
}
