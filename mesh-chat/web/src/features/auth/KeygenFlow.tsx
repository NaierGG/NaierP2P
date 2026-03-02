import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { useEncryption } from "@/shared/hooks/useEncryption";

export default function KeygenFlow() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { createAndStoreKeyPair, loadKeyPair } = useEncryption();
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const [keyPair, setLocalKeyPair] = useState<{
    publicKey: string;
    privateKey: string;
  } | null>(null);
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
    <section className="panel">
      <p className="eyebrow">Keygen</p>
      {step === 1 ? (
        <>
          <h2>Your keypair becomes your identity</h2>
          <p className="muted">
            Mesh Chat does not have password recovery. Your private key is the account.
          </p>
          <button className="primary-button" onClick={() => setStep(2)} type="button">
            Start
          </button>
        </>
      ) : null}

      {step === 2 ? (
        <>
          <h2>Generate keypair</h2>
          <p className="muted">
            This creates the local identity material used for challenge signing.
          </p>
          <button className="primary-button" disabled={loading} onClick={() => void generate()} type="button">
            {loading ? "Generating..." : "Generate now"}
          </button>
        </>
      ) : null}

      {step === 3 && keyPair ? (
        <>
          <h2>Back up your private key</h2>
          <p className="muted">
            If you lose this private key, the account cannot be recovered.
          </p>
          <textarea className="message-input" readOnly rows={6} value={keyPair.privateKey} />
          <div className="auth-actions keygen-actions">
            <button
              className="primary-button"
              onClick={() => void navigator.clipboard.writeText(keyPair.privateKey)}
              type="button"
            >
              Copy text
            </button>
            <a className="primary-button link-button" download="meshchat-keypair.json" href={downloadHref}>
              Download file
            </a>
          </div>
          <label className="checkbox-row">
            <input
              checked={confirmed}
              onChange={(event) => setConfirmed(event.target.checked)}
              type="checkbox"
            />
            <span>I understand this key cannot be recovered by the server.</span>
          </label>
          <button
            className="primary-button"
            disabled={!confirmed}
            onClick={() => navigate("/auth/register")}
            type="button"
          >
            Continue to registration
          </button>
        </>
      ) : null}
    </section>
  );
}
