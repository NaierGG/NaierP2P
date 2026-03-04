import type { AxiosError } from "axios";

import type {
  Channel,
  ChannelMember,
  Device,
  Message,
  User,
  WSEvent,
  TypingPayload,
} from "@/shared/types";
import type { EncryptedBackupBlob } from "@/shared/lib/crypto";

const MOCK_DB_KEY = "naier-mock-db";
const MOCK_CHALLENGE_PREFIX = "naier-mock-challenge:";

interface MockAuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

interface MockDatabase {
  users: User[];
  devices: Device[];
  channels: Channel[];
  messages: Record<string, Message[]>;
  channelMembers: Record<string, ChannelMember[]>;
  backups: Record<string, { backup_blob: string; backup_version: number; updated_at: string }>;
}

interface MockMessageListResponse {
  messages: Message[];
  next_cursor?: string;
  has_more: boolean;
}

function createSeedDatabase(): MockDatabase {
  const now = Date.now();
  const guideUser = createUser("guide", "Mesh Guide");
  const operatorUser = createUser("ops", "Relay Ops");
  const generalChannelId = crypto.randomUUID();
  const buildChannelId = crypto.randomUUID();

  const generalMessages = [
    createMessage({
      id: crypto.randomUUID(),
      channelId: generalChannelId,
      senderId: guideUser.id,
      content: "Welcome to Naier mock mode. The real backend is currently unavailable.",
      createdAt: new Date(now - 1000 * 60 * 60 * 4).toISOString(),
    }),
    createMessage({
      id: crypto.randomUUID(),
      channelId: generalChannelId,
      senderId: operatorUser.id,
      content: "You can still register, browse channels, and send local demo messages.",
      createdAt: new Date(now - 1000 * 60 * 55).toISOString(),
    }),
  ];

  const buildMessages = [
    createMessage({
      id: crypto.randomUUID(),
      channelId: buildChannelId,
      senderId: operatorUser.id,
      content: "Mock transport is active. WebSocket events will be simulated locally.",
      createdAt: new Date(now - 1000 * 60 * 30).toISOString(),
    }),
  ];

  return {
    users: [guideUser, operatorUser],
    devices: [
      {
        id: crypto.randomUUID(),
        user_id: guideUser.id,
        device_key: "mock-device-guide",
        device_name: "Guide browser",
        platform: "web",
        created_at: new Date(now - 1000 * 60 * 60 * 6).toISOString(),
        last_seen: new Date(now - 1000 * 60 * 5).toISOString(),
      },
      {
        id: crypto.randomUUID(),
        user_id: operatorUser.id,
        device_key: "mock-device-ops",
        device_name: "Ops browser",
        platform: "web",
        created_at: new Date(now - 1000 * 60 * 60 * 5).toISOString(),
        last_seen: new Date(now - 1000 * 60 * 7).toISOString(),
      },
    ],
    channels: [
      createChannel({
        id: generalChannelId,
        name: "General",
        description: "Product updates and onboarding",
        type: "group",
        memberCount: 2,
        lastMessage: generalMessages[generalMessages.length - 1],
        createdAt: new Date(now - 1000 * 60 * 60 * 24).toISOString(),
      }),
      createChannel({
        id: buildChannelId,
        name: "Build Room",
        description: "Delivery status, federation, runtime checks",
        type: "group",
        memberCount: 2,
        lastMessage: buildMessages[buildMessages.length - 1],
        createdAt: new Date(now - 1000 * 60 * 60 * 12).toISOString(),
      }),
    ],
    messages: {
      [generalChannelId]: generalMessages,
      [buildChannelId]: buildMessages,
    },
    channelMembers: {
      [generalChannelId]: [
        createMember(guideUser, "owner"),
        createMember(operatorUser, "member"),
      ],
      [buildChannelId]: [
        createMember(operatorUser, "owner"),
        createMember(guideUser, "member"),
      ],
    },
    backups: {},
  };
}

