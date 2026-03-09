import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowRight, Copy, Download, Key } from "lucide-react";

import { setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { useEncryption } from "@/shared/hooks/useEncryption";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";

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
    const href = URL.createObjectURL(new Blob([payload], { type: "application/json" }));
    setDownloadHref(href);

    return () => {
      URL.revokeObjectURL(href);
    };
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
        <CardTitle className="text-xl">Generate your identity</CardTitle>
        <CardDescription>
          {step === 1 && "Your identity keys anchor the account and every trusted device you add later."}
          {step === 2 && "Generating identity and first-device keys locally in this browser."}
          {step === 3 && "Back up this key bundle now. The server cannot recover it for you."}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-4">
        {step === 1 && (
          <Button data-testid="keygen-start" onClick={() => setStep(2)} className="w-full">
            Start
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        )}

        {step === 2 && (
          <Button data-testid="keygen-generate" disabled={loading} onClick={() => void generate()} className="w-full">
            {loading ? "Generating..." : "Generate keys"}
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
                Copy
              </Button>
              <Button variant="secondary" asChild className="flex-1">
                <a download="naier-keypair.json" href={downloadHref}>
                  <Download className="mr-2 h-4 w-4" />
                  Download
                </a>
              </Button>
            </div>

            <label className="flex cursor-pointer items-start gap-3 rounded-xl bg-accent p-3">
              <input
                data-testid="keygen-confirm"
                type="checkbox"
                checked={confirmed}
                onChange={(e) => setConfirmed(e.target.checked)}
                className="mt-0.5 h-4 w-4 rounded border-input accent-primary"
              />
              <span className="text-sm text-foreground">
                I understand that losing this bundle without an encrypted backup can lock me out permanently.
              </span>
            </label>

            <Button data-testid="keygen-continue" disabled={!confirmed} onClick={() => navigate("/auth/register")} className="w-full">
              Continue to registration
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  );
}
