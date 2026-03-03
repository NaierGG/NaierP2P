import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { UserPlus } from "lucide-react";

import { setAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { registerWithKeyPair, requestChallenge } from "@/features/auth/authApi";
import { signChallenge } from "@/shared/lib/crypto";
import { useEncryption } from "@/shared/hooks/useEncryption";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

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
      setError("사용자 이름과 표시 이름 모두 입력하세요.");
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
      setError(nextError instanceof Error ? nextError.message : "가입에 실패했습니다.");
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
        <CardTitle className="text-xl">계정 만들기</CardTitle>
        <CardDescription>
          Naier 계정을 만들어 안전한 대화를 시작하세요.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <Input
          onChange={(e) => setUsername(e.target.value)}
          placeholder="사용자 이름"
          value={username}
          autoFocus
        />
        <Input
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder="표시 이름"
          value={displayName}
        />
        {!hasKeyPair && (
          <div className="flex items-center gap-2 rounded-xl bg-accent p-3">
            <Badge variant="warning">키 필요</Badge>
            <p className="text-sm text-muted-foreground">
              <Link to="/auth/keygen" className="font-medium text-primary hover:underline">
                키 생성
              </Link>
              이 필요합니다.
            </p>
          </div>
        )}
        {error && <p className="text-sm text-destructive">{error}</p>}
        <Button
          disabled={!hasKeyPair || loading}
          onClick={() => void handleRegister()}
          className="w-full"
        >
          {loading ? "가입 중..." : "계정 만들기"}
        </Button>
        <p className="text-center text-sm text-muted-foreground">
          이미 계정이 있나요?{" "}
          <Link to="/auth/login" className="font-medium text-primary hover:underline">
            로그인
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