function createUser(username: string, displayName: string): User {
  return {
    id: crypto.randomUUID(),
    username,
    display_name: displayName,
    public_key: "mock-public-key",
    avatar_url: "",
    bio: "",
    server_id: "mock.local",
    created_at: new Date().toISOString(),
  };
}

function createChannel(input: {
  id: string;
  name: string;
  description: string;
  type: Channel["type"];
  memberCount: number;
  createdAt: string;
  lastMessage?: Message;
}): Channel {
  return {
    id: input.id,
    type: input.type,
    name: input.name,
    description: input.description,
    invite_code: "",
    owner_id: "",
    is_encrypted: true,
    max_members: 1000,
    member_count: input.memberCount,
    created_at: input.createdAt,
    last_message: input.lastMessage,
  };
}

function createMember(
  user: User,
  role: ChannelMember["role"] = "member"
): ChannelMember {
  return {
    user_id: user.id,
    username: user.username,
    display_name: user.display_name,
    role,
    joined_at: new Date().toISOString(),
    is_muted: false,
  };
}

function createMessage(input: {
  id: string;
  channelId: string;
  senderId: string;
  content: string;
  createdAt?: string;
}): Message {
  const createdAt = input.createdAt ?? new Date().toISOString();
  return {
    id: input.id,
    channel_id: input.channelId,
    sender_id: input.senderId,
    type: "text",
    content: input.content,
    is_edited: false,
    is_deleted: false,
    created_at: createdAt,
    updated_at: createdAt,
    reactions: [],
    status: "sent",
  };
}

function loadDatabase(): MockDatabase {
  try {
    const raw = window.localStorage.getItem(MOCK_DB_KEY);
    if (!raw) {
      const seeded = createSeedDatabase();
      saveDatabase(seeded);
      return seeded;
    }

    return JSON.parse(raw) as MockDatabase;
  } catch {
    const seeded = createSeedDatabase();
    saveDatabase(seeded);
    return seeded;
  }
}

function saveDatabase(database: MockDatabase) {
  window.localStorage.setItem(MOCK_DB_KEY, JSON.stringify(database));
}

function storeChallenge(username: string, challenge: string) {
  window.sessionStorage.setItem(`${MOCK_CHALLENGE_PREFIX}${username.toLowerCase()}`, challenge);
}

function loadChallenge(username: string) {
  return window.sessionStorage.getItem(`${MOCK_CHALLENGE_PREFIX}${username.toLowerCase()}`);
}

function ensureUserChannels(database: MockDatabase, user: User) {
  if (database.channels.some((channel) => channel.name === `DM with ${user.display_name}`)) {
    return database;
  }

  const dmChannelId = crypto.randomUUID();
  const message = createMessage({
    id: crypto.randomUUID(),
    channelId: dmChannelId,
    senderId: user.id,
    content: `Mock account ${user.display_name} is ready.`,
  });

  database.channels.unshift(
    createChannel({
      id: dmChannelId,
      name: `DM with ${user.display_name}`,
      description: "Local mock direct message",
      type: "dm",
      memberCount: 2,
      createdAt: new Date().toISOString(),
      lastMessage: message,
    })
  );
  database.messages[dmChannelId] = [message];
  const peerUser =
    database.users.find((entry) => entry.id !== user.id) ??
    createUser("relay", "Relay Guide");
  if (!database.users.some((entry) => entry.id === peerUser.id)) {
    database.users.push(peerUser);
  }
  database.channelMembers[dmChannelId] = [
    createMember(user, "owner"),
    createMember(peerUser, "member"),
  ];

  saveDatabase(database);
  return database;
}

function issueTokens(user: User): MockAuthResponse {
  return {
    access_token: `mock-access-${user.id}`,
    refresh_token: `mock-refresh-${user.id}`,
    user,
  };
}

export function isLikelyNetworkError(error: unknown) {
  const axiosError = error as AxiosError | undefined;
  return Boolean(
    axiosError &&
      (axiosError.code === "ERR_NETWORK" ||
        axiosError.message === "Network Error" ||
        axiosError.response == null)
  );
}

