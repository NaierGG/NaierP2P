import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowRight, Copy, KeyRound, ShieldCheck } from "lucide-react";

import { setAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { loginWithChallenge, requestChallenge } from "@/features/auth/authApi";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  decryptKeyBundleBackup,
  generateKeyPair,
  generateSigningKeyPair,
  signChallenge,
  type KeyBundle,
} from "@/shared/lib/crypto";
import { keyStore } from "@/shared/lib/keystore";

interface PendingDeviceKeys {
  signing: {
    publicKey: string;
    privateKey: string;
  };
  exchange: {
    publicKey: string;
    privateKey: string;
  };
}

export default function DeviceLinkPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();

  const [username, setUsername] = useState("");
  const [backupBlob, setBackupBlob] = useState("");
  const [passphrase, setPassphrase] = useState("");
  const [deviceKeys, setDeviceKeys] = useState<PendingDeviceKeys | null>(null);
  const [preparedBundle, setPreparedBundle] = useState<KeyBundle | null>(null);
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState<"payload" | null>(null);

  useEffect(() => {
    void (async () => {
      const existingBundle = await keyStore.getKeyBundle();
      if (!existingBundle) {
        return;
      }

      setPreparedBundle(existingBundle);
      setDeviceKeys({
        signing: existingBundle.device.signing,
        exchange: existingBundle.device.exchange,
      });
    })();
  }, []);

  const pairingPayload = useMemo(() => {
    if (!deviceKeys) {
      return "";
    }

    return JSON.stringify(
      {
        device_signing_key: deviceKeys.signing.publicKey,
        device_exchange_key: deviceKeys.exchange.publicKey,
        device_name: "New web browser",
        platform: "web",
      },
      null,
      2
    );
  }, [deviceKeys]);

  async function prepareDevice() {
    setLoading(true);
    setError(null);

    try {
      const parsedBackup = JSON.parse(backupBlob);
      const restoredBundle = await decryptKeyBundleBackup(parsedBackup, passphrase);
      const [nextSigning, nextExchange] = await Promise.all([
        generateSigningKeyPair(),
        generateKeyPair(),
      ]);

      const nextBundle: KeyBundle = {
        identity: restoredBundle.identity,
        device: {
          signing: nextSigning,
          exchange: nextExchange,
        },
      };

      await keyStore.saveKeyBundle(nextBundle);
      dispatch(setKeyPair(nextBundle));
      setDeviceKeys({
        signing: nextSigning,
        exchange: nextExchange,
      });
      setPreparedBundle(nextBundle);
      setNotice(
        "New device keys are ready. Approve this payload from a trusted device, then return here and complete sign-in."
      );
    } catch (nextError) {
      setError(
        nextError instanceof Error
          ? nextError.message
          : "Failed to prepare the new device from the encrypted backup."
      );
    } finally {
      setLoading(false);
    }
  }

  async function completeSignIn() {
    const trimmedUsername = username.trim();
    if (!preparedBundle || !trimmedUsername) {
      setError("Restore the encrypted backup and enter your username first.");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const challengeResponse = await requestChallenge({
        username: trimmedUsername,
        deviceSigningKey: preparedBundle.device.signing.publicKey,
        deviceName: "New web browser",
        platform: "web",
      });

      const signature = await signChallenge(
        challengeResponse.challenge,
        preparedBundle.device.signing.privateKey
      );

      const authResponse = await loginWithChallenge({
        username: trimmedUsername,
        challenge: challengeResponse.challenge,
        signature,
        deviceSigningKey: preparedBundle.device.signing.publicKey,
        deviceName: "New web browser",
        platform: "web",
      });

      dispatch(
        setAuth({
          user: authResponse.user,
          accessToken: authResponse.access_token,
          refreshToken: authResponse.refresh_token,
        })
      );
      dispatch(setKeyPair(preparedBundle));
      setNotice("This device is approved and signed in.");
      navigate("/app");
    } catch (nextError) {
      setError(
        nextError instanceof Error
          ? nextError.message
          : "Sign-in is not ready yet. Make sure the trusted device has approved this device."
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card className="w-full max-w-2xl">
      <CardHeader className="items-center text-center">
        <p className="text-[11px] font-semibold uppercase tracking-[0.28em] text-primary/75">
          Trusted device handoff
        </p>
        <div className="mb-2 flex h-12 w-12 items-center justify-center rounded-2xl border border-primary/15 bg-primary/10">
          <ShieldCheck className="h-6 w-6 text-primary" />
        </div>
        <CardTitle className="text-xl">Add a new device</CardTitle>
        <CardDescription>
          Restore your encrypted backup, generate fresh device keys, approve them from a trusted
          device, then complete sign-in on this browser.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">Username</label>
            <Input
              data-testid="device-link-username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="username"
            />
          </div>
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">Backup passphrase</label>
            <Input
              data-testid="device-link-passphrase"
              type="password"
              value={passphrase}
              onChange={(event) => setPassphrase(event.target.value)}
              placeholder="Enter the encrypted backup passphrase"
            />
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium">Encrypted backup blob</label>
          <Textarea
            data-testid="device-link-backup-blob"
            rows={8}
            className="font-mono text-xs"
            value={backupBlob}
            onChange={(event) => setBackupBlob(event.target.value)}
            placeholder="Paste the encrypted backup JSON exported from a trusted device"
          />
        </div>

        <div className="flex flex-wrap gap-2">
          <Button
            data-testid="device-link-prepare"
            disabled={loading}
            onClick={() => void prepareDevice()}
          >
            <KeyRound className="mr-2 h-4 w-4" />
            Prepare this device
          </Button>
          <Button
            data-testid="device-link-complete"
            disabled={!preparedBundle || loading}
            variant="secondary"
            onClick={() => void completeSignIn()}
          >
            Complete sign-in
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>

        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium">Pairing payload for the trusted device</label>
          <Textarea
            data-testid="device-link-pairing-payload"
            readOnly
            rows={6}
            className="font-mono text-xs"
            value={pairingPayload}
            placeholder="Prepare this device first to generate the pairing payload"
          />
          <div>
            <Button
              size="sm"
              variant="secondary"
              disabled={!pairingPayload}
              onClick={() => {
                void navigator.clipboard.writeText(pairingPayload);
                setCopied("payload");
              }}
            >
              <Copy className="mr-2 h-3.5 w-3.5" />
              {copied === "payload" ? "Copied pairing payload" : "Copy pairing payload"}
            </Button>
          </div>
        </div>

        <div className="rounded-2xl border border-border/60 bg-secondary/40 p-4 text-sm text-muted-foreground">
          <p className="font-medium text-foreground">Next step</p>
          <p className="mt-1">
            Open <span className="font-mono">Settings &gt; Security</span> on a trusted signed-in
            device, paste this pairing payload, approve it, then come back here and press
            <span className="font-medium"> Complete sign-in</span>.
          </p>
        </div>

        {notice && <p className="text-sm text-muted-foreground">{notice}</p>}
        {error && <p className="text-sm text-destructive">{error}</p>}

        <p className="text-center text-sm text-muted-foreground">
          Already have an approved device here?{" "}
          <Link to="/auth/login" className="font-medium text-primary hover:underline">
            Go to login
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
