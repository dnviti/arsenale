import { useMemo } from 'react';
import { cn } from '@/lib/utils';

interface DbResultsTableProps {
  columns: string[];
  rows: Record<string, unknown>[];
  rowCount: number;
  durationMs: number;
  truncated?: boolean;
}

export default function DbResultsTable({
  columns,
  rows,
  rowCount,
  durationMs,
  truncated,
}: DbResultsTableProps) {
  const displayRows = useMemo(() => rows.slice(0, 1000), [rows]);

  if (columns.length === 0 && rows.length === 0) {
    return (
      <div className="p-4 text-center">
        <p className="text-sm text-muted-foreground">
          Query executed successfully. {rowCount} row(s) affected in {durationMs}ms.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 min-h-0 min-w-0 overflow-hidden p-1.5">
      <div className="px-1 py-1 flex justify-between items-center">
        <span className="text-xs text-muted-foreground">
          {rowCount} row(s) returned in {durationMs}ms
          {truncated && ' (results truncated by server limit)'}
          {rows.length > 1000 && ' (showing first 1000)'}
        </span>
      </div>
      <div className="flex-1 overflow-auto min-h-0 rounded border border-border">
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr>
              <th
                className="sticky top-0 left-0 z-[3] bg-primary text-primary-foreground font-bold text-xs tracking-wide border-r border-white/10 py-1.5 px-2 w-12 min-w-[48px]"
              >
                #
              </th>
              {columns.map((col) => (
                <th
                  key={col}
                  className="sticky top-0 z-[2] bg-primary text-primary-foreground font-bold text-xs tracking-wide whitespace-nowrap py-1.5 px-2"
                >
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {displayRows.map((row, idx) => {
              const isOdd = idx % 2 === 1;
              return (
                <tr
                  key={idx}
                  className={cn(
                    'hover:bg-accent/50 transition-colors',
                    isOdd ? 'bg-[#1e1e1e]' : 'bg-card',
                  )}
                >
                  <td
                    className={cn(
                      'sticky left-0 z-[1] border-r border-border text-muted-foreground font-semibold text-xs py-1 px-2',
                      isOdd ? 'bg-[#2a2a2a]' : 'bg-[#252525]',
                    )}
                  >
                    {idx + 1}
                  </td>
                  {columns.map((col) => {
                    const val = row[col];
                    const isNull = val === null || val === undefined;
                    return (
                      <td
                        key={col}
                        className={cn(
                          'whitespace-nowrap max-w-[300px] overflow-hidden text-ellipsis text-[0.8rem] py-1 px-2',
                          isNull && 'text-muted-foreground italic',
                        )}
                      >
                        {formatCellValue(val)}
                      </td>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function formatCellValue(value: unknown): string {
  if (value === null || value === undefined) return 'NULL';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}