export async function mockRequestChallenge(username: string) {
  const challenge = crypto.randomUUID().replaceAll("-", "");
  storeChallenge(username, challenge);
  return {
    challenge,
    ttl: 300,
  };
}

export async function mockRegister(payload: {
  username: string;
  displayName: string;
  publicKey: string;
  signature?: string;
}) {
  const database = loadDatabase();
  const existing = database.users.find(
    (user) => user.username.toLowerCase() === payload.username.toLowerCase()
  );
  if (existing) {
    throw new Error("Username already exists in mock mode.");
  }

  const user: User = {
    id: crypto.randomUUID(),
    username: payload.username,
    display_name: payload.displayName,
    public_key: payload.publicKey,
    identity_signing_key: payload.publicKey,
    identity_exchange_key: payload.publicKey,
    avatar_url: "",
    bio: "",
    server_id: "mock.local",
    created_at: new Date().toISOString(),
  };

  database.users.push(user);
  database.devices.unshift({
    id: crypto.randomUUID(),
    user_id: user.id,
    device_key: "mock-device-key",
    device_signing_key: "mock-device-key",
    device_exchange_key: "mock-device-key",
    device_name: "Current browser",
    platform: "web",
    created_at: new Date().toISOString(),
    last_seen: new Date().toISOString(),
    trusted: true,
  });
  saveDatabase(database);
  ensureUserChannels(database, user);

  return issueTokens(user);
}

export async function mockLogin(payload: {
  username: string;
  challenge: string;
}) {
  const database = loadDatabase();
  const expected = loadChallenge(payload.username);
  if (!expected || expected !== payload.challenge) {
    throw new Error("Mock challenge expired. Retry login.");
  }

  const user = database.users.find(
    (entry) => entry.username.toLowerCase() === payload.username.toLowerCase()
  );
  if (!user) {
    throw new Error("User not found in mock mode.");
  }

  ensureUserChannels(database, user);
  return issueTokens(user);
}

export async function mockRefresh(refreshToken: string) {
  const userId = refreshToken.replace(/^mock-refresh-/, "");
  const database = loadDatabase();
  const user = database.users.find((entry) => entry.id === userId);
  if (!user) {
    throw new Error("Mock session expired.");
  }

  return issueTokens(user);
}

export async function mockGetProfile(accessToken: string) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  const user = database.users.find((entry) => entry.id === userId);
  if (!user) {
    throw new Error("Mock profile not found.");
  }

  return { user };
}

export async function mockUpdateProfile(
  accessToken: string,
  payload: { display_name: string; bio?: string; avatar_url?: string }
) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  const nextUsers = database.users.map((entry) =>
    entry.id === userId
      ? {
          ...entry,
          display_name: payload.display_name,
          bio: payload.bio ?? "",
          avatar_url: payload.avatar_url ?? "",
        }
      : entry
  );
  database.users = nextUsers;
  saveDatabase(database);

  const user = nextUsers.find((entry) => entry.id === userId);
  if (!user) {
    throw new Error("Mock profile not found.");
  }

  return { user };
}

export async function mockListDevices(accessToken: string) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  const devices = database.devices
    .filter((device) => device.user_id === userId)
    .map((device, index) => ({
      ...device,
      current: index === 0,
    }));
  return { devices };
}

export async function mockRevokeDevice(accessToken: string, deviceId: string) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  database.devices = database.devices.filter(
    (device, index) =>
      !(device.user_id === userId && device.id === deviceId && index !== 0)
  );
  saveDatabase(database);
}

export async function mockCreatePendingDevice(
  accessToken: string,
  payload: {
    device_signing_key: string;
    device_exchange_key: string;
    device_name: string;
    platform: "web" | "ios" | "android";
  }
) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  const device: Device = {
    id: crypto.randomUUID(),
    user_id: userId,
    device_key: payload.device_signing_key,
    device_signing_key: payload.device_signing_key,
    device_exchange_key: payload.device_exchange_key,
    device_name: payload.device_name,
    platform: payload.platform,
    created_at: new Date().toISOString(),
    last_seen: new Date().toISOString(),
    trusted: false,
  };

  database.devices.push(device);
  saveDatabase(database);

  return { device };
}

