import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Shield } from "lucide-react";

import { setAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { loginWithChallenge, requestChallenge } from "@/features/auth/authApi";
import { useEncryption } from "@/shared/hooks/useEncryption";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function LoginPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { loadKeyPair, signLoginChallenge } = useEncryption();
  const [username, setUsername] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleLogin() {
    const trimmedUsername = username.trim();
    if (!trimmedUsername) {
      setError("사용자 이름을 입력하세요.");
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

      dispatch(setKeyPair(existingKeyPair));
      const challengeResponse = await requestChallenge({
        username: trimmedUsername,
        deviceSigningKey: existingKeyPair.device.signing.publicKey,
        deviceName: "Web browser",
        platform: "web",
      });
      const signature = await signLoginChallenge(challengeResponse.challenge);
      const authResponse = await loginWithChallenge({
        username: trimmedUsername,
        challenge: challengeResponse.challenge,
        signature,
        deviceSigningKey: existingKeyPair.device.signing.publicKey,
        deviceName: "Web browser",
        platform: "web",
      });

      dispatch(
        setAuth({
          user: authResponse.user,
          accessToken: authResponse.access_token,
          refreshToken: authResponse.refresh_token,
        })
      );

      navigate("/app");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "로그인에 실패했습니다.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="items-center text-center">
        <div className="mb-2 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10">
          <Shield className="h-6 w-6 text-primary" />
        </div>
        <CardTitle className="text-xl">Naier</CardTitle>
        <CardDescription>
          디바이스 서명 키로 서버 챌린지에 서명하여 로그인합니다.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <Input
          onChange={(e) => setUsername(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") void handleLogin();
          }}
          placeholder="사용자 이름"
          value={username}
          autoFocus
        />
        {error && <p className="text-sm text-destructive">{error}</p>}
        <Button disabled={loading} onClick={() => void handleLogin()} className="w-full">
          {loading ? "서명 중..." : "로그인"}
        </Button>
        <p className="text-center text-sm text-muted-foreground">
          처음이신가요?{" "}
          <Link to="/auth/register" className="font-medium text-primary hover:underline">
            계정 만들기
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
