const MAX_CSV_LINE_LENGTH = 1_048_576; // 1 MB per line
const MAX_CSV_LINES = 100_000;

export function parseCSV(csv: string): { headers: string[]; rows: string[][] } {
  const lines = csv.split(/\r?\n/).filter(line => line.trim() !== '');
  if (lines.length === 0) {
    return { headers: [], rows: [] };
  }
  if (lines.length > MAX_CSV_LINES) {
    throw new Error(`CSV exceeds maximum of ${MAX_CSV_LINES} lines`);
  }

  const parseLine = (line: string): string[] => {
    if (line.length > MAX_CSV_LINE_LENGTH) {
      throw new Error(`CSV line exceeds maximum length of ${MAX_CSV_LINE_LENGTH} characters`);
    }
    const result: string[] = [];
    let current = '';
    let inQuotes = false;
    let i = 0;

    while (i < line.length) {
      const char = line[i];

      if (inQuotes) {
        if (char === '"') {
          if (i + 1 < line.length && line[i + 1] === '"') {
            current += '"';
            i += 2;
          } else {
            inQuotes = false;
            i++;
          }
        } else {
          current += char;
          i++;
        }
      } else {
        if (char === '"') {
          inQuotes = true;
          i++;
        } else if (char === ',') {
          result.push(current);
          current = '';
          i++;
        } else {
          current += char;
          i++;
        }
      }
    }

    result.push(current);
    return result;
  };

  const headers = parseLine(lines[0]);
  const rows = lines.slice(1).map(parseLine);

  return { headers, rows };
}

export function generateCSV(headers: string[], rows: string[][]): string {
  const escapeValue = (value: unknown): string => {
    if (value === null || value === undefined) {
      return '';
    }
    const str = String(value);
    if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
      return `"${str.replace(/"/g, '""')}"`;
    }
    return str;
  };

  const headerLine = headers.map(escapeValue).join(',');
  const dataLines = rows.map(row => row.map(escapeValue).join(','));

  return [headerLine, ...dataLines].join('\n');
}

export function escapeCSV(value: string): string {
  if (value.includes(',') || value.includes('"') || value.includes('\n') || value.includes('\r')) {
    return `"${value.replace(/"/g, '""')}"`;
  }
  return value;
}

export function unescapeCSV(value: string): string {
  if (value.startsWith('"') && value.endsWith('"')) {
    return value.slice(1, -1).replace(/""/g, '"');
  }
  return value;
}
