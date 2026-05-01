import { useMotionValue } from "motion/react";
import { useState, useEffect, useRef } from "react";
import { EncryptedText } from "@/components/ui/encrypted-text";
import { CardPattern, generateRandomString } from "@/components/ui/evervault-card";

interface Status { connected: boolean; tunnelIP: string }

interface Props {
  status: Status;
  apiKey: string;
  baseURL: string;
  loading: boolean;
  error: string;
  onApiKeyChange: (v: string) => void;
  onBaseURLChange: (v: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
}

const CONTENT_FADE = "transition-opacity duration-[1000ms]";

export default function ConnectionCard({
  status, apiKey, baseURL, loading, error, onApiKeyChange, onBaseURLChange, onConnect, onDisconnect
}: Props) {
  const mouseX = useMotionValue(0);
  const mouseY = useMotionValue(0);
  const [randomString, setRandomString] = useState("");
  const [isCleared, setIsCleared] = useState(false);
  const [inputFocused, setInputFocused] = useState(false);
  const [enterKey, setEnterKey] = useState(0);
  const scrambleRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const leaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setRandomString(generateRandomString(1500));
  }, []);

  // When connected, keep background cleared
  useEffect(() => {
    if (status.connected) {
      if (leaveTimerRef.current) clearTimeout(leaveTimerRef.current);
      if (scrambleRef.current) clearInterval(scrambleRef.current);
      setIsCleared(true);
    }
  }, [status.connected]);

  const showBackground = !isCleared && !status.connected;

  function startScramble(duration: number) {
    if (scrambleRef.current) clearInterval(scrambleRef.current);
    const start = Date.now();
    scrambleRef.current = setInterval(() => {
      setRandomString(generateRandomString(1500));
      if (Date.now() - start > duration) {
        clearInterval(scrambleRef.current!);
        scrambleRef.current = null;
      }
    }, 50);
  }

  function scheduleRestore() {
    if (leaveTimerRef.current) clearTimeout(leaveTimerRef.current);
    leaveTimerRef.current = setTimeout(() => {
      if (!inputFocused && !status.connected) {
        setIsCleared(false);
        startScramble(1500);
      }
    }, 2000);
  }

  function onMouseMove({ currentTarget, clientX, clientY }: React.MouseEvent<HTMLDivElement>) {
    const { left, top } = currentTarget.getBoundingClientRect();
    mouseX.set(clientX - left);
    mouseY.set(clientY - top);
    setRandomString(generateRandomString(1500));
  }

  function onMouseEnter() {
    if (leaveTimerRef.current) clearTimeout(leaveTimerRef.current);
    setIsCleared(true);
    setEnterKey(k => k + 1);
    startScramble(1500);
  }

  function onMouseLeave() {
    if (!inputFocused) scheduleRestore();
  }

  function onCardClick() {
    if (leaveTimerRef.current) clearTimeout(leaveTimerRef.current);
    setIsCleared(true);
  }

  function onInputFocus() {
    if (leaveTimerRef.current) clearTimeout(leaveTimerRef.current);
    setInputFocused(true);
    setIsCleared(true);
  }

  return (
    <div className="relative w-80 pointer-events-auto font-mono" style={{ opacity: 0.9 }}>
      <div
        onMouseMove={onMouseMove}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        onClick={onCardClick}
        className="group/card rounded-2xl w-full relative overflow-hidden flex items-center justify-center"
        style={{
          background: 'rgba(10, 10, 30, 0.72)',
          backdropFilter: 'blur(24px)',
          boxShadow: '0 0 60px rgba(99, 102, 241, 0.08), inset 0 1px 0 rgba(255,255,255,0.06)',
          isolation: 'isolate',
        }}
      >
        {/* Encrypted background — React-controlled opacity */}
        <div
          className="transition-opacity duration-[1500ms]"
          style={{ opacity: showBackground ? 1 : 0 }}
        >
          <CardPattern mouseX={mouseX} mouseY={mouseY} randomString={randomString} reversed />
        </div>

        {/* Card content */}
        <div className="relative z-10 w-full p-8 flex flex-col gap-5">

          {/* Logo / title */}
          <div className="flex flex-col items-center gap-1">

            {/* VPN Client — text-xs matches evervault background size exactly */}
            <span
              className={`text-xs tracking-widest uppercase ${CONTENT_FADE}`}
              style={{ opacity: isCleared ? 1 : 0 }}
            >
              <EncryptedText
                text="VPN Client"
                encryptedClassName="text-indigo-300/60"
                revealedClassName="text-indigo-200"
                revealDelayMs={30}
                trigger={enterKey}
              />
            </span>

            {/* SecureNode — fades in/out; evervault background fills the space when hidden */}
            <span
              className={`text-base font-bold ${CONTENT_FADE}`}
              style={{ opacity: isCleared ? 1 : 0 }}
            >
              <EncryptedText
                text="SecureNode"
                encryptedClassName="text-indigo-300/60"
                revealedClassName="text-indigo-100"
                revealDelayMs={40}
                trigger={enterKey}
              />
            </span>
          </div>

          {/* Status pill */}
          <div
            className={`flex justify-center ${CONTENT_FADE}`}
            style={{ opacity: isCleared ? 1 : 0 }}
          >
            <div
              className={`flex items-center gap-2 px-4 py-1.5 rounded-full text-sm font-medium ${
                status.connected
                  ? 'bg-green-500/15 text-green-400 border border-green-500/30'
                  : 'bg-red-500/15 text-red-400 border border-red-500/30'
              }`}
            >
              <span
                className={`w-2 h-2 rounded-full ${status.connected ? 'bg-green-400 animate-pulse' : 'bg-red-500'}`}
              />
              {status.connected ? `Connected · ${status.tunnelIP}` : 'Disconnected'}
            </div>
          </div>

          {/* Connection inputs */}
          {!status.connected && (
            <div
              className={`flex flex-col gap-2 ${CONTENT_FADE}`}
              style={{ opacity: isCleared ? 1 : 0 }}
            >
              <input
                className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/60 focus:bg-white/8 transition"
                placeholder="Node URL"
                value={baseURL}
                onChange={e => onBaseURLChange(e.target.value)}
                onFocus={onInputFocus}
                disabled={loading}
              />
              <input
                type="password"
                className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/60 focus:bg-white/8 transition"
                placeholder="API Key"
                value={apiKey}
                onChange={e => onApiKeyChange(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && onConnect()}
                onFocus={onInputFocus}
                disabled={loading}
              />
            </div>
          )}

          {/* Action button */}
          {status.connected ? (
            <button
              onClick={onDisconnect}
              disabled={loading}
              className="btn-disconnect rounded-lg py-2.5 text-sm font-semibold"
              style={{ opacity: isCleared ? (loading ? 0.4 : 1) : 0 }}
            >
              {loading ? 'Disconnecting…' : 'Disconnect'}
            </button>
          ) : (
            <button
              onClick={onConnect}
              disabled={loading}
              className="btn-connect rounded-lg py-2.5 text-sm font-semibold"
              style={{ opacity: isCleared ? (loading ? 0.4 : 0.8) : 0 }}
            >
              {loading ? 'Connecting…' : 'Connect'}
            </button>
          )}

          {error && (
            <p className="text-red-400/80 text-xs text-center -mt-2">{error}</p>
          )}
        </div>
      </div>
    </div>
  );
}
