interface Status { connected: boolean; tunnelIP: string }

interface Props {
  status: Status;
  userID: string;
  loading: boolean;
  error: string;
  onUserIDChange: (v: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
}

export default function ConnectionCard({
  status, userID, loading, error, onUserIDChange, onConnect, onDisconnect
}: Props) {
  return (
    <div
      className="relative w-80 rounded-2xl p-8 flex flex-col gap-5"
      style={{
        background: 'rgba(10, 10, 30, 0.72)',
        backdropFilter: 'blur(24px)',
        border: '1px solid rgba(99, 102, 241, 0.25)',
        boxShadow: '0 0 60px rgba(99, 102, 241, 0.08), inset 0 1px 0 rgba(255,255,255,0.06)',
      }}
    >
      {/* Logo / title */}
      <div className="flex flex-col items-center gap-1">
        <span className="text-indigo-400 text-xs font-semibold tracking-widest uppercase">
          VPN Client
        </span>
        <h1 className="text-white text-2xl font-bold tracking-tight">SecureNode</h1>
      </div>

      {/* Status pill */}
      <div className="flex justify-center">
        <div
          className={`flex items-center gap-2 px-4 py-1.5 rounded-full text-sm font-medium transition-all duration-500 ${
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

      {/* User ID input */}
      {!status.connected && (
        <input
          className="bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/60 focus:bg-white/8 transition"
          placeholder="User ID"
          value={userID}
          onChange={e => onUserIDChange(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && onConnect()}
          disabled={loading}
        />
      )}

      {/* Action button */}
      {status.connected ? (
        <button
          onClick={onDisconnect}
          disabled={loading}
          className="rounded-lg py-2.5 text-sm font-semibold transition-all duration-200 disabled:opacity-40"
          style={{
            background: loading ? 'rgba(239,68,68,0.3)' : 'rgba(239,68,68,0.15)',
            border: '1px solid rgba(239,68,68,0.35)',
            color: '#f87171',
          }}
          onMouseEnter={e => { if (!loading) (e.target as HTMLElement).style.background = 'rgba(239,68,68,0.25)' }}
          onMouseLeave={e => { (e.target as HTMLElement).style.background = loading ? 'rgba(239,68,68,0.3)' : 'rgba(239,68,68,0.15)' }}
        >
          {loading ? 'Disconnecting…' : 'Disconnect'}
        </button>
      ) : (
        <button
          onClick={onConnect}
          disabled={loading}
          className="rounded-lg py-2.5 text-sm font-semibold transition-all duration-200 disabled:opacity-40"
          style={{
            background: loading ? 'rgba(99,102,241,0.3)' : 'rgba(99,102,241,0.15)',
            border: '1px solid rgba(99,102,241,0.4)',
            color: '#a5b4fc',
          }}
          onMouseEnter={e => { if (!loading) (e.target as HTMLElement).style.background = 'rgba(99,102,241,0.28)' }}
          onMouseLeave={e => { (e.target as HTMLElement).style.background = loading ? 'rgba(99,102,241,0.3)' : 'rgba(99,102,241,0.15)' }}
        >
          {loading ? 'Connecting…' : 'Connect'}
        </button>
      )}

      {error && (
        <p className="text-red-400/80 text-xs text-center -mt-2">{error}</p>
      )}
    </div>
  );
}
