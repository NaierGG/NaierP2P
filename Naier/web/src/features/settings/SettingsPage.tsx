import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft, LogOut } from "lucide-react";

import { clearAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import NotificationSettings from "@/features/settings/NotificationSettings";
import ProfileSettings from "@/features/settings/ProfileSettings";
import SecuritySettings from "@/features/settings/SecuritySettings";
import { useSettings } from "@/features/settings/useSettings";

export default function SettingsPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const user = useAppSelector((state) => state.auth.user);
  const { settings, setSettings } = useSettings();

  function logout() {
    dispatch(clearAuth());
    navigate("/auth/login");
  }

  return (
    <div className="app-noise flex h-screen flex-col bg-transparent p-3 md:p-4 lg:p-5">
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.9rem] border border-border/70 bg-card/85 shadow-[0_28px_80px_rgba(3,10,22,0.38)] backdrop-blur-xl">
        <header className="flex items-center justify-between border-b border-border/70 bg-card/55 px-6 py-5 backdrop-blur-sm">
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="icon" asChild>
              <Link to="/app">
                <ArrowLeft className="h-4 w-4" />
              </Link>
            </Button>
            <div>
              <h1 className="text-lg font-semibold">Settings</h1>
              <p className="text-xs text-muted-foreground">
                {user ? `${user.display_name || user.username}` : "Account preferences"}
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={logout}
            className="gap-2 text-destructive hover:text-destructive"
          >
            <LogOut className="h-3.5 w-3.5" />
            Log out
          </Button>
        </header>

        <div className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-4xl px-6 py-6">
            <div className="mb-6 rounded-[1.5rem] border border-primary/15 bg-primary/10 px-5 py-4">
              <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-primary/75">
                Calm defaults
              </p>
              <p className="mt-2 text-sm text-muted-foreground">
                The interface uses restrained contrast and cool hues to reduce alert fatigue,
                improve scanning, and keep security actions visually distinct.
              </p>
            </div>

            <Tabs defaultValue="profile">
              <TabsList className="w-full justify-start">
                <TabsTrigger value="profile">Profile</TabsTrigger>
                <TabsTrigger value="notifications">Notifications</TabsTrigger>
                <TabsTrigger value="security" data-testid="settings-security-tab">
                  Security
                </TabsTrigger>
              </TabsList>

              <TabsContent value="profile">
                <ProfileSettings />
              </TabsContent>

              <TabsContent value="notifications">
                <NotificationSettings settings={settings} setSettings={setSettings} />
              </TabsContent>

              <TabsContent value="security">
                <SecuritySettings />
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </div>
    </div>
  );
}
