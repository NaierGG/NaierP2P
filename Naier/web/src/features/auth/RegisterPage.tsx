import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { UserPlus } from "lucide-react";

import { setAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { registerWithKeyPair, requestChallenge } from "@/features/auth/authApi";
import { signChallenge } from "@/shared/lib/crypto";
import { useEncryption } from "@/shared/hooks/useEncryption";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export default function RegisterPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { loadKeyPair, signLoginChallenge } = useEncryption();
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [hasKeyPair, setHasKeyPair] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      const existing = await loadKeyPair();
      setHasKeyPair(Boolean(existing));
      if (existing) {
        dispatch(setKeyPair(existing));
      }
    })();
  }, [dispatch, loadKeyPair]);

  async function handleRegister() {
    const trimmedUsername = username.trim();
    const trimmedDisplayName = displayName.trim();
    if (!trimmedUsername || !trimmedDisplayName) {
      setError("Enter both a username and a display name.");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const existingKeyPair = await loadKeyPair();
      if (!existingKeyPair) {
        navigate("/auth/keygen");
        return;
      }

      const challengeResponse = await requestChallenge({
        username: trimmedUsername,
        deviceSigningKey: existingKeyPair.device.signing.publicKey,
        deviceName: "Web browser",
        platform: "web",
      });
      const deviceSignature = await signLoginChallenge(challengeResponse.challenge);
      const identitySignatureOverDevice = await signChallenge(
        `${existingKeyPair.device.signing.publicKey}:${existingKeyPair.device.exchange.publicKey}`,
        existingKeyPair.identity.signing.privateKey
      );
      const authResponse = await registerWithKeyPair({
        username: trimmedUsername,
        displayName: trimmedDisplayName,
        identitySigningKey: existingKeyPair.identity.signing.publicKey,
        identityExchangeKey: existingKeyPair.identity.exchange.publicKey,
        deviceSigningKey: existingKeyPair.device.signing.publicKey,
        deviceExchangeKey: existingKeyPair.device.exchange.publicKey,
        deviceSignature,
        identitySignatureOverDevice,
        deviceName: "Web browser",
        platform: "web",
      });

      dispatch(setKeyPair(existingKeyPair));
      dispatch(
        setAuth({
          user: authResponse.user,
          accessToken: authResponse.access_token,
          refreshToken: authResponse.refresh_token,
        })
      );

      navigate("/app");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to create the account.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="items-center text-center">
        <div className="mb-2 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10">
          <UserPlus className="h-6 w-6 text-primary" />
        </div>
        <CardTitle className="text-xl">Create your account</CardTitle>
        <CardDescription>
          Register a username and bind it to your long-term identity keys.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <Input
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Username"
          value={username}
          autoFocus
        />
        <Input
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder="Display name"
          value={displayName}
        />
        {!hasKeyPair && (
          <div className="flex items-center gap-2 rounded-xl bg-accent p-3">
            <Badge variant="warning">Required</Badge>
            <p className="text-sm text-muted-foreground">
              <Link to="/auth/keygen" className="font-medium text-primary hover:underline">
                Generate your identity keys
              </Link>{" "}
              before creating the account.
            </p>
          </div>
        )}
        {error && <p className="text-sm text-destructive">{error}</p>}
        <Button
          disabled={!hasKeyPair || loading}
          onClick={() => void handleRegister()}
          className="w-full"
        >
          {loading ? "Creating account..." : "Create account"}
        </Button>
        <p className="text-center text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link to="/auth/login" className="font-medium text-primary hover:underline">
            Log in
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