export async function mockApproveDevice(accessToken: string, deviceId: string) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  database.devices = database.devices.map((device) =>
    device.user_id === userId && device.id === deviceId
      ? {
          ...device,
          trusted: true,
          approved_by_device_id:
            database.devices.find((entry, index) => entry.user_id === userId && index === 0)?.id ?? "",
        }
      : device
  );
  saveDatabase(database);
}

export async function mockExportBackup(accessToken: string, backupBlob: EncryptedBackupBlob) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  database.backups[userId] = {
    backup_blob: JSON.stringify(backupBlob),
    backup_version: backupBlob.version,
    updated_at: new Date().toISOString(),
  };
  saveDatabase(database);
  return {
    backup_version: backupBlob.version,
    updated_at: database.backups[userId].updated_at,
  };
}

export async function mockImportBackup(accessToken: string) {
  const userId = accessToken.replace(/^mock-access-/, "");
  const database = loadDatabase();
  const backup = database.backups[userId];
  if (!backup) {
    throw new Error("No encrypted backup is stored for this mock account.");
  }

  return {
    ...backup,
    parsed: JSON.parse(backup.backup_blob) as EncryptedBackupBlob,
  };
}

export async function mockListChannels() {
  const database = loadDatabase();
  return {
    channels: [...database.channels].sort((left, right) =>
      (right.last_message?.created_at ?? right.created_at).localeCompare(
        left.last_message?.created_at ?? left.created_at
      )
    ),
  };
}

export async function mockListChannelMembers(channelId: string) {
  const database = loadDatabase();
  return {
    members: database.channelMembers[channelId] ?? [],
  };
}

export async function mockListMessages(params: {
  channelId: string;
  cursor?: string | null;
  limit?: number;
}): Promise<MockMessageListResponse> {
  const database = loadDatabase();
  const all = [...(database.messages[params.channelId] ?? [])].sort((left, right) =>
    right.created_at.localeCompare(left.created_at)
  );
  const limit = params.limit ?? 40;

  const startIndex = params.cursor
    ? all.findIndex((message) => message.created_at < params.cursor!)
    : 0;
  const sliceStart = startIndex < 0 ? all.length : startIndex;
  const messages = all.slice(sliceStart, sliceStart + limit);
  const hasMore = sliceStart + limit < all.length;
  const nextCursor = hasMore ? messages[messages.length - 1]?.created_at : undefined;

  return {
    messages,
    next_cursor: nextCursor,
    has_more: hasMore,
  };
}

export async function mockHandleClientEvent(input: {
  event: WSEvent;
  currentUser: User | null;
}): Promise<WSEvent[]> {
  const { event, currentUser } = input;
  if (!currentUser) {
    return [];
  }

  const database = loadDatabase();

  switch (event.type) {
    case "MESSAGE_SEND": {
      const payload = event.payload as {
        channelId: string;
        content: string;
      };

      const message = createMessage({
        id: crypto.randomUUID(),
        channelId: payload.channelId,
        senderId: currentUser.id,
        content: payload.content,
      });

      database.messages[payload.channelId] = [
        ...(database.messages[payload.channelId] ?? []),
        message,
      ];
      database.channels = database.channels.map((channel) =>
        channel.id === payload.channelId
          ? {
              ...channel,
              last_message: message,
            }
          : channel
      );
      saveDatabase(database);

      return [
        {
          type: "MESSAGE_NEW",
          request_id: event.request_id,
          payload: message,
        },
      ];
    }
    case "TYPING_START":
    case "TYPING_STOP": {
      const payload = event.payload as { channelId: string };
      const typingPayload: TypingPayload = {
        channelId: payload.channelId,
        userId: currentUser.id,
        isTyping: event.type === "TYPING_START",
      };
      return [
        {
          type: "TYPING",
          payload: typingPayload,
        },
      ];
    }
    default:
      return [];
  }
}
