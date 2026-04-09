import { DatabaseZap, Monitor, TerminalSquare, X } from 'lucide-react';
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from '@/components/ui/context-menu';
import { useTabsStore } from '@/store/tabsStore';
import { cn } from '@/lib/utils';

function tabIcon(type: string) {
  switch (type) {
    case 'SSH':
      return <TerminalSquare className="size-4" />;
    case 'VNC':
      return <Monitor className="size-4" />;
    case 'DATABASE':
      return <DatabaseZap className="size-4" />;
    default:
      return <Monitor className="size-4" />;
  }
}

export default function TabBar() {
  const tabs = useTabsStore((state) => state.tabs);
  const activeTabId = useTabsStore((state) => state.activeTabId);
  const setActiveTab = useTabsStore((state) => state.setActiveTab);
  const closeTab = useTabsStore((state) => state.closeTab);

  if (tabs.length === 0) {
    return null;
  }

  const handleCloseOthers = (keepId: string) => {
    tabs.forEach((t) => { if (t.id !== keepId) closeTab(t.id); });
  };

  const handleCloseAll = () => {
    tabs.forEach((t) => closeTab(t.id));
  };

  return (
    <div className="border-b bg-background/70 px-2 py-1.5 backdrop-blur">
      <div className="flex gap-1.5 overflow-x-auto pb-0.5">
        {tabs.map((tab) => {
          const isActive = tab.id === activeTabId;

          return (
            <ContextMenu key={tab.id}>
              <ContextMenuTrigger asChild>
                <div
                  className={cn(
                    'group inline-flex shrink-0 items-center gap-1 rounded-lg border px-1.5 py-1 text-sm transition-colors',
                    isActive
                      ? 'border-primary/40 bg-primary/10 text-foreground'
                      : 'border-transparent bg-transparent text-muted-foreground hover:border-border hover:bg-accent hover:text-foreground',
                  )}
                >
                  <button
                    type="button"
                    className="inline-flex min-w-0 flex-1 items-center gap-2 rounded-md px-1.5 py-0.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                    onClick={(event) => {
                      event.currentTarget.blur();
                      setActiveTab(tab.id);
                    }}
                  >
                    <span className={cn(isActive ? 'text-primary' : 'text-muted-foreground')}>
                      {tabIcon(tab.connection.type)}
                    </span>
                    <span className="max-w-44 truncate">{tab.connection.name}</span>
                  </button>
                  <button
                    type="button"
                    aria-label={`Close ${tab.connection.name}`}
                    className="inline-flex size-6 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-background/80 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                    onClick={() => closeTab(tab.id)}
                  >
                    <X className="size-3" />
                  </button>
                </div>
              </ContextMenuTrigger>
              <ContextMenuContent className="w-48">
                <ContextMenuItem onSelect={() => closeTab(tab.id)}>
                  Close
                </ContextMenuItem>
                <ContextMenuItem onSelect={() => handleCloseOthers(tab.id)} disabled={tabs.length <= 1}>
                  Close Others
                </ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem onSelect={handleCloseAll}>
                  Close All
                </ContextMenuItem>
              </ContextMenuContent>
            </ContextMenu>
          );
        })}
      </div>
    </div>
  );
}
