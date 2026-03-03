export interface Reaction {
  user_id: string;
  emoji: string;
}

export interface ReactionEvent {
  message_id: string;
  channel_id: string;
  user_id: string;
  emoji: string;
  action: "add" | "remove";
  event_id: string;
  sequence: number;
  created_at: string;
}

export interface ReadState {
  channel_id: string;
  user_id: string;
  last_read_sequence: number;
  event_id: string;
  sequence: number;
  created_at: string;
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
  client_event_id?: string;
  server_event_id?: string;
  sequence?: number;
  created_at: string;
  updated_at: string;
  reactions?: Reaction[];
  status?: "sending" | "sent" | "failed";
}

export interface PendingMessage extends Message {
  client_id: string;
  error?: string;
}

export interface SyncEvent {
  type: "MESSAGE_NEW" | "MESSAGE_UPDATED" | "MESSAGE_DELETED" | "REACTION" | "READ_STATE";
  message?: Message;
  reaction?: ReactionEvent;
  read_state?: ReadState;
  event_id: string;
  sequence: number;
  channel_id: string;
}
