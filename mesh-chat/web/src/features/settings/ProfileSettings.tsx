import { useEffect, useState } from "react";

import { setAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { fetchProfile, updateProfile } from "@/features/settings/settingsApi";

export default function ProfileSettings() {
  const dispatch = useAppDispatch();
  const user = useAppSelector((state) => state.auth.user);
  const accessToken = useAppSelector((state) => state.auth.accessToken);
  const refreshToken = useAppSelector((state) => state.auth.refreshToken);
  const [displayName, setDisplayName] = useState(user?.display_name ?? "");
  const [bio, setBio] = useState(user?.bio ?? "");
  const [loading, setLoading] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDisplayName(user?.display_name ?? "");
    setBio(user?.bio ?? "");
  }, [user?.bio, user?.display_name]);

  useEffect(() => {
    if (!accessToken) {
      return;
    }

    let cancelled = false;
    void (async () => {
      try {
        const response = await fetchProfile(accessToken);
        if (cancelled) {
          return;
        }

        dispatch(
          setAuth({
            user: response.user,
            accessToken,
            refreshToken: refreshToken ?? undefined,
          })
        );
      } catch {
        // Keep current client state if the profile fetch fails.
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [accessToken, dispatch, refreshToken]);

  if (!user || !accessToken) {
    return null;
  }

  const currentUser = user;
  const currentAccessToken = accessToken;

  async function handleSave() {
    setLoading(true);
    setSaved(false);
    setError(null);

    try {
      const response = await updateProfile(accessToken, {
        display_name: displayName.trim() || currentUser.display_name,
        bio: bio.trim(),
        avatar_url: currentUser.avatar_url ?? "",
      });

      dispatch(
        setAuth({
          user: response.user,
          accessToken: currentAccessToken,
          refreshToken: refreshToken ?? undefined,
        })
      );
      setSaved(true);
      window.setTimeout(() => setSaved(false), 1600);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Profile update failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="settings-panel">
      <p className="eyebrow">Profile</p>
      <h2>Identity and presence</h2>
      <p className="muted">
        Profile settings now use the backend auth profile API when available and fall back to mock mode only if the API is unreachable.
      </p>

      <div className="settings-field-grid">
        <label className="settings-field">
          <span>Username</span>
          <input className="message-input" disabled value={currentUser.username} />
        </label>
        <label className="settings-field">
          <span>Display name</span>
          <input
            className="message-input"
            onChange={(event) => setDisplayName(event.target.value)}
            value={displayName}
          />
        </label>
      </div>

      <label className="settings-field">
        <span>Bio</span>
        <textarea
          className="message-input"
          onChange={(event) => setBio(event.target.value)}
          rows={4}
          value={bio}
        />
      </label>

      <div className="settings-actions">
        <button className="primary-button" disabled={loading} onClick={() => void handleSave()} type="button">
          {loading ? "Saving..." : "Save profile"}
        </button>
        {saved ? <span className="muted">Profile saved.</span> : null}
        {error ? <span className="error-text">{error}</span> : null}
      </div>
    </section>
  );
}
