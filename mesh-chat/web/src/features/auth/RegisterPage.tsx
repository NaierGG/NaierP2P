import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { setAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { registerWithKeyPair, requestChallenge } from "@/features/auth/authApi";
import { deriveSigningPublicKey } from "@/shared/lib/crypto";
import { useEncryption } from "@/shared/hooks/useEncryption";

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
      setError("Username and display name are required.");
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

      const challengeResponse = await requestChallenge(trimmedUsername);
      const signature = await signLoginChallenge(challengeResponse.challenge);
      const authResponse = await registerWithKeyPair({
        username: trimmedUsername,
        displayName: trimmedDisplayName,
        // Current backend auth verifies Ed25519 signatures, so registration publishes the signing key.
        publicKey: deriveSigningPublicKey(existingKeyPair.privateKey),
        signature,
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
      setError(nextError instanceof Error ? nextError.message : "Registration failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="auth-card">
      <div>
        <p className="eyebrow">Identity</p>
        <h1>Create your Mesh Chat account</h1>
        <p className="muted">
          Registration UI and key generation flow will plug into this route next.
        </p>
      </div>

      <div className="auth-actions">
        <input
          className="message-input"
          onChange={(event) => setUsername(event.target.value)}
          placeholder="Username"
          value={username}
        />
        <input
          className="message-input"
          onChange={(event) => setDisplayName(event.target.value)}
          placeholder="Display name"
          value={displayName}
        />
        {!hasKeyPair ? (
          <p className="muted">
            No keypair found. <Link to="/auth/keygen">Generate one first</Link>.
          </p>
        ) : null}
        {error ? <p className="error-text">{error}</p> : null}
        <button
          className="primary-button"
          disabled={!hasKeyPair || loading}
          onClick={() => void handleRegister()}
          type="button"
        >
          {loading ? "Registering..." : "Create account"}
        </button>
        <p className="muted">
          Already registered? <Link to="/auth/login">Back to login</Link>
        </p>
      </div>
    </section>
  );
}
