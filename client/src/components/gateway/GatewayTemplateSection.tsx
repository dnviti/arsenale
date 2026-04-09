import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Plus, Trash2, Pencil, Rocket, FileText, Loader2 } from 'lucide-react';
import { useGatewayStore } from '../../store/gatewayStore';
import { useNotificationStore } from '../../store/notificationStore';
import type { GatewayTemplateData } from '../../api/gateway.api';
import GatewayTemplateDialog from './GatewayTemplateDialog';
import { extractApiError } from '../../utils/apiError';
import { gatewayModeLabel, isGatewayGroup } from '../../utils/gatewayMode';

export default function GatewayTemplateSection() {
  const templates = useGatewayStore((s) => s.templates);
  const templatesLoading = useGatewayStore((s) => s.templatesLoading);
  const fetchTemplates = useGatewayStore((s) => s.fetchTemplates);
  const deleteTemplateAction = useGatewayStore((s) => s.deleteTemplate);
  const deployFromTemplateAction = useGatewayStore((s) => s.deployFromTemplate);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<GatewayTemplateData | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<GatewayTemplateData | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deployingId, setDeployingId] = useState<string | null>(null);
  const [error, setError] = useState('');
  const notify = useNotificationStore((s) => s.notify);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  const handleEdit = (tpl: GatewayTemplateData) => {
    setEditingTemplate(tpl);
    setDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setError('');
    try {
      await deleteTemplateAction(deleteTarget.id);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to delete template'));
    } finally {
      setDeleting(false);
      setDeleteTarget(null);
    }
  };

  const handleDeploy = async (tpl: GatewayTemplateData) => {
    setDeployingId(tpl.id);
    setError('');
    try {
      const gateway = await deployFromTemplateAction(tpl.id);
      notify(`Gateway "${gateway.name}" created and deployment started.`, 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to deploy from template'));
    } finally {
      setDeployingId(null);
    }
  };

  const typeBadge = (tpl: GatewayTemplateData) => {
    const label = tpl.type === 'GUACD' ? 'GUACD' : tpl.type === 'MANAGED_SSH' ? 'Managed SSH' : tpl.type === 'DB_PROXY' ? 'DB Proxy' : 'SSH Bastion';
    const cls = tpl.type === 'GUACD' ? 'bg-blue-500/15 text-blue-400 border-blue-500/30'
      : tpl.type === 'MANAGED_SSH' ? 'bg-green-500/15 text-green-400 border-green-500/30'
      : tpl.type === 'DB_PROXY' ? 'bg-purple-500/15 text-purple-400 border-purple-500/30'
      : 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30';
    return <Badge variant="outline" className={cls}>{label}</Badge>;
  };

  return (
    <div>
      <div className="flex items-center mb-3">
        <h3 className="text-lg font-semibold flex-1">Gateway Templates</h3>
        <Button
          variant="outline"
          onClick={() => { setEditingTemplate(null); setDialogOpen(true); }}
        >
          <Plus className="h-4 w-4 mr-1" />
          New Template
        </Button>
      </div>

      {error && (
        <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError('')} className="text-red-400 hover:text-red-300 text-xs">dismiss</button>
        </div>
      )}

      {templatesLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : templates.length === 0 ? (
        <div className="text-center py-12">
          <FileText className="h-16 w-16 text-muted-foreground/30 mx-auto mb-4" />
          <h4 className="text-lg font-semibold mb-2">No Templates Yet</h4>
          <p className="text-sm text-muted-foreground mb-6">
            Create a template to quickly deploy pre-configured gateways.
          </p>
          <Button onClick={() => { setEditingTemplate(null); setDialogOpen(true); }}>
            <Plus className="h-4 w-4 mr-1" />
            Create Template
          </Button>
        </div>
      ) : (
        <div className="rounded-lg border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="text-left py-2 px-3 font-medium">Name</th>
                <th className="text-left py-2 px-3 font-medium">Type</th>
                <th className="text-left py-2 px-3 font-medium">Endpoint</th>
                <th className="text-left py-2 px-3 font-medium">Auto-Scale</th>
                <th className="text-left py-2 px-3 font-medium">Deployed</th>
                <th className="text-right py-2 px-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {templates.map((tpl) => (
                <tr key={tpl.id} className="border-b border-border/50">
                  <td className="py-2 px-3">
                    <p className="text-sm">{tpl.name}</p>
                    {tpl.description && (
                      <p className="text-xs text-muted-foreground">{tpl.description}</p>
                    )}
                  </td>
                  <td className="py-2 px-3">{typeBadge(tpl)}</td>
                  <td className="py-2 px-3">
                    {isGatewayGroup(tpl) ? (
                      <>
                        <p className="text-sm text-muted-foreground italic">
                          {gatewayModeLabel(tpl)}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          Service port {tpl.port}
                        </p>
                      </>
                    ) : (
                      <p className="text-sm">{tpl.host}:{tpl.port}</p>
                    )}
                  </td>
                  <td className="py-2 px-3">
                    <Badge variant="outline" className={tpl.autoScale ? 'bg-green-500/15 text-green-400 border-green-500/30' : ''}>
                      {tpl.autoScale ? 'Enabled' : 'Disabled'}
                    </Badge>
                  </td>
                  <td className="py-2 px-3">{tpl._count.gateways}</td>
                  <td className="py-2 px-3 text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => handleDeploy(tpl)}
                      disabled={deployingId === tpl.id}
                      title="Deploy from template"
                    >
                      {deployingId === tpl.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
                    </Button>
                    <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleEdit(tpl)} title="Edit">
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-7 w-7 text-red-400" onClick={() => setDeleteTarget(tpl)} title="Delete">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <GatewayTemplateDialog
        open={dialogOpen}
        onClose={() => { setDialogOpen(false); setEditingTemplate(null); }}
        template={editingTemplate}
      />

      <Dialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Template</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete template <strong>{deleteTarget?.name}</strong>?
              Existing gateways created from this template will not be affected.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
