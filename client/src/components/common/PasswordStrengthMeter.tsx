import { useState, useEffect, useRef, useCallback } from 'react';
import { Box, LinearProgress, Typography } from '@mui/material';
import { zxcvbnAsync, zxcvbnOptions } from '@zxcvbn-ts/core';
import * as zxcvbnCommonPackage from '@zxcvbn-ts/language-common';
import * as zxcvbnEnPackage from '@zxcvbn-ts/language-en';

let optionsLoaded = false;

function ensureOptions() {
  if (optionsLoaded) return;
  zxcvbnOptions.setOptions({
    translations: zxcvbnEnPackage.translations,
    graphs: zxcvbnCommonPackage.adjacencyGraphs,
    dictionary: {
      ...zxcvbnCommonPackage.dictionary,
      ...zxcvbnEnPackage.dictionary,
    },
  });
  optionsLoaded = true;
}

interface PasswordStrengthMeterProps {
  password: string;
  onScoreChange?: (score: number) => void;
}

const SCORE_CONFIG = [
  { label: 'Very Weak', color: 'error' as const, value: 5 },
  { label: 'Weak', color: 'error' as const, value: 25 },
  { label: 'Fair', color: 'warning' as const, value: 50 },
  { label: 'Strong', color: 'info' as const, value: 75 },
  { label: 'Very Strong', color: 'success' as const, value: 100 },
];

export default function PasswordStrengthMeter({ password, onScoreChange }: PasswordStrengthMeterProps) {
  const [score, setScore] = useState(0);
  const [feedback, setFeedback] = useState('');
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const onScoreChangeRef = useRef(onScoreChange);
  useEffect(() => { onScoreChangeRef.current = onScoreChange; });

  const evaluate = useCallback((pwd: string) => {
    clearTimeout(timerRef.current);
    if (!pwd) return;
    timerRef.current = setTimeout(async () => {
      ensureOptions();
      const result = await zxcvbnAsync(pwd);
      setScore(result.score);
      const msg = result.feedback.warning || result.feedback.suggestions[0] || '';
      setFeedback(msg);
      onScoreChangeRef.current?.(result.score);
    }, 300);
  }, []);

  useEffect(() => {
    evaluate(password);
    return () => clearTimeout(timerRef.current);
  }, [password, evaluate]);

  if (!password) return null;

  const config = SCORE_CONFIG[score];

  return (
    <Box sx={{ mt: 0.5, mb: 0.5 }}>
      <LinearProgress
        variant="determinate"
        value={config.value}
        color={config.color}
        sx={{ height: 6, borderRadius: 3 }}
      />
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 0.25 }}>
        <Typography variant="caption" color={`${config.color}.main`}>
          {config.label}
        </Typography>
        {feedback && (
          <Typography variant="caption" color="text.secondary" sx={{ textAlign: 'right', maxWidth: '70%' }}>
            {feedback}
          </Typography>
        )}
      </Box>
    </Box>
  );
}
