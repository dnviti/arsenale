import { useEffect, useRef, useState, useCallback } from 'react';
import * as Guacamole from '@glokon/guacamole-common-js';
import { Pause, Play, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { getRecordingStreamUrl } from '../../api/recordings.api';
import { useAuthStore } from '../../store/authStore';

interface GuacPlayerProps {
  recordingId: string;
  onError?: (message: string) => void;
}

export default function GuacPlayer({ recordingId, onError }: GuacPlayerProps) {
  const displayRef = useRef<HTMLDivElement>(null);
  const recordingRef = useRef<Guacamole.SessionRecording | null>(null);
  const displayInstanceRef = useRef<Guacamole.Display | null>(null);

  const [playing, setPlaying] = useState(false);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const [speed, setSpeed] = useState(1);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    const url = getRecordingStreamUrl(recordingId);
    const token = useAuthStore.getState().accessToken;

    const tunnel = new Guacamole.StaticHTTPTunnel(url, false, {
      'Authorization': `Bearer ${token}`,
    });
    const recording = new Guacamole.SessionRecording(tunnel);
    const display = recording.getDisplay();

    recordingRef.current = recording;
    displayInstanceRef.current = display;

    if (displayRef.current) {
      displayRef.current.innerHTML = '';
      displayRef.current.appendChild(display.getElement());
    }

    const scaleToFit = () => {
      if (!displayRef.current) return;
      const containerWidth = displayRef.current.clientWidth;
      const containerHeight = displayRef.current.clientHeight;
      const displayWidth = display.getWidth();
      const displayHeight = display.getHeight();
      if (displayWidth > 0 && displayHeight > 0) {
        const scale = Math.min(containerWidth / displayWidth, containerHeight / displayHeight, 1);
        display.scale(scale);
      }
    };

    // Important: Scale display whenever Guacamole reports a new resolution
    // This fixes the black screen issue by recalculating the scale when the video actually has dimensions
    (display as unknown as { onresize: (() => void) | null }).onresize = scaleToFit;

    recording.onload = () => {
      setDuration(recording.getDuration() / 1000);
      setLoaded(true);
      scaleToFit();
      // Deferred scale — display dimensions may be 0 at onload time
      requestAnimationFrame(scaleToFit);
    };

    recording.onplay = () => setPlaying(true);
    recording.onpause = () => setPlaying(false);
    recording.onseek = (pos) => setCurrentTime(pos / 1000);

    recording.onerror = (message: string) => {
      onError?.(message || 'Failed to load recording');
    };

    const ro = new ResizeObserver(() => {
      requestAnimationFrame(scaleToFit);
    });
    if (displayRef.current) {
      ro.observe(displayRef.current);
    }

    recording.connect();

    return () => {
      ro.disconnect();
      recording.disconnect();
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recordingId]);

  const play = useCallback(() => {
    recordingRef.current?.play();
  }, []);

  const pause = useCallback(() => {
    recordingRef.current?.pause();
  }, []);

  const seekTo = useCallback((value: number[]) => {
    const targetMs = value[0] * 1000;
    recordingRef.current?.seek(targetMs);
  }, []);

  const restart = useCallback(() => {
    recordingRef.current?.seek(0, () => {
      recordingRef.current?.pause();
      setCurrentTime(0);
    });
  }, []);

  // Speed control: at non-1x speeds, drive playback via periodic seek
  // (guacamole-common-js 1.6.0 removed the playbackSpeed property)
  useEffect(() => {
    if (!playing || speed === 1 || !recordingRef.current) return;
    const recording = recordingRef.current;
    const stepMs = 100;
    const interval = setInterval(() => {
      const pos = recording.getPosition();
      const dur = recording.getDuration();
      const advance = pos + stepMs * speed;
      if (advance < dur) {
        recording.seek(advance);
      } else {
        recording.seek(dur);
      }
    }, stepMs);
    return () => clearInterval(interval);
  }, [playing, speed]);

  // Periodic time update during playback
  useEffect(() => {
    if (!playing) return;
    const interval = setInterval(() => {
      if (recordingRef.current) {
        setCurrentTime(recordingRef.current.getPosition() / 1000);
      }
    }, 250);
    return () => clearInterval(interval);
  }, [playing]);

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  return (
    <div className="flex flex-col h-full">
      <div
        ref={displayRef}
        className="flex-1 bg-black rounded overflow-hidden flex justify-center items-center"
      />
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
