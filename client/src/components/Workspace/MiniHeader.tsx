import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { Separator } from '@/components/ui/separator';
import { logoutApi } from '@/api/auth.api';
import { useAuthStore } from '@/store/authStore';
import { useTabsStore } from '@/store/tabsStore';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import NotificationBell from '../Layout/NotificationBell';
import TenantSwitcher from '../Layout/TenantSwitcher';
import type { NavigationActions } from '@/utils/notificationActions';

function userInitial(user: { username?: string | null; email?: string | null } | null | undefined) {
  return (user?.username || user?.email || '?').trim().charAt(0).toUpperCase();
}

interface MiniHeaderProps {
  navigationActions: NavigationActions;
  onOpenSettings: (tab?: string) => void;
}

export default function MiniHeader({ navigationActions, onOpenSettings }: MiniHeaderProps) {
  const user = useAuthStore((s) => s.user);
  const authLogout = useAuthStore((s) => s.logout);
  const multiTenancyEnabled = useFeatureFlagsStore((s) => s.multiTenancyEnabled);
  const featureFlagsLoaded = useFeatureFlagsStore((s) => s.loaded);

  const handleLogout = async () => {
    try {
      await logoutApi();
    } catch {
      // Ignore logout API errors and clear local state anyway.
    }
    await useTabsStore.getState().clearAll();
    authLogout();
  };

  return (
    <header className="flex h-9 shrink-0 items-center gap-2 border-b bg-background/85 px-2 backdrop-blur-xl">
      <SidebarTrigger className="-ml-0.5" />
      <Separator orientation="vertical" className="mr-1 data-[orientation=vertical]:h-4" />

      <div className="font-heading text-sm tracking-tight text-foreground">
        Arsenale
      </div>

      {featureFlagsLoaded && multiTenancyEnabled ? (
        <TenantSwitcher onCreateOrg={() => onOpenSettings('organization')} />
      ) : null}

      <div className="ml-auto flex items-center gap-1">
        <NotificationBell navigationActions={navigationActions} />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              aria-label="Account menu"
              className="relative inline-flex size-7 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
            >
              <Avatar className="size-6">
                {user?.avatarData ? <AvatarImage src={user.avatarData} alt={user?.username || user?.email || 'User'} /> : null}
                <AvatarFallback className="text-[10px]">{userInitial(user)}</AvatarFallback>
              </Avatar>
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="space-y-0.5">
              <div className="truncate text-sm font-medium text-foreground">
                {user?.username || user?.email}
              </div>
              {user?.email && user?.username ? (
                <div className="truncate text-xs text-muted-foreground">{user.email}</div>
              ) : null}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => onOpenSettings()}>
              Settings
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => void handleLogout()}>
              Logout
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
