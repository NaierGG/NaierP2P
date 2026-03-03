export interface User {
  id: string;
  username: string;
  display_name: string;
  public_key?: string;
  identity_signing_key?: string;
  identity_exchange_key?: string;
  avatar_url?: string;
  bio?: string;
  server_id: string;
  created_at: string;
}

export interface Device {
  id: string;
  user_id: string;
  device_key?: string;
  device_signing_key?: string;
  device_exchange_key?: string;
  device_name?: string;
  platform: "web" | "ios" | "android";
  push_token?: string;
  last_seen?: string;
  created_at: string;
  trusted?: boolean;
  approved_by_device_id?: string;
  revoked_at?: string;
}
