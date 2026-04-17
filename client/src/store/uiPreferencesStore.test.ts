import { useUiPreferencesStore } from './uiPreferencesStore';

describe('useUiPreferencesStore', () => {
  beforeEach(() => {
    localStorage.clear();
    useUiPreferencesStore.setState(useUiPreferencesStore.getInitialState(), true);
  });

  it('sets arbitrary preference keys and persists them', () => {
    useUiPreferencesStore.getState().set('settingsActiveTab', 'notifications');

    expect(useUiPreferencesStore.getState().settingsActiveTab).toBe('notifications');

    const persisted = JSON.parse(localStorage.getItem('arsenale-ui-preferences') ?? '{}');
    expect(persisted.state.settingsActiveTab).toBe('notifications');
  });

  it('toggles boolean preferences', () => {
    expect(useUiPreferencesStore.getState().sidebarCompact).toBe(false);

    useUiPreferencesStore.getState().toggle('sidebarCompact');

    expect(useUiPreferencesStore.getState().sidebarCompact).toBe(true);
  });

  it('toggles team sections from their default expanded state', () => {
    useUiPreferencesStore.getState().toggleTeamSection('team-1');
    expect(useUiPreferencesStore.getState().sidebarTeamSections).toEqual({ 'team-1': false });

    useUiPreferencesStore.getState().toggleTeamSection('team-1');
    expect(useUiPreferencesStore.getState().sidebarTeamSections).toEqual({ 'team-1': true });
  });

  it('toggles keychain folder expansion from its default expanded state', () => {
    useUiPreferencesStore.getState().toggleKeychainFolder('folder-1');

    expect(useUiPreferencesStore.getState().keychainFolderExpandState).toEqual({
      'folder-1': false,
    });
  });

  it('persists database editor state per tab instance id', () => {
    useUiPreferencesStore.getState().set('dbQuerySubTabs', {
      'tab-a': {
        tabs: [{ id: 'query-a', label: 'Query 1', sql: 'select 1' }],
        activeId: 'query-a',
      },
      'tab-b': {
        tabs: [{ id: 'query-b', label: 'Query 1', sql: 'select 2' }],
        activeId: 'query-b',
      },
    });
    useUiPreferencesStore.getState().set('dbSessionConfigs', {
      'tab-a': { activeDatabase: 'primary' },
      'tab-b': { activeDatabase: 'analytics' },
    });

    expect(useUiPreferencesStore.getState().dbQuerySubTabs['tab-a']?.tabs[0]?.sql).toBe('select 1');
    expect(useUiPreferencesStore.getState().dbQuerySubTabs['tab-b']?.tabs[0]?.sql).toBe('select 2');
    expect(useUiPreferencesStore.getState().dbSessionConfigs['tab-a']?.activeDatabase).toBe('primary');
    expect(useUiPreferencesStore.getState().dbSessionConfigs['tab-b']?.activeDatabase).toBe('analytics');

    const persisted = JSON.parse(localStorage.getItem('arsenale-ui-preferences') ?? '{}');
    expect(persisted.state.dbQuerySubTabs['tab-a'].tabs[0].sql).toBe('select 1');
    expect(persisted.state.dbSessionConfigs['tab-b'].activeDatabase).toBe('analytics');
  });

  it('removes database editor state for a closed tab instance', () => {
    useUiPreferencesStore.getState().set('dbQuerySubTabs', {
      'tab-a': { tabs: [{ id: 'query-a', label: 'Query 1', sql: 'select 1' }], activeId: 'query-a' },
      'tab-b': { tabs: [{ id: 'query-b', label: 'Query 1', sql: 'select 2' }], activeId: 'query-b' },
    });
    useUiPreferencesStore.getState().set('dbSessionConfigs', {
      'tab-a': { activeDatabase: 'primary' },
      'tab-b': { activeDatabase: 'analytics' },
    });

    useUiPreferencesStore.getState().removeDbTabState('tab-a');

    expect(useUiPreferencesStore.getState().dbQuerySubTabs).toEqual({
      'tab-b': { tabs: [{ id: 'query-b', label: 'Query 1', sql: 'select 2' }], activeId: 'query-b' },
    });
    expect(useUiPreferencesStore.getState().dbSessionConfigs).toEqual({
      'tab-b': { activeDatabase: 'analytics' },
    });
  });
});
