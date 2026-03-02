import { useEffect, useMemo, useState } from "react";

import { clearAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { fetchDevices, revokeDevice } from "@/features/settings/settingsApi";
import { useEncryption } from "@/shared/hooks/useEncryption";
import { keyStore } from "@/shared/lib/keystore";
import type { Device } from "@/shared/types";

type DeviceView = Device & { current?: boolean };

export default function SecuritySettings() {
  const dispatch = useAppDispatch();
  const accessToken = useAppSelector((state) => state.auth.accessToken);
  const keyPair = useAppSelector((state) => state.auth.keyPair);
  const [devices, setDevices] = useState<DeviceView[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [importValue, setImportValue] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { clearStoredKeys } = useEncryption();

  const publicKeyPreview = useMemo(
    () => keyPair?.publicKey ?? "No key loaded",
    [keyPair?.publicKey]
  );

  useEffect(() => {
    if (!accessToken) {
      return;
    }

    let cancelled = false;
    setLoadingDevices(true);
    void (async () => {
      try {
        const response = await fetchDevices(accessToken);
        if (!cancelled) {
          setDevices(response.devices);
        }
      } catch (nextError) {
        if (!cancelled) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load devices.");
        }
      } finally {
        if (!cancelled) {
          setLoadingDevices(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [accessToken]);

  async function exportPrivateKey() {
    if (!keyPair) {
      return;
    }

    await navigator.clipboard.writeText(keyPair.privateKey);
    setNotice("Private key copied.");
  }

  async function exportPublicKey() {
    if (!keyPair) {
      return;
    }

    await navigator.clipboard.writeText(keyPair.publicKey);
    setNotice("Public key copied.");
  }

  async function importPrivateKey() {
    const trimmed = importValue.trim();
    if (!trimmed || !keyPair?.publicKey) {
      setNotice("Load or generate a keypair first.");
      return;
    }

    await keyStore.saveKeyPair(keyPair.publicKey, trimmed);
    dispatch(
      setKeyPair({
        publicKey: keyPair.publicKey,
        privateKey: trimmed,
      })
    );
    setImportValue("");
    setNotice("Private key imported.");
  }

  async function resetIdentity() {
    await clearStoredKeys();
    dispatch(clearAuth());
  }

  async function handleRevokeDevice(deviceId: string) {
    if (!accessToken) {
      return;
    }

    try {
      await revokeDevice(accessToken, deviceId);
      setDevices((current) => current.filter((device) => device.id !== deviceId));
      setNotice("Device revoked.");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to revoke device.");
    }
  }

  return (
    <div className="settings-panel-stack">
      <section className="settings-panel">
        <p className="eyebrow">Security</p>
        <h2>Identity keys</h2>
        <p className="muted">
          Local key import and export stays in the browser by design. Device sessions are now loaded from the backend auth device API when available.
        </p>

        <div className="settings-field">
          <span>Public key</span>
          <div className="key-preview">{publicKeyPreview}</div>
        </div>

        <div className="settings-actions">
          <button className="primary-button" onClick={() => void exportPublicKey()} type="button">
            Copy public key
          </button>
          <button className="secondary-button" onClick={() => void exportPrivateKey()} type="button">
            Copy private key
          </button>
        </div>

        <div className="settings-field" style={{ marginTop: 18 }}>
          <span>Import private key</span>
          <textarea
            className="message-input"
            onChange={(event) => setImportValue(event.target.value)}
            placeholder="Paste private key to replace the local identity"
            rows={5}
            value={importValue}
          />
        </div>

        <div className="settings-actions">
          <button className="primary-button" onClick={() => void importPrivateKey()} type="button">
            Import key
          </button>
          <button className="secondary-button is-danger" onClick={() => void resetIdentity()} type="button">
            Clear local identity
          </button>
          {notice ? <span className="muted">{notice}</span> : null}
          {error ? <span className="error-text">{error}</span> : null}
        </div>
      </section>

      <section className="settings-panel">
        <p className="eyebrow">Devices</p>
        <h2>Signed-in devices</h2>
        {loadingDevices ? <p className="muted">Loading devices...</p> : null}
        <div className="device-list">
          {devices.map((device) => (
            <article className="device-card" key={device.id}>
              <div>
                <h4>{device.device_name || "Unnamed device"}</h4>
                <p className="muted">
                  {device.platform} · last seen {device.last_seen ? new Date(device.last_seen).toLocaleString() : "unknown"}
                  {device.current ? " · current" : ""}
                </p>
              </div>
              <button
                className="secondary-button"
                disabled={device.current}
                onClick={() => void handleRevokeDevice(device.id)}
                type="button"
              >
                {device.current ? "Current" : "Revoke"}
              </button>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
