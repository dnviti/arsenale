import { useEffect, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { useConnectionsStore } from '../../store/connectionsStore';
import { useTeamStore } from '../../store/teamStore';
import type { TeamData } from '../../api/team.api';

interface TeamDialogProps {
  open: boolean;
  onClose: () => void;
  team?: TeamData | null;
}

export default function TeamDialog({ open, onClose, team }: TeamDialogProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const { loading, error, setError, clearError, run } = useAsyncAction();
  const createTeam = useTeamStore((state) => state.createTeam);
  const updateTeam = useTeamStore((state) => state.updateTeam);
  const fetchConnections = useConnectionsStore((state) => state.fetchConnections);

  const isEditMode = Boolean(team);

  useEffect(() => {
    if (!open) {
      return;
    }

    if (team) {
      setName(team.name);
      setDescription(team.description || '');
    } else {
      setName('');
      setDescription('');
    }

    clearError();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- clearError is stable and only used to reset dialog state on open
  }, [open, team]);

  const handleClose = () => {
    setName('');
    setDescription('');
    clearError();
    onClose();
  };

  const handleSubmit = async () => {
    const trimmedName = name.trim();
    const trimmedDescription = description.trim();

    if (!trimmedName) {
      setError('Team name is required');
      return;
    }

    if (trimmedName.length < 2 || trimmedName.length > 100) {
      setError('Team name must be between 2 and 100 characters');
      return;
    }

    const isSuccessful = await run(async () => {
      if (team) {
        const payload: { name?: string; description?: string | null } = {};

        if (trimmedName !== team.name) {
          payload.name = trimmedName;
        }

        if (trimmedDescription !== (team.description || '')) {
          payload.description = trimmedDescription || null;
        }

        if (Object.keys(payload).length > 0) {
          await updateTeam(team.id, payload);
        }
      } else {
        await createTeam(trimmedName, trimmedDescription || undefined);
        await fetchConnections();
      }
    }, isEditMode ? 'Failed to update team' : 'Failed to create team');

    if (isSuccessful) {
      handleClose();
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          handleClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit Team' : 'Create Team'}</DialogTitle>
          <DialogDescription>
            {isEditMode
              ? 'Update the team name and description without changing its members.'
              : 'Create a focused team for shared connections, folders, and delegated access.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="team-name">Team name</Label>
            <Input
              id="team-name"
              value={name}
              autoFocus
              maxLength={100}
              placeholder="Platform Operations"
              onChange={(event) => setName(event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Use a clear, stable name that members can scan quickly in settings and sharing flows.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="team-description">Description</Label>
            <Textarea
              id="team-description"
              value={description}
              rows={4}
              maxLength={500}
              placeholder="Owns database proxy operations and on-call gateway access."
              onChange={(event) => setDescription(event.target.value)}
            />
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={loading}>
            {loading
              ? (isEditMode ? 'Saving...' : 'Creating...')
              : (isEditMode ? 'Save Changes' : 'Create Team')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
