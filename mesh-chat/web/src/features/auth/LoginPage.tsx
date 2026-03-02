import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { setAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch } from "@/app/store/hooks";
import { loginWithChallenge, requestChallenge } from "@/features/auth/authApi";
import { useEncryption } from "@/shared/hooks/useEncryption";

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
      setError("Username is required.");
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
      const challengeResponse = await requestChallenge(trimmedUsername);
      const signature = await signLoginChallenge(challengeResponse.challenge);
      const authResponse = await loginWithChallenge({
        username: trimmedUsername,
        challenge: challengeResponse.challenge,
        signature,
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
      setError(nextError instanceof Error ? nextError.message : "Login failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="auth-card">
      <div>
        <p className="eyebrow">Mesh Chat</p>
        <h1>Challenge-based login</h1>
        <p className="muted">
          The full key challenge flow lands in the next step. This scaffold keeps
          routing and state ready for it.
        </p>
      </div>

      <div className="auth-actions">
        <input
          className="message-input"
          onChange={(event) => setUsername(event.target.value)}
          placeholder="Username"
          value={username}
        />
        {error ? <p className="error-text">{error}</p> : null}
        <button
          className="primary-button"
          disabled={loading}
          onClick={() => void handleLogin()}
          type="button"
        >
          {loading ? "Signing in..." : "Continue"}
        </button>
        <p className="muted">
          New here? <Link to="/auth/register">Create an account</Link>
        </p>
      </div>
    </section>
  );
}
