import { useState, useCallback } from 'react';
import {
  Sparkles, Check, X, RefreshCw, Shield, Send, Maximize2, Loader2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Separator } from '@/components/ui/separator';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';
import {
  optimizeQuery, continueOptimization,
  type OptimizeQueryResult, type DataRequest,
} from '../../api/aiQuery.api';
import { introspectDatabase, type IntrospectionType } from '../../api/database.api';
import { extractApiError } from '../../utils/apiError';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AiQueryOptimizerProps {
  sql: string;
  executionPlan: unknown;
  sessionId: string;
  dbProtocol: string;
  dbVersion?: string;
  schemaContext?: unknown;
  onApply?: (optimizedSql: string) => void;
  onDismiss?: () => void;
}

type Step = 'idle' | 'loading' | 'permissions' | 'result' | 'error';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function AiQueryOptimizer({
  sql, executionPlan, sessionId, dbProtocol, dbVersion, schemaContext,
  onApply, onDismiss,
}: AiQueryOptimizerProps) {
  const [step, setStep] = useState<Step>('idle');
  const [error, setError] = useState('');
  const [result, setResult] = useState<OptimizeQueryResult | null>(null);
  const [dataRequests, setDataRequests] = useState<DataRequest[]>([]);
  const [approvals, setApprovals] = useState<Record<number, boolean>>({});
  const [reviewOpen, setReviewOpen] = useState(false);

  // ---- Step 1: Start optimization ----

  const handleOptimize = useCallback(async () => {
    setStep('loading');
    setError('');
    try {
      const res = await optimizeQuery({
        sql, executionPlan, sessionId, dbProtocol, dbVersion, schemaContext,
      });

      if (res.status === 'needs_data' && res.dataRequests) {
        setResult(res);
        setDataRequests(res.dataRequests);
        // Default all permissions to denied; user must explicitly approve
        const defaultApprovals: Record<number, boolean> = {};
        res.dataRequests.forEach((_, i) => { defaultApprovals[i] = false; });
        setApprovals(defaultApprovals);
        setStep('permissions');
      } else {
        setResult(res);
        setStep('result');
      }
    } catch (err) {
      setError(extractApiError(err, 'Failed to start optimization'));
      setStep('error');
    }
  }, [sql, executionPlan, sessionId, dbProtocol, dbVersion, schemaContext]);

  // ---- Step 2: Submit approved data ----

  const handleSubmitApproved = useCallback(async () => {
    if (!result) return;
    setStep('loading');
    setError('');

    try {
      // Fetch approved introspection data with bounded concurrency
      const approvedData: Record<string, unknown> = {};

      const tasks: Array<() => Promise<void>> = [];
      for (let i = 0; i < dataRequests.length; i++) {
        if (!approvals[i]) continue;
        const req = dataRequests[i];
        if (req.type === 'custom_query') continue;

        const key = `${req.type}_${req.target}`;
        tasks.push(async () => {
          try {
            const introspectionResult = await introspectDatabase(
              sessionId,
              req.type as IntrospectionType,
              req.target,
            );
            approvedData[key] = introspectionResult.data;
          } catch {
            approvedData[key] = { error: 'fetch_failed' };
          }
        });
      }

      // Run with bounded concurrency (max 3 parallel requests)
      const concurrency = 3;
      for (let start = 0; start < tasks.length; start += concurrency) {
        await Promise.all(tasks.slice(start, start + concurrency).map((t) => t()));
      }

      const res = await continueOptimization(result.conversationId, approvedData);
      setResult(res);
      setStep('result');
    } catch (err) {
      setError(extractApiError(err, 'Failed to continue optimization'));
      setStep('error');
    }
  }, [result, dataRequests, approvals, sessionId]);

  // ---- Render: Idle ----

  if (step === 'idle') {
    return (
      <div className="flex justify-center py-4">
        <Button onClick={handleOptimize} size="sm">
          <Sparkles className="size-4" />
          Optimize with AI
        </Button>
      </div>
    );
  }

  // ---- Render: Loading ----

  if (step === 'loading') {
    return (
      <div className="flex items-center justify-center gap-3 py-6">
        <Loader2 className="size-5 animate-spin" />
        <span className="text-sm text-muted-foreground">Analyzing query...</span>
      </div>
    );
  }

  // ---- Render: Error ----

  if (step === 'error') {
    return (
      <div className="py-4">
        <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 mb-2">
          {error}
        </div>
        <Button variant="ghost" size="sm" onClick={handleOptimize}>
          <RefreshCw className="size-4" />
          Retry
        </Button>
      </div>
    );
  }

  // ---- Render: Permission requests ----

  if (step === 'permissions') {
    const approvedCount = Object.values(approvals).filter(Boolean).length;

    return (
      <div className="py-2">
        <h4 className="text-sm font-semibold mb-1 flex items-center gap-1">
          <Shield className="size-4 text-yellow-400" />
          AI needs additional data ({approvedCount}/{dataRequests.length} approved)
        </h4>
        <p className="text-sm text-muted-foreground mb-3">
          Review and approve the data the AI needs to analyze your query:
        </p>

        <div className="flex flex-col gap-2 mb-4">
          {dataRequests.map((req, i) => (
            <div
              key={i}
              className={cn(
                'rounded-lg border border-border p-3',
                approvals[i] && 'bg-accent/50',
              )}
            >
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <Badge variant="outline" className="capitalize h-5">
                      {req.type.replace(/_/g, ' ')}
                    </Badge>
                    <span className="text-sm font-semibold">{req.target}</span>
                  </div>
                  <span className="text-xs text-muted-foreground">{req.reason}</span>
                </div>
                <div className="flex items-center gap-2 ml-4">
                  <span className="text-xs text-muted-foreground">
                    {approvals[i] ? 'Allow' : 'Deny'}
                  </span>
                  <Switch
                    checked={approvals[i] ?? false}
                    onCheckedChange={(checked) => setApprovals((prev) => ({ ...prev, [i]: checked }))}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>

        <div className="flex gap-2">
          <Button
            size="sm"
            onClick={handleSubmitApproved}
            disabled={approvedCount === 0}
          >
            <Send className="size-4" />
            Submit approved ({approvedCount})
          </Button>
          <Button variant="ghost" size="sm" onClick={onDismiss}>Cancel</Button>
        </div>
      </div>
    );
  }

  // ---- Render: Result ----

  if (step === 'result' && result) {
    return (
      <div className="py-2">
        <h4 className="text-sm font-semibold mb-2">AI Optimization Result</h4>

        {result.explanation && (
          <div className="border border-border rounded-lg p-3 mb-3 bg-accent/30">
            <p className="text-sm whitespace-pre-wrap">
              {result.explanation}
            </p>
          </div>
        )}

        {result.changes && result.changes.length > 0 && (
          <div className="mb-3">
            <span className="text-xs font-semibold text-muted-foreground">
              Changes:
            </span>
            <div className="flex flex-wrap gap-1 mt-1">
              {result.changes.map((change, i) => (
                <Badge key={i} variant="secondary">{change}</Badge>
              ))}
            </div>
          </div>
        )}

        {result.optimizedSql && result.optimizedSql !== sql && (
          <>
            <Separator className="my-3" />
            <div
              className="grid grid-cols-2 gap-3 cursor-pointer hover:opacity-90"
              onClick={() => setReviewOpen(true)}
            >
              <div>
                <span className="text-xs font-semibold text-muted-foreground">Original</span>
                <div className="mt-1 p-2 bg-red-900/40 text-red-200 rounded font-mono text-[0.8rem] whitespace-pre-wrap break-all max-h-[200px] overflow-auto opacity-85">
                  {sql}
                </div>
              </div>
              <div>
                <span className="text-xs font-semibold text-muted-foreground">Optimized</span>
                <div className="mt-1 p-2 bg-green-900/40 text-green-200 rounded font-mono text-[0.8rem] whitespace-pre-wrap break-all max-h-[200px] overflow-auto">
                  {result.optimizedSql}
                </div>
              </div>
            </div>
            <span className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
              <Maximize2 className="size-3.5" /> Click to review and accept changes
            </span>
          </>
        )}

        <div className="flex gap-2 pt-3">
          {result.optimizedSql && result.optimizedSql !== sql && (
            <Button size="sm" onClick={() => setReviewOpen(true)}>
              <Check className="size-4" />
              Review & Apply
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={handleOptimize}>
            <RefreshCw className="size-4" />
            Re-optimize
          </Button>
          <Button variant="ghost" size="sm" onClick={onDismiss}>
            <X className="size-4" />
            Dismiss
          </Button>
        </div>

        {/* Review & Accept dialog */}
        <Dialog open={reviewOpen} onOpenChange={setReviewOpen}>
          <DialogContent className="max-w-4xl max-h-[85vh] p-0 gap-0">
            <DialogHeader className="px-6 py-3 flex-row items-center gap-2">
              <Sparkles className="size-5 text-primary" />
              <DialogTitle>Review AI Optimization</DialogTitle>
            </DialogHeader>

            {/* Changes summary */}
            {result.changes && result.changes.length > 0 && (
              <div className="px-6 py-3 bg-accent/30">
                <span className="text-xs font-semibold text-muted-foreground">Changes:</span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {result.changes.map((change, i) => (
                    <Badge key={i} variant="secondary">{change}</Badge>
                  ))}
                </div>
              </div>
            )}

            {/* Side-by-side SQL */}
            <div className="grid grid-cols-2 min-h-[300px] border-t border-border">
              <div className="border-r border-border">
                <div className="px-4 py-2 bg-red-900/60 text-red-200">
                  <span className="text-sm font-semibold">Original Query</span>
                </div>
                <div className="p-4 font-mono text-[0.85rem] whitespace-pre-wrap break-all overflow-auto max-h-[calc(85vh-250px)] bg-[#111] text-gray-300 leading-relaxed">
                  {sql}
                </div>
              </div>
              <div>
                <div className="px-4 py-2 bg-green-900/60 text-green-200">
                  <span className="text-sm font-semibold">Optimized Query</span>
                </div>
                <div className="p-4 font-mono text-[0.85rem] whitespace-pre-wrap break-all overflow-auto max-h-[calc(85vh-250px)] bg-[#111] text-gray-100 leading-relaxed">
                  {result.optimizedSql}
                </div>
              </div>
            </div>

            <DialogFooter className="px-6 py-3 border-t border-border">
              <Button variant="ghost" onClick={() => setReviewOpen(false)}>
                Cancel
              </Button>
              <Button
                className="bg-green-600 hover:bg-green-700 text-white"
                onClick={() => {
                  onApply?.(result.optimizedSql ?? '');
                  setReviewOpen(false);
                }}
              >
                <Check className="size-4" />
                Accept & Apply to Editor
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    );
  }

  return null;
}
