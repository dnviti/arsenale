import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface SessionTimeoutConfigProps {
  value: string;
  onChange: (value: string) => void;
}

export default function SessionTimeoutConfig({ value, onChange }: SessionTimeoutConfigProps) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor="session-timeout">Session Inactivity Timeout (minutes)</Label>
      <Input
        id="session-timeout"
        type="number"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        min={1}
        max={1440}
      />
      <p className="text-xs text-muted-foreground">
        Idle sessions will be automatically closed after this period (1-1440 minutes)
      </p>
    </div>
  );
}
