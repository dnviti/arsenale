import {
  Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, Stack, Box, Paper,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  SkipNext as SkipIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import type { SyncPlanData } from '../../api/sync.api';

interface SyncPreviewDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  plan: SyncPlanData | null;
  confirming: boolean;
}

export default function SyncPreviewDialog({ open, onClose, onConfirm, plan, confirming }: SyncPreviewDialogProps) {
  if (!plan) return null;

  const totalItems = plan.toCreate.length + plan.toUpdate.length + plan.toSkip.length + plan.errors.length;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Sync Preview</DialogTitle>
      <DialogContent>
        <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
          <Chip icon={<AddIcon />} label={`Create: ${plan.toCreate.length}`} color="success" variant="outlined" />
          <Chip icon={<EditIcon />} label={`Update: ${plan.toUpdate.length}`} color="info" variant="outlined" />
          <Chip icon={<SkipIcon />} label={`Skip: ${plan.toSkip.length}`} variant="outlined" />
          {plan.errors.length > 0 && (
            <Chip icon={<ErrorIcon />} label={`Errors: ${plan.errors.length}`} color="error" variant="outlined" />
          )}
        </Stack>

        {totalItems === 0 ? (
          <Typography color="text.secondary">No changes to apply.</Typography>
        ) : (
          <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 400 }}>
            <Table size="small" stickyHeader>
              <TableHead>
                <TableRow>
                  <TableCell>Action</TableCell>
                  <TableCell>Name</TableCell>
                  <TableCell>Host</TableCell>
                  <TableCell>Protocol</TableCell>
                  <TableCell>Details</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {plan.toCreate.map((d) => (
                  <TableRow key={d.externalId}>
                    <TableCell><Chip label="Create" color="success" size="small" /></TableCell>
                    <TableCell>{d.name}</TableCell>
                    <TableCell>{d.host}:{d.port}</TableCell>
                    <TableCell>{d.protocol}</TableCell>
                    <TableCell>{d.siteName && `${d.siteName}${d.rackName ? ` / ${d.rackName}` : ''}`}</TableCell>
                  </TableRow>
                ))}
                {plan.toUpdate.map((entry) => (
                  <TableRow key={entry.device.externalId}>
                    <TableCell><Chip label="Update" color="info" size="small" /></TableCell>
                    <TableCell>{entry.device.name}</TableCell>
                    <TableCell>{entry.device.host}:{entry.device.port}</TableCell>
                    <TableCell>{entry.device.protocol}</TableCell>
                    <TableCell>
                      <Box sx={{ fontSize: '0.75rem' }}>{entry.changes.join(', ')}</Box>
                    </TableCell>
                  </TableRow>
                ))}
                {plan.toSkip.map((entry) => (
                  <TableRow key={entry.device.externalId}>
                    <TableCell><Chip label="Skip" size="small" /></TableCell>
                    <TableCell>{entry.device.name}</TableCell>
                    <TableCell>{entry.device.host}:{entry.device.port}</TableCell>
                    <TableCell>{entry.device.protocol}</TableCell>
                    <TableCell>
                      <Box sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>{entry.reason}</Box>
                    </TableCell>
                  </TableRow>
                ))}
                {plan.errors.map((entry) => (
                  <TableRow key={entry.device.externalId}>
                    <TableCell><Chip label="Error" color="error" size="small" /></TableCell>
                    <TableCell>{entry.device.name}</TableCell>
                    <TableCell>{entry.device.host}:{entry.device.port}</TableCell>
                    <TableCell>{entry.device.protocol}</TableCell>
                    <TableCell>
                      <Box sx={{ fontSize: '0.75rem', color: 'error.main' }}>{entry.error}</Box>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={confirming}>Cancel</Button>
        <Button
          onClick={onConfirm}
          variant="contained"
          disabled={confirming || (plan.toCreate.length === 0 && plan.toUpdate.length === 0)}
        >
          {confirming ? 'Importing...' : 'Confirm Import'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
