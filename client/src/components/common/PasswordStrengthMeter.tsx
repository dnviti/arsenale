import { useEffect, useRef, useState } from 'react';
import { zxcvbnAsync, zxcvbnOptions } from '@zxcvbn-ts/core';
import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

let optionsLoaded = false;
let optionsPromise: Promise<void> | null = null;

async function ensureOptionsLoaded() {
  if (optionsLoaded) return;
  if (!optionsPromise) {
    optionsPromise = Promise.all([
      import('@zxcvbn-ts/language-common'),
      import('@zxcvbn-ts/language-en'),
    ])
      .then(([zxcvbnCommonPackage, zxcvbnEnPackage]) => {
        zxcvbnOptions.setOptions({
          translations: zxcvbnEnPackage.translations,
          graphs: zxcvbnCommonPackage.adjacencyGraphs,
          dictionary: {
            ...zxcvbnCommonPackage.dictionary,
            ...zxcvbnEnPackage.dictionary,
          },
        });
        optionsLoaded = true;
      })
      .finally(() => {
        if (!optionsLoaded) {
          optionsPromise = null;
        }
      });
  }

  await optionsPromise;
}

interface PasswordStrengthMeterProps {
  password: string;
  onScoreChange?: (score: number) => void;
}

const SCORE_CONFIG = [
  { label: 'Very Weak', value: 5, indicatorClassName: 'bg-destructive', textClassName: 'text-destructive' },
  { label: 'Weak', value: 25, indicatorClassName: 'bg-destructive', textClassName: 'text-destructive' },
  { label: 'Fair', value: 50, indicatorClassName: 'bg-chart-5', textClassName: 'text-foreground' },
  { label: 'Strong', value: 75, indicatorClassName: 'bg-primary', textClassName: 'text-primary' },
  { label: 'Very Strong', value: 100, indicatorClassName: 'bg-primary', textClassName: 'text-primary' },
] as const;

export default function PasswordStrengthMeter({
  password,
  onScoreChange,
}: PasswordStrengthMeterProps) {
  const [score, setScore] = useState(0);
  const [feedback, setFeedback] = useState('');
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const onScoreChangeRef = useRef(onScoreChange);

  useEffect(() => {
    onScoreChangeRef.current = onScoreChange;
  });

  useEffect(() => {
    clearTimeout(timerRef.current);

    timerRef.current = setTimeout(async () => {
      if (!password) {
        setScore(0);
        setFeedback('');
        onScoreChangeRef.current?.(0);
        return;
      }

      try {
        await ensureOptionsLoaded();
        const result = await zxcvbnAsync(password);
        setScore(result.score);
        setFeedback(result.feedback.warning || result.feedback.suggestions[0] || '');
        onScoreChangeRef.current?.(result.score);
      } catch {
        setScore(0);
        setFeedback('');
        onScoreChangeRef.current?.(0);
      }
    }, 300);

    return () => clearTimeout(timerRef.current);
  }, [password]);

  if (!password) return null;

  const config = SCORE_CONFIG[score];

  return (
    <div className="space-y-2">
      <Progress value={config.value} indicatorClassName={config.indicatorClassName} />
      <div className="flex flex-wrap items-start justify-between gap-2 text-xs">
        <span className={cn('font-medium', config.textClassName)}>
          {config.label}
        </span>
        {feedback && (
          <span className="max-w-[70%] text-right leading-5 text-muted-foreground">
            {feedback}
          </span>
        )}
      </div>
    </div>
  );
}
