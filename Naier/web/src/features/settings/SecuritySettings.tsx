import { useEffect, useMemo, useState } from "react";
import {
  Copy,
  Download,
  ExternalLink,
  KeyRound,
  Shield,
  Smartphone,
  Trash2,
  Upload,
} from "lucide-react";

import { clearAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import {
  approveDevice,
  createPendingDevice,
  exportEncryptedBackup,
  fetchDevices,
  importEncryptedBackup,
  revokeDevice,
} from "@/features/settings/settingsApi";
import {
  decryptKeyBundleBackup,
  encryptKeyBundleBackup,
  generateKeyPair,
  generateSigningKeyPair,
} from "@/shared/lib/crypto";
import { keyStore } from "@/shared/lib/keystore";
import { useEncryption } from "@/shared/hooks/useEncryption";
import type { Device } from "@/shared/types";

type DeviceView = Device & { current?: boolean };

interface PairingPayload {
  device_signing_key: string;
  device_exchange_key: string;
  device_name: string;
  platform: "web";
}

export default function SecuritySettings() {
  const dispatch = useAppDispatch();
  const accessToken = useAppSelector((state) => state.auth.accessToken);
  const keyPair = useAppSelector((state) => state.auth.keyPair);
  const currentUser = useAppSelector((state) => state.auth.user);

  const [devices, setDevices] = useState<DeviceView[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [backupPassphrase, setBackupPassphrase] = useState("");
  const [restorePassphrase, setRestorePassphrase] = useState("");
  const [restoreBlobInput, setRestoreBlobInput] = useState("");
  const [pairingValue, setPairingValue] = useState("");
  const [pendingDeviceId, setPendingDeviceId] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { clearStoredKeys } = useEncryption();

  const publicKeyPreview = useMemo(
    () => keyPair?.identity.signing.publicKey ?? "No identity signing key loaded",
    [keyPair?.identity.signing.publicKey]
  );
  const pendingDevices = useMemo(
    () => devices.filter((device) => !device.current && device.trusted === false && !device.revoked_at),
    [devices]
  );
  const trustedDevices = useMemo(
    () => devices.filter((device) => device.trusted !== false && !device.revoked_at),
    [devices]
  );
  const deviceLinkUrl = useMemo(() => `${window.location.origin}/auth/device`, []);

  useEffect(() => {
    void loadDevices();
  }, [accessToken]);

  async function loadDevices() {
    if (!accessToken) return;

    setLoadingDevices(true);
    try {
      const response = await fetchDevices(accessToken);
      setDevices(response.devices);
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to load devices.");
    } finally {
      setLoadingDevices(false);
    }
  }

  async function copyIdentityKey() {
    if (!keyPair) return;
    await navigator.clipboard.writeText(keyPair.identity.signing.publicKey);
    setNotice("Identity signing key copied.");
  }

  async function handleExportBackup() {
    if (!keyPair || !accessToken) {
      setError("Sign in and load keys before exporting a backup.");
      return;
    }
    if (!backupPassphrase.trim()) {
      setError("Enter a backup passphrase first.");
      return;
    }

    try {
      const encryptedBackup = await encryptKeyBundleBackup(keyPair, backupPassphrase);
      await exportEncryptedBackup(accessToken, encryptedBackup);
      const serialized = JSON.stringify(encryptedBackup, null, 2);
      setRestoreBlobInput(serialized);
      try {
        await navigator.clipboard.writeText(serialized);
        setNotice("Encrypted backup exported and copied to the clipboard.");
      } catch {
        setNotice("Encrypted backup exported. Clipboard access was not available in this browser.");
      }
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to export encrypted backup.");
    }
  }

  async function handleLoadStoredBackup() {
    if (!accessToken) return;

    try {
      const response = await importEncryptedBackup(accessToken);
      setRestoreBlobInput(JSON.stringify(response.parsed, null, 2));
      setNotice("Stored encrypted backup loaded.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to load the stored backup.");
    }
  }

  async function handleRestoreBackup() {
    if (!restoreBlobInput.trim()) {
      setError("Paste or load an encrypted backup blob first.");
      return;
    }
    if (!restorePassphrase.trim()) {
      setError("Enter the restore passphrase.");
      return;
    }

    try {
      const parsed = JSON.parse(restoreBlobInput);
      const restoredBundle = await decryptKeyBundleBackup(parsed, restorePassphrase);
      await keyStore.saveKeyBundle(restoredBundle);
      dispatch(setKeyPair(restoredBundle));
      setNotice("Encrypted backup restored to this device.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to restore encrypted backup.");
    }
  }

  async function resetIdentity() {
    await clearStoredKeys();
    dispatch(clearAuth());
  }

  async function handleRevokeDevice(deviceId: string) {
    if (!accessToken) return;

    try {
      await revokeDevice(accessToken, deviceId);
      await loadDevices();
      setNotice("Device revoked.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to revoke device.");
    }
  }

  async function generatePairingPayload() {
    try {
      const [signing, exchange] = await Promise.all([generateSigningKeyPair(), generateKeyPair()]);
      const payload: PairingPayload = {
        device_signing_key: signing.publicKey,
        device_exchange_key: exchange.publicKey,
        device_name: "New web browser",
        platform: "web",
      };

      setPairingValue(JSON.stringify(payload, null, 2));
      setNotice("Pairing payload generated.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to generate pairing payload.");
    }
  }

  function parsePairingPayload(): PairingPayload {
    const trimmed = pairingValue.trim();
    if (!trimmed) {
      throw new Error("Paste or generate a pairing payload first.");
    }

    const parsed = JSON.parse(trimmed) as Partial<PairingPayload>;
    if (!parsed.device_signing_key || !parsed.device_exchange_key || !parsed.device_name || !parsed.platform) {
      throw new Error("The pairing payload is missing required fields.");
    }

    return {
      device_signing_key: parsed.device_signing_key,
      device_exchange_key: parsed.device_exchange_key,
      device_name: parsed.device_name,
      platform: parsed.platform,
    };
  }

  async function handleRegisterPendingDevice() {
    if (!accessToken) return;

    try {
      const payload = parsePairingPayload();
      const response = await createPendingDevice(accessToken, payload);
      setPendingDeviceId(response.device.id);
      await loadDevices();
      setNotice(`Pending device registered: ${response.device.id}`);
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to register pending device.");
    }
  }

  async function handleApprovePendingDevice() {
    if (!accessToken || !pendingDeviceId) {
      setError("Register a pending device first.");
      return;
    }

    try {
      await approveDevice(accessToken, pendingDeviceId);
      await loadDevices();
      setPendingDeviceId(null);
      setPairingValue("");
      setNotice("Pending device approved.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to approve device.");
    }
  }

  async function handleApproveSpecificDevice(deviceId: string) {
    if (!accessToken) return;

    try {
      await approveDevice(accessToken, deviceId);
      await loadDevices();
      if (pendingDeviceId === deviceId) {
        setPendingDeviceId(null);
      }
      setNotice("Pending device approved.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to approve device.");
    }
  }

  async function handleRegisterAndApproveDevice() {
    if (!accessToken) return;

    try {
      const payload = parsePairingPayload();
      const response = await createPendingDevice(accessToken, payload);
      await approveDevice(accessToken, response.device.id);
      await loadDevices();
      setPendingDeviceId(null);
      setPairingValue("");
      setNotice(`Device approved: ${response.device.device_name || response.device.id}`);
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to register and approve device.");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="h-4 w-4 text-primary" />
            Identity keys
          </CardTitle>
          <CardDescription>
            {currentUser?.display_name ?? currentUser?.username ?? "Current account"} uses this
            signing key as the long-term identity anchor.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="rounded-2xl border border-border/70 bg-secondary/50 p-3 font-mono text-xs text-muted-foreground break-all">
            {publicKeyPreview}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => void copyIdentityKey()}>
              <Copy className="mr-2 h-3.5 w-3.5" />
              Copy identity key
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => void navigator.clipboard.writeText(deviceLinkUrl)}
            >
              <ExternalLink className="mr-2 h-3.5 w-3.5" />
              Copy new-device link
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Download className="h-4 w-4 text-primary" />
            Encrypted backup
          </CardTitle>
          <CardDescription>
            Your key bundle is encrypted locally with a passphrase. The server only stores the opaque blob.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium">Export passphrase</label>
              <Input
                data-testid="backup-export-passphrase"
                type="password"
                value={backupPassphrase}
                onChange={(event) => setBackupPassphrase(event.target.value)}
                placeholder="Enter a strong backup passphrase"
              />
              <Button data-testid="backup-export-submit" size="sm" onClick={() => void handleExportBackup()}>
                <Download className="mr-2 h-3.5 w-3.5" />
                Export encrypted backup
              </Button>
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium">Restore passphrase</label>
              <Input
                data-testid="backup-restore-passphrase"
                type="password"
                value={restorePassphrase}
                onChange={(event) => setRestorePassphrase(event.target.value)}
                placeholder="Enter the backup passphrase"
              />
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="secondary" onClick={() => void handleLoadStoredBackup()}>
                  Load stored backup
                </Button>
                <Button data-testid="backup-restore-submit" size="sm" onClick={() => void handleRestoreBackup()}>
                  <Upload className="mr-2 h-3.5 w-3.5" />
                  Restore to this device
                </Button>
              </div>
            </div>
          </div>

          <Separator />

          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">Encrypted backup blob</label>
            <Textarea
              data-testid="backup-blob-input"
              rows={8}
              className="font-mono text-xs"
              placeholder="Encrypted backup JSON"
              value={restoreBlobInput}
              onChange={(event) => setRestoreBlobInput(event.target.value)}
            />
          </div>

          <Button size="sm" variant="destructive" onClick={() => void resetIdentity()}>
            <Trash2 className="mr-2 h-3.5 w-3.5" />
            Clear local identity
          </Button>

          {notice && <p className="text-sm text-muted-foreground">{notice}</p>}
          {error && <p className="text-sm text-destructive">{error}</p>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-primary" />
            Device pairing
          </CardTitle>
          <CardDescription>
            Generate a payload on the new device, register it here, then approve it from a trusted session.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <Button
            data-testid="device-pairing-generate"
            size="sm"
            variant="secondary"
            onClick={() => void generatePairingPayload()}
          >
            Generate pairing payload
          </Button>

          <Textarea
            data-testid="device-pairing-input"
            rows={6}
            className="font-mono text-xs"
            placeholder="Paste a pairing payload JSON blob"
            value={pairingValue}
            onChange={(event) => setPairingValue(event.target.value)}
          />

          <div className="flex flex-wrap items-center gap-2">
            <Button
              data-testid="device-pairing-register-approve"
              size="sm"
              onClick={() => void handleRegisterAndApproveDevice()}
            >
              Register and approve
            </Button>
            <Button
              data-testid="device-pairing-register-pending"
              size="sm"
              variant="outline"
              onClick={() => void handleRegisterPendingDevice()}
            >
              Register pending device
            </Button>
            <Button
              data-testid="device-pairing-approve-pending"
              size="sm"
              variant="secondary"
              onClick={() => void handleApprovePendingDevice()}
            >
              Approve pending device
            </Button>
            {pendingDeviceId && <Badge variant="secondary">Pending: {pendingDeviceId}</Badge>}
          </div>

          <div className="rounded-2xl border border-border/60 bg-secondary/40 p-4 text-sm text-muted-foreground">
            <p className="font-medium text-foreground">Recommended flow</p>
            <p className="mt-1">
              Open the new-device link on the target browser, restore the encrypted backup there,
              copy its pairing payload, and paste it here. Use
              <span className="font-medium"> Register and approve</span> for the fastest path.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Smartphone className="h-4 w-4 text-primary" />
            Active devices
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loadingDevices && <p className="text-sm text-muted-foreground">Loading devices...</p>}

          {pendingDevices.length > 0 && (
            <div className="mb-4 flex flex-col gap-2">
              <p className="text-sm font-medium">Pending approval</p>
              {pendingDevices.map((device) => (
                <div
                  key={device.id}
                  className="flex items-center justify-between gap-4 rounded-2xl border border-amber-500/30 bg-amber-500/5 p-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium">{device.device_name || "Unnamed device"}</p>
                    <p className="text-xs text-muted-foreground">
                      {device.platform} • created {new Date(device.created_at).toLocaleString("en-US")}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">Pending</Badge>
                    <Button
                      data-testid={`approve-device-${device.id}`}
                      size="sm"
                      onClick={() => void handleApproveSpecificDevice(device.id)}
                    >
                      Approve
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="flex flex-col gap-2">
            {trustedDevices.map((device) => (
              <div
                key={device.id}
                className="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-secondary/50 p-3"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium">{device.device_name || "Unnamed device"}</p>
                  <p className="text-xs text-muted-foreground">
                    {device.platform}
                    {device.last_seen &&
                      ` • last seen ${new Date(device.last_seen).toLocaleString("en-US")}`}
                    {device.current && " • current"}
                    {device.approved_by_device_id && !device.current && " • approved"}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant={device.current ? "secondary" : "outline"}
                  disabled={device.current}
                  onClick={() => void handleRevokeDevice(device.id)}
                >
                  {device.current ? "Current" : "Revoke"}
                </Button>
              </div>
            ))}
            {!loadingDevices && trustedDevices.length === 0 && pendingDevices.length === 0 && (
              <p className="text-sm text-muted-foreground">No devices found for this account.</p>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
