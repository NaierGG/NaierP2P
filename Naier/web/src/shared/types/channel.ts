import type { Message } from "@/shared/types/message";

export type ChannelType = "dm" | "group" | "public";

export interface Channel {
  id: string;
  type: ChannelType;
  name: string;
  description: string;
  invite_code?: string;
  owner_id?: string;
  is_encrypted: boolean;
  max_members: number;
  member_count: number;
  created_at: string;
  last_message?: Message;
}

export interface ChannelMember {
  user_id: string;
  username: string;
  display_name: string;
  role: "owner" | "admin" | "member";
  joined_at: string;
  is_muted: boolean;
}
