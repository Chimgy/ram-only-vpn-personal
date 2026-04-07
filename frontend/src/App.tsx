import { useState, useRef } from 'react'

const BACKEND = 'http://localhost:3000'
const VPN_API = 'http://192.168.1.108:8080'

type View = 'connect' | 'register' | 'registered' | 'vpn' | 'config'

interface Status {
  msg: string
  error: boolean
}

interface WgConfig {
  conf: string
  tunnelIp: string
}

export default function App() {
  const [view, setView] = useState<View>('connect')
  const [status, setStatus] = useState<Status | null>(null)
  const [generatedId, setGeneratedId] = useState('')
  const [userId, setUserId] = useState('')
  const [copyLabel, setCopyLabel] = useState('Copy ID')
  const [copyConfLabel, setCopyConfLabel] = useState('Copy Config')
  const [wgConfig, setWgConfig] = useState<WgConfig | null>(null)
  const [disconnectStatus, setDisconnectStatus] = useState<Status | null>(null)
  const [activePubkey, setActivePubkey] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const pubkeyRef = useRef<HTMLTextAreaElement>(null)
  const disconnectRef = useRef<HTMLTextAreaElement>(null)

  function showStatus(msg: string, error = false) {
    setStatus({ msg, error })
  }

  function goTo(v: View) {
    setStatus(null)
    setView(v)
  }

  function handleConnect() {
    const id = inputRef.current?.value.trim() ?? ''
    if (!/^\d{16}$/.test(id)) {
      showStatus('ID must be exactly 16 digits.', true)
      return
    }
    setUserId(id)
    goTo('vpn')
  }

  async function handleRegister() {
    showStatus('Registering…')
    try {
      const res = await fetch(`${BACKEND}/auth/register`, { method: 'POST' })
      if (!res.ok) throw new Error(`Server error ${res.status}`)
      const data = (await res.json()) as { user_id: string }
      setGeneratedId(data.user_id)
      goTo('registered')
    } catch {
      showStatus('Registration failed. Try again.', true)
    }
  }

  async function handleGetConfig() {
    const pubkey = pubkeyRef.current?.value.trim() ?? ''
    if (!pubkey) {
      showStatus('Public key required.', true)
      return
    }
    showStatus('Contacting VPN node…')
    try {
      const res = await fetch(`${VPN_API}/peer`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ public_key: pubkey, user_id: userId }),
      })
      const data = await res.json()
      if (!res.ok) {
        showStatus(data.error ?? 'Request failed.', true)
        return
      }
      const conf = `[Interface]
PrivateKey = <your private key>
Address = ${data.tunnel_ip}/24

[Peer]
PublicKey = ${data.server_pubkey}
Endpoint = ${data.server_endpoint}
AllowedIPs = 0.0.0.0/0`
      setActivePubkey(pubkey)
      setWgConfig({ conf, tunnelIp: data.tunnel_ip })
      setDisconnectStatus(null)
      goTo('config')
    } catch {
      showStatus('Could not reach VPN node — is it online?', true)
    }
  }

  async function handleDisconnect() {
    const pubkey = disconnectRef.current?.value.trim() ?? ''
    if (!pubkey) {
      setDisconnectStatus({ msg: 'Public key required.', error: true })
      return
    }
    setDisconnectStatus({ msg: 'Disconnecting…', error: false })
    try {
      const res = await fetch(`${VPN_API}/peer`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ public_key: pubkey }),
      })
      const data = await res.json()
      if (!res.ok) {
        setDisconnectStatus({ msg: data.error ?? 'Request failed.', error: true })
        return
      }
      setDisconnectStatus({ msg: 'Disconnected.', error: false })
      setWgConfig(null)
    } catch {
      setDisconnectStatus({ msg: 'Could not reach VPN node — is it online?', error: true })
    }
  }

  function handleCopyId() {
    navigator.clipboard.writeText(generatedId)
    setCopyLabel('Copied!')
    setTimeout(() => setCopyLabel('Copy ID'), 2000)
  }

  function handleCopyConf() {
    if (!wgConfig) return
    navigator.clipboard.writeText(wgConfig.conf)
    setCopyConfLabel('Copied!')
    setTimeout(() => setCopyConfLabel('Copy Config'), 2000)
  }

  return (
    <div className="card">
      <div className={`logo-mark${view === 'registered' || view === 'config' ? ' success' : ''}`} />

      {view === 'connect' && (
        <>
          <h1>RAM<span className="accent">VPN</span></h1>
          <p className="subtitle">Zero-persistence. Every reboot is a clean slate.</p>
          <div className="form">
            <input
              ref={inputRef}
              type="text"
              inputMode="numeric"
              maxLength={16}
              placeholder="Enter your 16-digit ID"
              autoComplete="off"
              spellCheck={false}
              onKeyDown={e => e.key === 'Enter' && handleConnect()}
            />
            <button className="btn-primary" onClick={handleConnect}>Connect</button>
          </div>
          {status && <p className={`status ${status.error ? 'error' : 'info'}`}>{status.msg}</p>}
          <button className="btn-ghost" onClick={() => goTo('register')}>
            No account? Register
          </button>
        </>
      )}

      {view === 'register' && (
        <>
          <h1>RAM<span className="accent">VPN</span></h1>
          <p className="subtitle">Create an account. No email. No name. Just a number.</p>
          <button className="btn-primary" onClick={handleRegister}>Generate Account</button>
          {status && <p className={`status ${status.error ? 'error' : 'info'}`}>{status.msg}</p>}
          <button className="btn-ghost" onClick={() => goTo('connect')}>
            Already have an ID? Connect
          </button>
        </>
      )}

      {view === 'registered' && (
        <>
          <h1>Account <span className="accent">Created</span></h1>
          <p className="subtitle">This is your ID. It is your only credential — save it now.</p>
          <div className="id-display">{generatedId}</div>
          <button className="btn-secondary" onClick={handleCopyId}>{copyLabel}</button>
          <p className="hint">There is no recovery. If you lose this, your account is gone.</p>
          <button className="btn-primary" style={{ marginTop: 8 }} onClick={() => {
            setUserId(generatedId)
            goTo('vpn')
          }}>
            Connect Now
          </button>
        </>
      )}

      {view === 'vpn' && (
        <>
          <h1>Get <span className="accent">Config</span></h1>
          <p className="subtitle">Paste your WireGuard public key to receive your tunnel config.</p>
          <div className="form">
            <textarea
              ref={pubkeyRef}
              className="pubkey-input"
              rows={3}
              placeholder="Paste your WireGuard public key"
              spellCheck={false}
              autoComplete="off"
            />
            <button className="btn-primary" onClick={handleGetConfig}>Generate Config</button>
          </div>
          {status && <p className={`status ${status.error ? 'error' : 'info'}`}>{status.msg}</p>}
          <button className="btn-ghost" onClick={() => goTo('connect')}>
            ← Back
          </button>
        </>
      )}

      {view === 'config' && wgConfig && (
        <>
          <h1>Tunnel <span className="accent">Ready</span></h1>
          <p className="subtitle">
            Your tunnel IP is <span className="accent">{wgConfig.tunnelIp}</span>. Replace the placeholder with your private key.
          </p>
          <pre className="conf-box">{wgConfig.conf}</pre>
          <button className="btn-secondary" onClick={handleCopyConf}>{copyConfLabel}</button>
          <p className="hint" style={{ color: 'var(--text)' }}>
            Keep your private key secret — never share or paste it here.
          </p>
          <div className="divider" />
          <p className="subtitle" style={{ alignSelf: 'flex-start' }}>Disconnect</p>
          <div className="form">
            <textarea
              ref={disconnectRef}
              className="pubkey-input"
              rows={2}
              defaultValue={activePubkey}
              placeholder="Paste the public key to disconnect"
              spellCheck={false}
              autoComplete="off"
            />
            <button className="btn-danger" onClick={handleDisconnect}>Disconnect</button>
          </div>
          {disconnectStatus && (
            <p className={`status ${disconnectStatus.error ? 'error' : 'info'}`}>
              {disconnectStatus.msg}
            </p>
          )}
          <button className="btn-ghost" onClick={() => goTo('connect')}>
            ← Done
          </button>
        </>
      )}
    </div>
  )
}
