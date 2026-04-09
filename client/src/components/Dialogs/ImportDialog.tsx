import { useState, useRef } from 'react';
import {
  Dialog, DialogContent, DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Upload, Loader2, X } from 'lucide-react';
import { importConnections, type ImportResult, type ImportOptions } from '../../api/importExport.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { useNotificationStore } from '../../store/notificationStore';

interface ImportDialogProps {
  open: boolean;
  onClose: () => void;
}

interface ConnectionPreview {
  name: string;
  host: string;
  port: number;
  type: string;
  username?: string;
  folder?: string;
}

const STEPS = ['Upload', 'Preview', 'Options', 'Results'];

export default function ImportDialog({ open, onClose }: ImportDialogProps) {
  const [step, setStep] = useState(0);
  const [file, setFile] = useState<File | null>(null);
  const [format, setFormat] = useState<'CSV' | 'JSON' | 'MREMOTENG' | 'RDP' | null>(null);
  const [preview, setPreview] = useState<ConnectionPreview[]>([]);
  const [duplicateStrategy, setDuplicateStrategy] = useState<'SKIP' | 'OVERWRITE' | 'RENAME'>('SKIP');
  const { loading, error, setError, clearError, run } = useAsyncAction();
  const [result, setResult] = useState<ImportResult | null>(null);
  const notify = useNotificationStore((s) => s.notify);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (!selected) return;

    const detectedFormat = detectFormat(selected.name);
    if (!detectedFormat) {
      setError('Unsupported file format. Please upload CSV, JSON, XML (mRemoteNG), or RDP files.');
      return;
    }

    setFile(selected);
    setFormat(detectedFormat);
    clearError();
    parseFilePreview(selected, detectedFormat);
  };

  const detectFormat = (filename: string): 'CSV' | 'JSON' | 'MREMOTENG' | 'RDP' | null => {
    const lower = filename.toLowerCase();
    if (lower.endsWith('.csv')) return 'CSV';
    if (lower.endsWith('.json')) return 'JSON';
    if (lower.endsWith('.xml')) return 'MREMOTENG';
    if (lower.endsWith('.rdp')) return 'RDP';
    return null;
  };

  const parseFilePreview = async (file: File, fileFormat: string) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      try {
        if (fileFormat === 'JSON') {
          const data = JSON.parse(content);
          const connections = Array.isArray(data) ? data : data.connections || [];
          setPreview(connections.slice(0, 5).map((c: Record<string, unknown>) => ({
            name: String(c.name || 'Unnamed'),
            host: String(c.host || ''),
            port: Number(c.port || 22),
            type: String(c.type || 'SSH'),
            username: c.username as string | undefined,
            folder: c.folderName as string | undefined,
          })));
        } else if (fileFormat === 'CSV') {
          const lines = content.split(/\r?\n/).filter(l => l.trim());
          if (lines.length > 1) {
            const sampleRows = lines.slice(1, 6);
            setPreview(sampleRows.map(row => {
              const values = row.split(',');
              return {
                name: values[0] || 'Unnamed',
                host: values[1] || '',
                port: parseInt(values[2] || '22', 10),
                type: values[3] || 'SSH',
              };
            }));
          }
        } else if (fileFormat === 'RDP') {
          const fullAddressMatch = content.match(/full address:s:(.+)/);
          const hostname = fullAddressMatch ? fullAddressMatch[1].trim() : 'Unknown';
          setPreview([{ name: hostname, host: hostname, port: 3389, type: 'RDP' }]);
        } else {
          setPreview([{ name: 'mRemoteNG import', host: 'Multiple', port: 0, type: 'Mixed' }]);
        }
        setStep(1);
      } catch {
        setError('Failed to parse file. Please check the format.');
      }
    };
    reader.readAsText(file);
  };

  const handleImport = async () => {
    if (!file) return;

    await run(async () => {
      const options: ImportOptions = {
        duplicateStrategy,
        format: format || undefined,
      };

      const res = await importConnections(file, options);
      setResult(res);
      setStep(3);
      notify(`Import complete: ${res.imported} imported, ${res.skipped} skipped, ${res.failed} failed`, 'success');
    }, 'Import failed');
  };

  const handleClose = () => {
    setStep(0);
    setFile(null);
    setFormat(null);
    setPreview([]);
    setResult(null);
    clearError();
    onClose();
  };

  const handleNext = () => {
    if (step === 1 && format === 'CSV') {
      setStep(2);
    } else if (step === 1) {
      setStep(2);
    } else if (step === 2) {
      handleImport();
    }
  };

  const handleBack = () => {
    setStep((prev) => prev - 1);
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Import Connections</DialogTitle>
        <DialogDescription className="sr-only">Import connections from a file</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Import Connections</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={handleClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1">
          <div className="mx-auto max-w-2xl px-6 py-4">
            {/* Stepper */}
            <div className="flex items-center gap-2 mb-4">
              {STEPS.map((label, i) => (
                <div key={label} className="flex items-center gap-2">
                  <div className={`flex items-center justify-center size-7 rounded-full text-xs font-medium ${
                    i <= step ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
                  }`}>
                    {i + 1}
                  </div>
                  <span className={`text-sm ${i <= step ? 'text-foreground' : 'text-muted-foreground'}`}>
                    {label}
                  </span>
                  {i < STEPS.length - 1 && <div className="h-px w-8 bg-border" />}
                </div>
              ))}
            </div>

            {error && (
              <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}

            {step === 0 && (
              <div
                className="flex flex-col items-center justify-center border-2 border-dashed border-border rounded-lg p-8 cursor-pointer hover:border-primary/50 transition-colors"
                onClick={() => fileInputRef.current?.click()}
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".csv,.json,.xml,.rdp"
                  className="hidden"
                  onChange={handleFileSelect}
                />
                <Upload className="size-16 mb-4 text-muted-foreground" />
                <h3 className="text-lg font-medium">Drag &amp; drop or click to browse</h3>
                <p className="text-xs text-muted-foreground mt-1">
                  Supported: CSV, JSON, mRemoteNG XML, RDP
                </p>
              </div>
            )}

            {step === 1 && (
              <div>
                <h3 className="text-lg font-medium mb-2">Preview</h3>
                <p className="text-sm text-muted-foreground mb-2">Detected format: {format}</p>
                {preview.length > 0 && (
                  <div className="rounded-lg border overflow-hidden">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="text-left px-3 py-2 font-medium">Name</th>
                          <th className="text-left px-3 py-2 font-medium">Host</th>
                          <th className="text-left px-3 py-2 font-medium">Port</th>
                          <th className="text-left px-3 py-2 font-medium">Type</th>
                        </tr>
                      </thead>
                      <tbody>
                        {preview.map((conn, i) => (
                          <tr key={i} className="border-b last:border-0">
                            <td className="px-3 py-2">{conn.name}</td>
                            <td className="px-3 py-2">{conn.host}</td>
                            <td className="px-3 py-2">{conn.port}</td>
                            <td className="px-3 py-2">{conn.type}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                {preview.length === 0 && format === 'RDP' && (
                  <p className="text-sm">Single RDP connection detected</p>
                )}
              </div>
            )}

            {step === 2 && (
              <div>
                <h3 className="text-lg font-medium mb-4">Import Options</h3>

                <div className="space-y-2 mb-4">
                  <Label>Duplicate Handling</Label>
                  <Select value={duplicateStrategy} onValueChange={(v) => setDuplicateStrategy(v as 'SKIP' | 'OVERWRITE' | 'RENAME')}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="SKIP">Skip duplicates (keep existing)</SelectItem>
                      <SelectItem value="OVERWRITE">Overwrite existing connections</SelectItem>
                      <SelectItem value="RENAME">Rename new connections (add suffix)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="rounded-md border border-blue-600/50 bg-blue-600/10 px-4 py-3 text-sm text-blue-400">
                  {preview.length} connection(s) will be imported. Folders will be created automatically.
                </div>
              </div>
            )}

            {step === 3 && result && (
              <div>
                {loading ? (
                  <div className="flex flex-col items-center py-8">
                    <Loader2 className="size-8 animate-spin text-muted-foreground" />
                    <p className="mt-4 text-sm">Importing connections...</p>
                  </div>
                ) : (
                  <>
                    <p className="text-sm mb-4">
                      {result.imported} imported, {result.skipped} skipped, {result.failed} failed
                    </p>
                    {result.errors.length > 0 && (
                      <div className="rounded-lg border overflow-hidden">
                        <table className="w-full text-sm">
                          <thead>
                            <tr className="border-b bg-muted/50">
                              <th className="text-left px-3 py-2 font-medium">Row</th>
                              <th className="text-left px-3 py-2 font-medium">Error</th>
                            </tr>
                          </thead>
                          <tbody>
                            {result.errors.slice(0, 10).map((err, i) => (
                              <tr key={i} className="border-b last:border-0">
                                <td className="px-3 py-2">{err.row || 'N/A'}</td>
                                <td className="px-3 py-2">{err.error}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </>
                )}
              </div>
            )}
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          {step > 0 && step < 3 && (
            <Button variant="outline" onClick={handleBack} disabled={loading}>
              Back
            </Button>
          )}
          {step === 0 && (
            <Button variant="outline" onClick={handleClose}>Cancel</Button>
          )}
          {step === 1 && (
            <Button onClick={handleNext}>Next</Button>
          )}
          {step === 2 && (
            <Button onClick={handleNext} disabled={loading}>
              {loading ? 'Importing...' : 'Import'}
            </Button>
          )}
          {step === 3 && (
            <Button onClick={handleClose}>Close</Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
