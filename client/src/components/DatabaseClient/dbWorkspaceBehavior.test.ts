import {
  activeQueryTabIdForTabs,
  classifyQueryType,
  defaultSessionConfigForProtocol,
  hasSessionConfigValues,
  persistableQuerySubTabs,
  restoreQuerySubTabs,
  resultToCsv,
  stripLeadingComments,
} from './dbWorkspaceBehavior';

describe('dbWorkspaceBehavior', () => {
  it('strips leading comments before classifying queries', () => {
    expect(stripLeadingComments('-- explain next\nselect * from users')).toBe('select * from users');
    expect(stripLeadingComments('/* migration */\nupdate users set name = ?')).toBe('update users set name = ?');
  });

  it('classifies workspace query actions', () => {
    expect(classifyQueryType('select * from users')).toBe('SELECT');
    expect(classifyQueryType('with q as (select 1) update users set name = q.x')).toBe('UPDATE');
    expect(classifyQueryType('merge into users using incoming on users.id = incoming.id')).toBe('UPDATE');
    expect(classifyQueryType('call refresh_stats()')).toBe('EXEC');
  });

  it('derives protocol session defaults', () => {
    expect(defaultSessionConfigForProtocol('postgresql', 'app')).toMatchObject({
      activeDatabase: 'app',
      searchPath: 'public',
    });
    expect(defaultSessionConfigForProtocol('mssql', 'app')).toEqual({ activeDatabase: 'app' });
    expect(defaultSessionConfigForProtocol('mongodb', 'app')).toEqual({});
  });

  it('restores query tabs without transient execution state', () => {
    const tabs = restoreQuerySubTabs({
      activeId: 'tab-2',
      tabs: [
        { id: 'tab-1', label: 'Query 1', sql: 'select 1' },
        { id: 'tab-2', label: 'Report', sql: 'select 2' },
      ],
    });

    expect(tabs).toEqual([
      { id: 'tab-1', label: 'Query 1', sql: 'select 1', result: null, executing: false },
      { id: 'tab-2', label: 'Report', sql: 'select 2', result: null, executing: false },
    ]);
    expect(activeQueryTabIdForTabs(tabs, { activeId: 'tab-2', tabs: [] })).toBe('tab-2');
    expect(persistableQuerySubTabs(tabs, 'tab-2')).toEqual({
      activeId: 'tab-2',
      tabs: [
        { id: 'tab-1', label: 'Query 1', sql: 'select 1' },
        { id: 'tab-2', label: 'Report', sql: 'select 2' },
      ],
    });
  });

  it('detects populated session config values', () => {
    expect(hasSessionConfigValues({})).toBe(false);
    expect(hasSessionConfigValues({ activeDatabase: '' })).toBe(false);
    expect(hasSessionConfigValues({ activeDatabase: 'analytics' })).toBe(true);
  });

  it('exports result rows as escaped CSV', () => {
    expect(resultToCsv({
      columns: ['id', 'name'],
      rows: [{ id: 1, name: 'ACME, "West"' }],
      rowCount: 1,
      durationMs: 2,
      truncated: false,
    })).toBe('id,name\n1,"ACME, ""West"""');
  });
});
