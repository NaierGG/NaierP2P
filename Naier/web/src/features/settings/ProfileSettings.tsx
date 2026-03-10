import { useEffect, useState } from "react";

import { setAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
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
    if (!accessToken) return;

    let cancelled = false;
    void (async () => {
      try {
        const response = await fetchProfile(accessToken);
        if (cancelled) return;

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

  if (!user || !accessToken) return null;

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
    <Card>
      <CardHeader>
        <CardTitle>Profile</CardTitle>
        <CardDescription>
          Control the name and bio other members see when they enter a trusted conversation.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid gap-4 md:grid-cols-2">
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Username</span>
            <Input disabled value={currentUser.username} />
          </label>
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Display name</span>
            <Input onChange={(event) => setDisplayName(event.target.value)} value={displayName} />
          </label>
        </div>

        <label className="flex flex-col gap-1.5">
          <span className="text-sm font-medium">Bio</span>
          <Textarea
            onChange={(event) => setBio(event.target.value)}
            rows={3}
            value={bio}
            placeholder="Write a short introduction or trust signal for your contacts."
          />
        </label>

        <div className="flex items-center gap-3">
          <Button disabled={loading} onClick={() => void handleSave()}>
            {loading ? "Saving..." : "Save profile"}
          </Button>
          {saved && <span className="text-sm text-emerald-500">Saved.</span>}
          {error && <span className="text-sm text-destructive">{error}</span>}
        </div>
      </CardContent>
    </Card>
  );
}
