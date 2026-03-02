import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { clearAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import NotificationSettings from "@/features/settings/NotificationSettings";
import ProfileSettings from "@/features/settings/ProfileSettings";
import SecuritySettings from "@/features/settings/SecuritySettings";
import { useSettings } from "@/features/settings/useSettings";

type SettingsSection = "profile" | "notifications" | "security";

export default function SettingsPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const user = useAppSelector((state) => state.auth.user);
  const [activeSection, setActiveSection] = useState<SettingsSection>("profile");
  const { settings, setSettings } = useSettings();

  function logout() {
    dispatch(clearAuth());
    navigate("/auth/login");
  }

  return (
    <section className="settings-shell">
      <div className="settings-page">
        <div className="settings-topbar">
          <div>
            <p className="eyebrow">Settings</p>
            <h1>{user ? `${user.display_name || user.username} preferences` : "App settings"}</h1>
            <p className="muted">
              앱 동작, 알림, 키 관리 설정을 한 곳에서 조정합니다.
            </p>
          </div>

          <div className="settings-actions">
            <Link className="secondary-button link-button" to="/app">
              Back to chat
            </Link>
            <button className="secondary-button is-danger" onClick={logout} type="button">
              Sign out
            </button>
          </div>
        </div>

        <div className="settings-grid">
          <aside className="settings-nav">
            <button
              className={`secondary-button ${activeSection === "profile" ? "is-active" : ""}`}
              onClick={() => setActiveSection("profile")}
              type="button"
            >
              Profile
            </button>
            <button
              className={`secondary-button ${activeSection === "notifications" ? "is-active" : ""}`}
              onClick={() => setActiveSection("notifications")}
              type="button"
            >
              Notifications
            </button>
            <button
              className={`secondary-button ${activeSection === "security" ? "is-active" : ""}`}
              onClick={() => setActiveSection("security")}
              type="button"
            >
              Security
            </button>
          </aside>

          <div className="settings-panel-stack">
            {activeSection === "profile" ? <ProfileSettings /> : null}
            {activeSection === "notifications" ? (
              <NotificationSettings settings={settings} setSettings={setSettings} />
            ) : null}
            {activeSection === "security" ? <SecuritySettings /> : null}
          </div>
        </div>
      </div>
    </section>
  );
}
