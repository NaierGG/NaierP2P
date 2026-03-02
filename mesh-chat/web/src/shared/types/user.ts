export interface User {
  id: string;
  username: string;
  display_name: string;
  public_key: string;
  avatar_url?: string;
  bio?: string;
  server_id: string;
  created_at: string;
}

export interface Device {
  id: string;
  user_id: string;
  device_key: string;
  device_name?: string;
  platform: "web" | "ios" | "android";
  push_token?: string;
  last_seen?: string;
  created_at: string;
}
