import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { Pause, Play, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { getRecordingStreamUrl } from '../../api/recordings.api';
import { useAuthStore } from '../../store/authStore';
import '@xterm/xterm/css/xterm.css';

interface AsciicastHeader {
  version: number;
  width: number;
  height: number;
}

type AsciicastEvent = [number, string, string]; // [time, type, data]

interface SshPlayerProps {
  recordingId: string;
  onError?: (message: string) => void;
}

export default function SshPlayer({ recordingId, onError }: SshPlayerProps) {
  const termContainerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const roRef = useRef<ResizeObserver | null>(null);
  const eventsRef = useRef<AsciicastEvent[]>([]);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const indexRef = useRef(0);
  const startTimeRef = useRef(0);

  const [playing, setPlaying] = useState(false);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const [speed, setSpeed] = useState(1);
  const [loaded, setLoaded] = useState(false);

  // Parse asciicast v2 data
  const parseAsciicast = useCallback((text: string): { header: AsciicastHeader; events: AsciicastEvent[] } => {
    const lines = text.trim().split('\n');
    const header = JSON.parse(lines[0]) as AsciicastHeader;
    const events: AsciicastEvent[] = [];
    for (let i = 1; i < lines.length; i++) {
      try {
        const parsed = JSON.parse(lines[i]) as AsciicastEvent;
        if (parsed[1] === 'o') events.push(parsed); // only output events
      } catch { /* skip malformed lines */ }
    }
    return { header, events };
  }, []);

  // Load recording data
  useEffect(() => {
    let cancelled = false;
    const url = getRecordingStreamUrl(recordingId);
    const token = useAuthStore.getState().accessToken;
    fetch(url, {
      headers: { 'Authorization': `Bearer ${token}` },
    })
      .then((res) => {
        if (!res.ok) throw new Error('Failed to load recording');
        return res.text();
      })
      .then((text) => {
        if (cancelled) return;
        const { header, events } = parseAsciicast(text);
        eventsRef.current = events;

        // Size terminal to fit container instead of using recording dimensions
        const CHAR_W = 9.05;
        const CHAR_H = 17;
        const container = termContainerRef.current;
        const cols = container
          ? Math.max(Math.floor(container.clientWidth / CHAR_W), header.width || 80)
          : (header.width || 80);
        const rows = container
          ? Math.floor(container.clientHeight / CHAR_H)
          : (header.height || 24);

        const term = new Terminal({
          cols,
          rows,
          disableStdin: true,
          cursorBlink: false,
          scrollback: 5000,
          convertEol: true,
          theme: { background: '#1e1e1e' },
        });

        if (termRef.current) {
          term.open(termRef.current);
        }

        terminalRef.current = term;

        // Re-fit on container resize (e.g. fullscreen toggle)
        if (container) {
          const ro = new ResizeObserver(() => {
            if (!termContainerRef.current || !terminalRef.current) return;
            const newCols = Math.max(Math.floor(termContainerRef.current.clientWidth / CHAR_W), 80);
            const newRows = Math.floor(termContainerRef.current.clientHeight / CHAR_H);
            if (newCols > 0 && newRows > 0) {
              terminalRef.current.resize(newCols, newRows);
            }
          });
          ro.observe(container);
          roRef.current = ro;
        }

        if (events.length > 0) {
          setDuration(events[events.length - 1][0]);
        }
        setLoaded(true);
      })
      .catch((err) => {
        if (!cancelled) onError?.(err.message);
      });

    return () => {
      cancelled = true;
      if (timerRef.current) clearTimeout(timerRef.current);
      terminalRef.current?.dispose();
      roRef.current?.disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recordingId]);

  // Playback engine
  const scheduleNext = useCallback(() => {
    const events = eventsRef.current;
    const idx = indexRef.current;
    if (idx >= events.length) {
      setPlaying(false);
      return;
    }

    const event = events[idx];
    const elapsed = (Date.now() - startTimeRef.current) / 1000;
    const delay = Math.max(0, (event[0] - elapsed) * (1000 / speed));

    timerRef.current = setTimeout(() => {
      terminalRef.current?.write(event[2]);
      setCurrentTime(event[0]);
      indexRef.current = idx + 1;
      scheduleNext();
    }, delay);
  }, [speed]);

  const play = useCallback(() => {
    if (!loaded) return;
    const events = eventsRef.current;
    if (indexRef.current >= events.length) {
      // Restart from beginning
      indexRef.current = 0;
      terminalRef.current?.reset();
    }
    const currentEventTime = indexRef.current > 0 ? events[indexRef.current - 1]?.[0] ?? 0 : 0;
    startTimeRef.current = Date.now() - currentEventTime * 1000;
    setPlaying(true);
    scheduleNext();
  }, [loaded, scheduleNext]);

  const pause = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    setPlaying(false);
  }, []);

  const seekTo = useCallback((value: number[]) => {
    const targetTime = value[0];
    if (timerRef.current) clearTimeout(timerRef.current);

    terminalRef.current?.reset();
    const events = eventsRef.current;

    // Replay all events up to target time
    let i = 0;
    for (; i < events.length && events[i][0] <= targetTime; i++) {
      terminalRef.current?.write(events[i][2]);
    }
    indexRef.current = i;
    setCurrentTime(targetTime);

    if (playing) {
      startTimeRef.current = Date.now() - targetTime * 1000;
      scheduleNext();
    }
  }, [playing, scheduleNext]);

  const restart = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    terminalRef.current?.reset();
    indexRef.current = 0;
    setCurrentTime(0);
    setPlaying(false);
  }, []);

  // Handle speed changes during playback
  useEffect(() => {
    if (playing) {
      if (timerRef.current) clearTimeout(timerRef.current);
      const currentEventTime = eventsRef.current[indexRef.current - 1]?.[0] ?? 0;
      startTimeRef.current = Date.now() - currentEventTime * 1000;
      scheduleNext();
    }
  }, [speed, playing, scheduleNext]);

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div
        ref={termContainerRef}
        className="flex-1 bg-[#1e1e1e] rounded overflow-hidden"
      >
        <div
          ref={termRef}
          className="[&_.xterm-viewport]:!bg-[#1e1e1e]"
        />
      </div>
      <div className="flex items-center gap-2 mt-2 px-2">
        {playing ? (
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={pause}>
            <Pause className="h-4 w-4" />
          </Button>
        ) : (
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={play} disabled={!loaded}>
            <Play className="h-4 w-4" />
          </Button>
        )}
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={restart} disabled={!loaded}>
          <RotateCcw className="h-4 w-4" />
        </Button>
        <span className="text-xs min-w-[40px]">{formatTime(currentTime)}</span>
        <Slider
          value={[currentTime]}
          max={duration || 1}
          step={0.1}
          onValueChange={seekTo}
          className="flex-1"
        />
        <span className="text-xs min-w-[40px]">{formatTime(duration)}</span>
        <Select value={String(speed)} onValueChange={(v) => setSpeed(Number(v))}>
          <SelectTrigger className="h-7 w-[70px] text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="0.5">0.5x</SelectItem>
            <SelectItem value="1">1x</SelectItem>
            <SelectItem value="2">2x</SelectItem>
            <SelectItem value="4">4x</SelectItem>
            <SelectItem value="8">8x</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
