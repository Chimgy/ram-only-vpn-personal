import { useState, useEffect, useRef } from 'react';
import { Connect, Disconnect, GetStatus } from '../wailsjs/go/main/App';
import MapRippleBackground from './components/MapRippleBackground';
import ConnectionCard from './components/ConnectionCard';

type Status = { connected: boolean; tunnelIP: string };
type Result = { ok: boolean; tunnelIP: string; error: string };

export default function App() {
  const rootRef = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<Status>({ connected: false, tunnelIP: '' });
  const [apiKey, setApiKey] = useState('');
  const [baseURL, setBaseURL] = useState(() => localStorage.getItem('vpn_base_url') ?? 'http://ramonlyvpn.duckdns.org:8080');
  const [loading, setLoading] = useState(false);
  const [error, setError]   = useState('');

  useEffect(() => { GetStatus().then(setStatus); }, []);

  // Write mouse position straight to CSS vars — no React state, no re-renders
  // requestAnimationFrame keeps updates in sync with the display refresh rate
  useEffect(() => {
    let rafId: number;
    const handler = (e: MouseEvent) => {
      cancelAnimationFrame(rafId);
      rafId = requestAnimationFrame(() => {
        const el = rootRef.current;
        if (!el) return;
        el.style.setProperty('--cx', `${e.clientX}px`);
        el.style.setProperty('--cy', `${e.clientY}px`);
      });
    };
    window.addEventListener('mousemove', handler, { passive: true });
    return () => { window.removeEventListener('mousemove', handler); cancelAnimationFrame(rafId); };
  }, []);

  function handleBaseURLChange(v: string) {
    setBaseURL(v);
    localStorage.setItem('vpn_base_url', v);
  }

  async function handleConnect() {
    if (!apiKey.trim()) { setError('Enter an API key'); return; }
    setLoading(true); setError('');
    const res: Result = await Connect(apiKey.trim(), baseURL.trim());
    if (res.ok) setStatus({ connected: true, tunnelIP: res.tunnelIP });
    else setError(res.error);
    setLoading(false);
  }

  async function handleDisconnect() {
    setLoading(true); setError('');
    const res: Result = await Disconnect();
    if (res.ok) setStatus({ connected: false, tunnelIP: '' });
    else setError(res.error);
    setLoading(false);
  }

  return (
    <div ref={rootRef} className="app-root relative min-h-screen bg-[#05050f] overflow-hidden select-none">
      {/* Blobs — CSS animation, no JS */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none" aria-hidden>
        <div className="blob blob-1" />
        <div className="blob blob-2" />
      </div>

      {/* World map */}
      <MapRippleBackground />


      {/* Card */}
      <div className="relative z-30 min-h-screen flex items-center justify-center pointer-events-none">
        <ConnectionCard
          status={status}
          apiKey={apiKey}
          baseURL={baseURL}
          loading={loading}
          error={error}
          onApiKeyChange={setApiKey}
          onBaseURLChange={handleBaseURLChange}
          onConnect={handleConnect}
          onDisconnect={handleDisconnect}
        />
      </div>
    </div>
  );
}
