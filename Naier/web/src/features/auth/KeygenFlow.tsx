import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Key, Download, Copy, ArrowRight } from "lucide-react";

import { setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { useEncryption } from "@/shared/hooks/useEncryption";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function KeygenFlow() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { createAndStoreKeyPair, loadKeyPair } = useEncryption();
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const [keyPair, setLocalKeyPair] = useState<Awaited<ReturnType<typeof createAndStoreKeyPair>> | null>(null);
  const [downloadHref, setDownloadHref] = useState("#");

  useEffect(() => {
    void (async () => {
      const existing = await loadKeyPair();
      if (existing) {
        setLocalKeyPair(existing);
        setStep(3);
        dispatch(setKeyPair(existing));
      }
    })();
  }, [dispatch, loadKeyPair]);

  useEffect(() => {
    if (!keyPair) {
      setDownloadHref("#");
      return;
    }

    const payload = JSON.stringify(keyPair, null, 2);
    const href = URL.createObjectURL(
      new Blob([payload], { type: "application/json" })
    );
    setDownloadHref(href);

    return () => { URL.revokeObjectURL(href); };
  }, [keyPair]);

  async function generate() {
    setLoading(true);
    try {
      const nextKeyPair = await createAndStoreKeyPair();
      setLocalKeyPair(nextKeyPair);
      dispatch(setKeyPair(nextKeyPair));
      setStep(3);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card className="w-full max-w-lg">
      <CardHeader className="items-center text-center">
        <div className="mb-2 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10">
          <Key className="h-6 w-6 text-primary" />
        </div>
        <CardTitle className="text-xl">키 생성</CardTitle>
        <CardDescription>
          {step === 1 && "신원 키와 디바이스 키가 계정의 기초가 됩니다."}
          {step === 2 && "장기 신원 키와 첫 번째 신뢰 디바이스 키를 생성합니다."}
          {step === 3 && "키 번들을 반드시 백업하세요. 서버는 복구할 수 없습니다."}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-4">
        {step === 1 && (
          <Button onClick={() => setStep(2)} className="w-full">
            시작하기
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        )}

        {step === 2 && (
          <Button disabled={loading} onClick={() => void generate()} className="w-full">
            {loading ? "생성 중..." : "키 생성"}
          </Button>
        )}

        {step === 3 && keyPair && (
          <>
            <Textarea
              readOnly
              rows={10}
              value={JSON.stringify(keyPair, null, 2)}
              className="font-mono text-xs"
            />
            <div className="flex gap-2">
              <Button
                variant="secondary"
                onClick={() => void navigator.clipboard.writeText(JSON.stringify(keyPair, null, 2))}
                className="flex-1"
              >
                <Copy className="mr-2 h-4 w-4" />
                복사
              </Button>
              <Button variant="secondary" asChild className="flex-1">
                <a download="naier-keypair.json" href={downloadHref}>
                  <Download className="mr-2 h-4 w-4" />
                  다운로드
                </a>
              </Button>
            </div>

            <label className="flex items-start gap-3 rounded-xl bg-accent p-3 cursor-pointer">
              <input
                type="checkbox"
                checked={confirmed}
                onChange={(e) => setConfirmed(e.target.checked)}
                className="mt-0.5 h-4 w-4 rounded border-input accent-primary"
              />
              <span className="text-sm text-foreground">
                이 키는 서버에서 복구할 수 없음을 이해합니다.
              </span>
            </label>

            <Button
              disabled={!confirmed}
              onClick={() => navigate("/auth/register")}
              className="w-full"
            >
              가입 계속하기
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  );
}
