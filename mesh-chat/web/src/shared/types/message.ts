export interface Reaction {
  user_id: string;
  emoji: string;
}

export interface Message {
  id: string;
  channel_id: string;
  sender_id: string;
  type: "text" | "image" | "file" | "system";
  content: string;
  iv?: string;
  reply_to_id?: string;
  is_edited: boolean;
  is_deleted: boolean;
  signature?: string;
  created_at: string;
  updated_at: string;
  reactions?: Reaction[];
  status?: "sending" | "sent" | "failed";
}

export interface PendingMessage extends Message {
  client_id: string;
  error?: string;
}
