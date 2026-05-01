import { useState, useCallback, useMemo } from 'react';
import {
  Table2, Columns3, KeyRound, ChevronUp, ChevronDown, RefreshCw, ChevronLeft,
  Play, List, FunctionSquare, Plus, Pencil, Trash2, Layers, Copy, Filter,
  ArrowUpDown, CircleDot, Eye, Zap, ListOrdered, SettingsIcon, Package, ShapesIcon,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import type {
  DbSchemaInfo,
  DbTableInfo,
  DbColumnInfo,
  DbViewInfo,
  DbRoutineInfo,
  DbTriggerInfo,
  DbSequenceInfo,
  DbPackageInfo,
  DbTypeInfo,
} from '../../api/database.api';
import {
  buildLimitedSelectSql,
  buildMongoCollectionQuery,
  buildMongoQuerySpec,
  getSchemaBrowserTerms,
  normalizeDbProtocol,
  qualifyDbObjectName,
  type DbProtocolHint,
} from './dbBrowserHelpers';

type BrowsableObjectType = 'table' | 'view' | 'function' | 'procedure' | 'trigger' | 'sequence' | 'package' | 'type';

interface DbSchemaBrowserProps {
  schema: DbSchemaInfo;
  open: boolean;
  onClose: () => void;
  onRefresh: () => void;
  onTableClick?: (tableName: string, schemaName: string) => void;
  onInsertSql?: (sql: string) => void;
  dbProtocol?: DbProtocolHint;
  loading?: boolean;
}

interface MenuState {
  anchor: HTMLElement;
  objectType: BrowsableObjectType;
  objectName: string;
  schema: string;
  table?: DbTableInfo;
  column?: DbColumnInfo;
}

interface SchemaGroup {
  tables: DbTableInfo[];
  views: DbViewInfo[];
  functions: DbRoutineInfo[];
  procedures: DbRoutineInfo[];
  triggers: DbTriggerInfo[];
  sequences: DbSequenceInfo[];
  packages: DbPackageInfo[];
  types: DbTypeInfo[];
}

type SectionType = keyof SchemaGroup;

interface SectionConfig {
  key: SectionType;
  label: string;
  icon: React.ReactNode;
}

function getSectionConfigs(tableSectionLabel: string): SectionConfig[] {
  return [
    { key: 'tables', label: tableSectionLabel, icon: <Table2 className="size-4 text-primary" /> },
    { key: 'views', label: 'Views', icon: <Eye className="size-4 text-blue-400" /> },
    { key: 'functions', label: 'Functions', icon: <FunctionSquare className="size-4 text-purple-400" /> },
    { key: 'procedures', label: 'Procedures', icon: <SettingsIcon className="size-4 text-purple-400" /> },
    { key: 'triggers', label: 'Triggers', icon: <Zap className="size-4 text-yellow-400" /> },
    { key: 'sequences', label: 'Sequences', icon: <ListOrdered className="size-4 text-muted-foreground" /> },
    { key: 'packages', label: 'Packages', icon: <Package className="size-4 text-muted-foreground" /> },
    { key: 'types', label: 'Types', icon: <ShapesIcon className="size-4 text-muted-foreground" /> },
  ];
}

function emptyGroup(): SchemaGroup {
  return { tables: [], views: [], functions: [], procedures: [], triggers: [], sequences: [], packages: [], types: [] };
}

function copyToClipboard(text: string) {
  try { navigator?.clipboard?.writeText(text); } catch { /* ignore */ }
}

function mongoFieldPlaceholder(column: DbColumnInfo): unknown {
  switch (column.dataType) {
    case 'number':
      return 0;
    case 'bool':
      return false;
    case 'array':
      return [];
    case 'document':
      return {};
    case 'date':
      return '2026-01-01T00:00:00Z';
    default:
      return '';
  }
}

export default function DbSchemaBrowser({
  schema,
  open,
  onClose,
  onRefresh,
  onTableClick,
  onInsertSql,
  dbProtocol,
  loading = false,
}: DbSchemaBrowserProps) {
  const [expandedTables, setExpandedTables] = useState<Record<string, boolean>>({});
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});
  const [menu, setMenu] = useState<MenuState | null>(null);
  const normalizedProtocol = normalizeDbProtocol(dbProtocol);
  const terms = useMemo(() => getSchemaBrowserTerms(normalizedProtocol), [normalizedProtocol]);
  const sectionConfigs = useMemo(() => getSectionConfigs(terms.tableSectionLabel), [terms.tableSectionLabel]);
  const isMongoProtocol = normalizedProtocol === 'mongodb';
  const fallbackGroupName = useMemo(() => {
    switch (normalizedProtocol) {
      case 'mongodb':
      case 'mysql':
        return 'default';
      case 'oracle':
        return 'current';
      default:
        return 'public';
    }
  }, [normalizedProtocol]);

  const closeMenu = useCallback(() => setMenu(null), []);

  const insertSql = useCallback((sql: string) => {
    onInsertSql?.(sql);
    setMenu(null);
  }, [onInsertSql]);

  // Build grouped schema map
  const schemaGroups = useMemo(() => {
    const groups: Record<string, SchemaGroup> = {};

    const ensure = (s: string) => {
      if (!groups[s]) groups[s] = emptyGroup();
      return groups[s];
    };

    for (const t of schema.tables) ensure(t.schema || fallbackGroupName).tables.push(t);
    for (const v of schema.views ?? []) ensure(v.schema || fallbackGroupName).views.push(v);
    for (const f of schema.functions ?? []) ensure(f.schema || fallbackGroupName).functions.push(f);
    for (const p of schema.procedures ?? []) ensure(p.schema || fallbackGroupName).procedures.push(p);
    for (const tr of schema.triggers ?? []) ensure(tr.schema || fallbackGroupName).triggers.push(tr);
    for (const sq of schema.sequences ?? []) ensure(sq.schema || fallbackGroupName).sequences.push(sq);
    for (const pk of schema.packages ?? []) ensure(pk.schema || fallbackGroupName).packages.push(pk);
    for (const tp of schema.types ?? []) ensure(tp.schema || fallbackGroupName).types.push(tp);

    return groups;
  }, [fallbackGroupName, schema]);

  const totalObjects = useMemo(() => {
    return schema.tables.length
      + (schema.views?.length ?? 0)
      + (schema.functions?.length ?? 0)
      + (schema.procedures?.length ?? 0)
      + (schema.triggers?.length ?? 0)
      + (schema.sequences?.length ?? 0)
      + (schema.packages?.length ?? 0)
      + (schema.types?.length ?? 0);
  }, [schema]);

  const handleContextMenu = useCallback((
    e: React.MouseEvent<HTMLElement>,
    objectType: BrowsableObjectType,
    objectName: string,
    schemaName: string,
    table?: DbTableInfo,
    column?: DbColumnInfo,
  ) => {
    e.preventDefault();
    e.stopPropagation();
    setMenu({ anchor: e.currentTarget, objectType, objectName, schema: schemaName, table, column });
  }, []);

  if (!open) return null;

  const toggleTable = (tableKey: string) => {
    setExpandedTables((prev) => ({ ...prev, [tableKey]: !prev[tableKey] }));
  };

  const toggleSection = (sectionKey: string) => {
    setExpandedSections((prev) => ({ ...prev, [sectionKey]: !prev[sectionKey] }));
  };

  const isSectionExpanded = (schemaName: string, section: SectionType): boolean => {
    const key = `${schemaName}:${section}`;
    // Tables default to expanded, everything else collapsed
    return expandedSections[key] ?? (section === 'tables');
  };

  // --- Context menu helpers ---
  const menuQn = menu ? qualifyDbObjectName(normalizedProtocol, menu.schema, menu.objectName) : '';
  const menuMongoQuery = (payload: Record<string, unknown>) => {
    if (menu?.schema) {
      return buildMongoQuerySpec({ database: menu.schema, ...payload });
    }
    return buildMongoQuerySpec(payload);
  };

  // Table-specific helpers (only valid when menu.table is present)
  const cols = menu?.table?.columns ?? [];
  const colNames = cols.map((c) => c.name).join(', ');
  const colPlaceholders = cols.map(() => '?').join(', ');
  const colSetters = cols.map((c) => `${c.name} = ?`).join(', ');

  // --- Context menu renderers per object type ---
  const renderSqlTableMenuItems = () => [
    <DropdownMenuItem key="select-all" onClick={() => insertSql(buildLimitedSelectSql(normalizedProtocol, '*', menuQn))}>
      <List className="size-4" />
      SELECT *
    </DropdownMenuItem>,
    cols.length > 0 && (
      <DropdownMenuItem key="select-cols" onClick={() => insertSql(buildLimitedSelectSql(normalizedProtocol, colNames, menuQn))}>
        <Play className="size-4" />
        SELECT columns
      </DropdownMenuItem>
    ),
    <DropdownMenuItem key="count" onClick={() => insertSql(`SELECT COUNT(*)\nFROM ${menuQn};`)}>
      <FunctionSquare className="size-4" />
      COUNT(*)
    </DropdownMenuItem>,
    <DropdownMenuSeparator key="d1" />,
    cols.length > 0 && (
      <DropdownMenuItem key="insert" onClick={() => insertSql(`INSERT INTO ${menuQn} (${colNames})\nVALUES (${colPlaceholders});`)}>
        <Plus className="size-4" />
        INSERT template
      </DropdownMenuItem>
    ),
    cols.length > 0 && (
      <DropdownMenuItem key="update" onClick={() => insertSql(`UPDATE ${menuQn}\nSET ${colSetters}\nWHERE ...;`)}>
        <Pencil className="size-4" />
        UPDATE template
      </DropdownMenuItem>
    ),
    <DropdownMenuItem key="delete" onClick={() => insertSql(`DELETE FROM ${menuQn}\nWHERE ...;`)}>
      <Trash2 className="size-4" />
      DELETE template
    </DropdownMenuItem>,
    <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP TABLE ${menuQn};`)}>
      <Layers className="size-4" />
      DROP TABLE
    </DropdownMenuItem>,
    <DropdownMenuSeparator key="d2" />,
    <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
      <Copy className="size-4" />
      {`Copy ${terms.tableObjectLabel} name`}
    </DropdownMenuItem>,
  ];

  const renderMongoTableMenuItems = () => {
    const projection = Object.fromEntries(cols.map((column) => [column.name, 1]));
    const documentTemplate = Object.fromEntries(
      cols
        .filter((column) => column.name !== '_id')
        .slice(0, 8)
        .map((column) => [column.name, mongoFieldPlaceholder(column)]),
    );

    return [
      <DropdownMenuItem key="find-docs" onClick={() => insertSql(buildMongoCollectionQuery(menu?.objectName ?? '', menu?.schema))}>
        <List className="size-4" />
        Find documents
      </DropdownMenuItem>,
      cols.length > 0 && (
        <DropdownMenuItem
          key="find-projection"
          onClick={() => insertSql(buildMongoCollectionQuery(menu?.objectName ?? '', menu?.schema, { projection }))}
        >
          <Play className="size-4" />
          Find with projection
        </DropdownMenuItem>
      ),
      <DropdownMenuItem
        key="count-docs"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'count',
          collection: menu?.objectName ?? '',
          filter: {},
        }))}
      >
        <FunctionSquare className="size-4" />
        Count documents
      </DropdownMenuItem>,
      <DropdownMenuItem
        key="aggregate"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'aggregate',
          collection: menu?.objectName ?? '',
          pipeline: [
            { $match: {} },
            { $limit: 100 },
          ],
        }))}
      >
        <CircleDot className="size-4" />
        Aggregate template
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="d1" />,
      <DropdownMenuItem
        key="insert-doc"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'insertOne',
          collection: menu?.objectName ?? '',
          document: documentTemplate,
        }))}
      >
        <Plus className="size-4" />
        Insert document
      </DropdownMenuItem>,
      <DropdownMenuItem
        key="update-docs"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'updateMany',
          collection: menu?.objectName ?? '',
          filter: {},
          update: {
            $set: documentTemplate,
          },
        }))}
      >
        <Pencil className="size-4" />
        Update documents
      </DropdownMenuItem>,
      <DropdownMenuItem
        key="delete-docs"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'deleteMany',
          collection: menu?.objectName ?? '',
          filter: {},
        }))}
      >
        <Trash2 className="size-4" />
        Delete documents
      </DropdownMenuItem>,
      <DropdownMenuItem
        key="drop-collection"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'runCommand',
          command: { drop: menu?.objectName ?? '' },
        }))}
      >
        <Layers className="size-4" />
        Drop collection
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="d2" />,
      <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menu?.objectName ?? ''); closeMenu(); }}>
        <Copy className="size-4" />
        Copy collection name
      </DropdownMenuItem>,
    ];
  };

  const renderTableMenuItems = () => (
    isMongoProtocol ? renderMongoTableMenuItems() : renderSqlTableMenuItems()
  );

  const renderSqlColumnMenuItems = () => {
    if (!menu?.column) return null;
    const colName = menu.column.name;
    return [
      <DropdownMenuItem key="copy-col" onClick={() => { copyToClipboard(colName); closeMenu(); }}>
        <Copy className="size-4" />
        Copy column name
      </DropdownMenuItem>,
      <DropdownMenuItem key="select-distinct" onClick={() => insertSql(`SELECT DISTINCT ${colName}\nFROM ${menuQn};`)}>
        <Play className="size-4" />
        SELECT DISTINCT
      </DropdownMenuItem>,
      <DropdownMenuItem key="where" onClick={() => { copyToClipboard(`WHERE ${colName} = ?`); closeMenu(); }}>
        <Filter className="size-4" />
        Copy WHERE clause
      </DropdownMenuItem>,
      <DropdownMenuItem key="order-by" onClick={() => { copyToClipboard(`ORDER BY ${colName} ASC`); closeMenu(); }}>
        <ArrowUpDown className="size-4" />
        Copy ORDER BY
      </DropdownMenuItem>,
      <DropdownMenuItem key="group-count" onClick={() => insertSql(`SELECT ${colName}, COUNT(*)\nFROM ${menuQn}\nGROUP BY ${colName}\nORDER BY COUNT(*) DESC;`)}>
        <CircleDot className="size-4" />
        GROUP BY + COUNT
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="col-divider" />,
    ];
  };

  const renderMongoColumnMenuItems = () => {
    if (!menu?.column) return null;
    const colName = menu.column.name;
    return [
      <DropdownMenuItem key="copy-col" onClick={() => { copyToClipboard(colName); closeMenu(); }}>
        <Copy className="size-4" />
        Copy field name
      </DropdownMenuItem>,
      <DropdownMenuItem
        key="distinct"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'distinct',
          collection: menu.objectName,
          field: colName,
          filter: {},
        }))}
      >
        <Play className="size-4" />
        Distinct values
      </DropdownMenuItem>,
      <DropdownMenuItem key="copy-filter" onClick={() => { copyToClipboard(`{ "${colName}": "" }`); closeMenu(); }}>
        <Filter className="size-4" />
        Copy filter
      </DropdownMenuItem>,
      <DropdownMenuItem key="copy-sort" onClick={() => { copyToClipboard(`{ "${colName}": 1 }`); closeMenu(); }}>
        <ArrowUpDown className="size-4" />
        Copy sort
      </DropdownMenuItem>,
      <DropdownMenuItem
        key="group-count"
        onClick={() => insertSql(menuMongoQuery({
          operation: 'aggregate',
          collection: menu.objectName,
          pipeline: [
            {
              $group: {
                _id: `$${colName}`,
                count: { $sum: 1 },
              },
            },
            { $sort: { count: -1 } },
            { $limit: 100 },
          ],
        }))}
      >
        <CircleDot className="size-4" />
        Group by field
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="col-divider" />,
    ];
  };

  const renderColumnMenuItems = () => (
    isMongoProtocol ? renderMongoColumnMenuItems() : renderSqlColumnMenuItems()
  );

  const renderViewMenuItems = () => {
    // Check if view is materialized (look it up from schema data)
    const viewObj = (schema.views ?? []).find((v) => v.name === menu?.objectName && v.schema === menu?.schema);
    const isMaterialized = viewObj?.materialized;

    return [
      <DropdownMenuItem key="select-all" onClick={() => insertSql(buildLimitedSelectSql(normalizedProtocol, '*', menuQn))}>
        <List className="size-4" />
        SELECT *
      </DropdownMenuItem>,
      <DropdownMenuItem key="count" onClick={() => insertSql(`SELECT COUNT(*)\nFROM ${menuQn};`)}>
        <FunctionSquare className="size-4" />
        COUNT(*)
      </DropdownMenuItem>,
      isMaterialized && normalizedProtocol === 'postgresql' && (
        <DropdownMenuItem key="refresh-mat" onClick={() => insertSql(`REFRESH MATERIALIZED VIEW ${menuQn};`)}>
          <RefreshCw className="size-4" />
          REFRESH MATERIALIZED VIEW
        </DropdownMenuItem>
      ),
      <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP VIEW ${menuQn};`)}>
        <Layers className="size-4" />
        DROP VIEW
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="d1" />,
      <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
        <Copy className="size-4" />
        Copy name
      </DropdownMenuItem>,
    ];
  };

  const renderFunctionMenuItems = () => {
    const callSyntax = (dbProtocol === 'oracle' || dbProtocol === 'mssql')
      ? `SELECT ${menuQn}()`
      : `SELECT ${menuQn}();`;

    return [
      <DropdownMenuItem key="call" onClick={() => insertSql(callSyntax)}>
        <Play className="size-4" />
        Call function
      </DropdownMenuItem>,
      <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP FUNCTION ${menuQn};`)}>
        <Layers className="size-4" />
        DROP FUNCTION
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="d1" />,
      <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
        <Copy className="size-4" />
        Copy name
      </DropdownMenuItem>,
    ];
  };

  const renderProcedureMenuItems = () => {
    const callSyntax = (dbProtocol === 'mssql' || dbProtocol === 'oracle')
      ? `EXEC ${menuQn};`
      : `CALL ${menuQn}();`;

    return [
      <DropdownMenuItem key="call" onClick={() => insertSql(callSyntax)}>
        <Play className="size-4" />
        Call procedure
      </DropdownMenuItem>,
      <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP PROCEDURE ${menuQn};`)}>
        <Layers className="size-4" />
        DROP PROCEDURE
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="d1" />,
      <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
        <Copy className="size-4" />
        Copy name
      </DropdownMenuItem>,
    ];
  };

  const renderTriggerMenuItems = () => [
    <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP TRIGGER ${menuQn};`)}>
      <Layers className="size-4" />
      DROP TRIGGER
    </DropdownMenuItem>,
    <DropdownMenuSeparator key="d1" />,
    <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
      <Copy className="size-4" />
      Copy name
    </DropdownMenuItem>,
  ];

  const renderSequenceMenuItems = () => {
    let nextvalSql: string;
    switch (dbProtocol) {
      case 'oracle':
        nextvalSql = `SELECT ${menuQn}.NEXTVAL FROM DUAL;`;
        break;
      case 'mssql':
      case 'db2':
        nextvalSql = `SELECT NEXT VALUE FOR ${menuQn};`;
        break;
      default:
        nextvalSql = `SELECT nextval('${menuQn}');`;
        break;
    }

    return [
      <DropdownMenuItem key="nextval" onClick={() => insertSql(nextvalSql)}>
        <Play className="size-4" />
        NEXTVAL
      </DropdownMenuItem>,
      <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP SEQUENCE ${menuQn};`)}>
        <Layers className="size-4" />
        DROP SEQUENCE
      </DropdownMenuItem>,
      <DropdownMenuSeparator key="d1" />,
      <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
        <Copy className="size-4" />
        Copy name
      </DropdownMenuItem>,
    ];
  };

  const renderPackageMenuItems = () => [
    <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP PACKAGE ${menuQn};`)}>
      <Layers className="size-4" />
      DROP PACKAGE
    </DropdownMenuItem>,
    <DropdownMenuSeparator key="d1" />,
    <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
      <Copy className="size-4" />
      Copy name
    </DropdownMenuItem>,
  ];

  const renderTypeMenuItems = () => [
    <DropdownMenuItem key="drop" onClick={() => insertSql(`DROP TYPE ${menuQn};`)}>
      <Layers className="size-4" />
      DROP TYPE
    </DropdownMenuItem>,
    <DropdownMenuSeparator key="d1" />,
    <DropdownMenuItem key="copy" onClick={() => { copyToClipboard(menuQn); closeMenu(); }}>
      <Copy className="size-4" />
      Copy name
    </DropdownMenuItem>,
  ];

  const renderMenuContent = () => {
    if (!menu) return null;

    switch (menu.objectType) {
      case 'table':
        return (
          <>
            {menu.column && renderColumnMenuItems()}
            {renderTableMenuItems()}
          </>
        );
      case 'view':
        return <>{renderViewMenuItems()}</>;
      case 'function':
        return <>{renderFunctionMenuItems()}</>;
      case 'procedure':
        return <>{renderProcedureMenuItems()}</>;
      case 'trigger':
        return <>{renderTriggerMenuItems()}</>;
      case 'sequence':
        return <>{renderSequenceMenuItems()}</>;
      case 'package':
        return <>{renderPackageMenuItems()}</>;
      case 'type':
        return <>{renderTypeMenuItems()}</>;
      default:
        return null;
    }
  };

  // --- Section item renderers ---
  const renderTableItem = (table: DbTableInfo, schemaName: string) => {
    const tableKey = `${schemaName}.${table.name}`;
    const isExpanded = expandedTables[tableKey] ?? false;

    return (
      <div key={tableKey}>
        <button
          onClick={() => toggleTable(tableKey)}
          onDoubleClick={() => onTableClick?.(table.name, schemaName)}
          onContextMenu={(e) => handleContextMenu(e, 'table', table.name, schemaName, table)}
          className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
        >
          <Table2 className="size-3.5 text-primary shrink-0" />
          <span className="text-sm truncate flex-1">{table.name}</span>
          {isMongoProtocol && (
            <span className="text-[0.65rem] text-muted-foreground">{table.columns.length} fields</span>
          )}
          {isExpanded ? (
            <ChevronUp className="size-3.5 shrink-0" />
          ) : (
            <ChevronDown className="size-3.5 shrink-0" />
          )}
        </button>

        {isExpanded && (
          <div>
            {table.columns.map((col) => (
              <button
                key={col.name}
                className="w-full flex items-center gap-1.5 py-0 pl-14 pr-2 hover:bg-accent/50 transition-colors text-left"
                onContextMenu={(e) => handleContextMenu(e, 'table', table.name, schemaName, table, col)}
              >
                {col.isPrimaryKey ? (
                  <KeyRound className="size-3 text-yellow-400 shrink-0" />
                ) : (
                  <Columns3 className="size-3 text-muted-foreground shrink-0" />
                )}
                <span className="text-xs truncate">{col.name}</span>
                <span className="text-[0.65rem] text-muted-foreground ml-auto shrink-0">
                  {col.dataType}{col.nullable ? ' (nullable)' : ''}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    );
  };

  const renderViewItem = (view: DbViewInfo, schemaName: string) => (
    <button
      key={`${schemaName}.${view.name}`}
      className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
      onContextMenu={(e) => handleContextMenu(e, 'view', view.name, schemaName)}
    >
      <Eye className="size-3.5 text-blue-400 shrink-0" />
      <span className="text-sm truncate flex-1">{view.name}</span>
      {view.materialized && <span className="text-[0.65rem] text-muted-foreground">materialized</span>}
    </button>
  );

  const renderRoutineItem = (routine: DbRoutineInfo, schemaName: string, type: 'function' | 'procedure') => {
    const Icon = type === 'function' ? FunctionSquare : SettingsIcon;
    return (
      <button
        key={`${schemaName}.${routine.name}`}
        className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
        onContextMenu={(e) => handleContextMenu(e, type, routine.name, schemaName)}
      >
        <Icon className="size-3.5 text-purple-400 shrink-0" />
        <span className="text-sm truncate flex-1">{routine.name}</span>
        {type === 'function' && routine.returnType && (
          <span className="text-[0.65rem] text-muted-foreground">{'\u2192'} {routine.returnType}</span>
        )}
      </button>
    );
  };

  const renderTriggerItem = (trigger: DbTriggerInfo, schemaName: string) => (
    <button
      key={`${schemaName}.${trigger.name}`}
      className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
      onContextMenu={(e) => handleContextMenu(e, 'trigger', trigger.name, schemaName)}
    >
      <Zap className="size-3.5 text-yellow-400 shrink-0" />
      <span className="text-sm truncate flex-1">{trigger.name}</span>
      <span className="text-[0.65rem] text-muted-foreground">on {trigger.tableName}</span>
    </button>
  );

  const renderSequenceItem = (seq: DbSequenceInfo, schemaName: string) => (
    <button
      key={`${schemaName}.${seq.name}`}
      className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
      onContextMenu={(e) => handleContextMenu(e, 'sequence', seq.name, schemaName)}
    >
      <ListOrdered className="size-3.5 text-muted-foreground shrink-0" />
      <span className="text-sm truncate">{seq.name}</span>
    </button>
  );

  const renderPackageItem = (pkg: DbPackageInfo, schemaName: string) => (
    <button
      key={`${schemaName}.${pkg.name}`}
      className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
      onContextMenu={(e) => handleContextMenu(e, 'package', pkg.name, schemaName)}
    >
      <Package className="size-3.5 text-muted-foreground shrink-0" />
      <span className="text-sm truncate flex-1">{pkg.name}</span>
      {pkg.hasBody && <span className="text-[0.65rem] text-muted-foreground">body</span>}
    </button>
  );

  const renderTypeItem = (typeObj: DbTypeInfo, schemaName: string) => (
    <button
      key={`${schemaName}.${typeObj.name}`}
      className="w-full flex items-center gap-1.5 py-0.5 pl-8 pr-2 hover:bg-accent/50 transition-colors text-left"
      onContextMenu={(e) => handleContextMenu(e, 'type', typeObj.name, schemaName)}
    >
      <ShapesIcon className="size-3.5 text-muted-foreground shrink-0" />
      <span className="text-sm truncate flex-1">{typeObj.name}</span>
      {typeObj.kind && <span className="text-[0.65rem] text-muted-foreground">{typeObj.kind}</span>}
    </button>
  );

  const renderSectionItems = (sectionKey: SectionType, items: SchemaGroup[SectionType], schemaName: string) => {
    switch (sectionKey) {
      case 'tables':
        return (items as DbTableInfo[]).map((t) => renderTableItem(t, schemaName));
      case 'views':
        return (items as DbViewInfo[]).map((v) => renderViewItem(v, schemaName));
      case 'functions':
        return (items as DbRoutineInfo[]).map((f) => renderRoutineItem(f, schemaName, 'function'));
      case 'procedures':
        return (items as DbRoutineInfo[]).map((p) => renderRoutineItem(p, schemaName, 'procedure'));
      case 'triggers':
        return (items as DbTriggerInfo[]).map((tr) => renderTriggerItem(tr, schemaName));
      case 'sequences':
        return (items as DbSequenceInfo[]).map((sq) => renderSequenceItem(sq, schemaName));
      case 'packages':
        return (items as DbPackageInfo[]).map((pk) => renderPackageItem(pk, schemaName));
      case 'types':
        return (items as DbTypeInfo[]).map((tp) => renderTypeItem(tp, schemaName));
      default:
        return null;
    }
  };

  return (
    <div className="w-[260px] min-w-[260px] border-l border-border flex flex-col overflow-hidden">
      <div className="flex items-center justify-between px-2 py-1 border-b border-border">
        <span className="text-sm font-semibold">
          {terms.title}
        </span>
        <div className="flex">
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            title="Refresh schema"
            onClick={onRefresh}
            disabled={loading}
          >
            <RefreshCw className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            title="Close schema browser"
            onClick={onClose}
          >
            <ChevronLeft className="size-4" />
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {totalObjects === 0 && !loading && (
          <span className="text-xs text-muted-foreground p-4 block">
            {terms.emptyMessage}
          </span>
        )}

        {loading && (
          <span className="text-xs text-muted-foreground p-4 block">
            Loading schema...
          </span>
        )}

        {Object.entries(schemaGroups).map(([schemaName, group]) => (
          <div key={schemaName}>
            <span className="text-[0.65rem] uppercase tracking-wide text-muted-foreground px-4 pt-2 block">
              {`${terms.groupLabel}: ${schemaName}`}
            </span>
            <Separator />

            {sectionConfigs.map(({ key, label, icon }) => {
              const items = group[key];
              if (items.length === 0) return null;

              const sectionKey = `${schemaName}:${key}`;
              const expanded = isSectionExpanded(schemaName, key);

              return (
                <div key={sectionKey}>
                  <button
                    onClick={() => toggleSection(sectionKey)}
                    className="w-full flex items-center gap-1.5 py-0.5 pl-4 pr-2 hover:bg-accent/50 transition-colors text-left"
                  >
                    {icon}
                    <span className="text-sm font-medium truncate flex-1">
                      {label} ({items.length})
                    </span>
                    {expanded ? (
                      <ChevronUp className="size-3.5" />
                    ) : (
                      <ChevronDown className="size-3.5" />
                    )}
                  </button>

                  {expanded && (
                    <div>
                      {renderSectionItems(key, items, schemaName)}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ))}
      </div>

      {/* Context menu */}
      <DropdownMenu open={Boolean(menu)} onOpenChange={(open) => !open && closeMenu()}>
        <DropdownMenuTrigger asChild>
          <span className="hidden" />
        </DropdownMenuTrigger>
        <DropdownMenuContent
          className="min-w-[220px] max-w-[320px]"
          style={menu?.anchor ? {
            position: 'fixed',
            left: menu.anchor.getBoundingClientRect().left,
            top: menu.anchor.getBoundingClientRect().bottom,
          } : undefined}
        >
          {renderMenuContent()}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
