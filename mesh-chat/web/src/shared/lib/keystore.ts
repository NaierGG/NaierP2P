import { openDB, type DBSchema, type IDBPDatabase } from "idb";

interface MeshChatKeyDB extends DBSchema {
  keypairs: {
    key: string;
    value: {
      publicKey: string;
      privateKey: string;
      updatedAt: string;
    };
  };
  channelkeys: {
    key: string;
    value: {
      key: string;
      updatedAt: string;
    };
  };
}

export interface KeyStore {
  saveKeyPair(publicKey: string, privateKey: string): Promise<void>;
  getKeyPair(): Promise<{ publicKey: string; privateKey: string } | null>;
  saveChannelKey(channelId: string, key: string): Promise<void>;
  getChannelKey(channelId: string): Promise<string | null>;
  clearAll(): Promise<void>;
}

const DB_NAME = "meshchat-keys";
const DB_VERSION = 1;
const IDENTITY_KEY = "identity";

let dbPromise: Promise<IDBPDatabase<MeshChatKeyDB>> | null = null;

async function getDB() {
  if (!dbPromise) {
    dbPromise = openDB<MeshChatKeyDB>(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains("keypairs")) {
          db.createObjectStore("keypairs");
        }

        if (!db.objectStoreNames.contains("channelkeys")) {
          db.createObjectStore("channelkeys");
        }
      },
    });
  }

  return dbPromise;
}

export const keyStore: KeyStore = {
  async saveKeyPair(publicKey, privateKey) {
    const db = await getDB();

    // IndexedDB persistence improves UX, but these values are still exposed to any successful XSS.
    await db.put("keypairs", {
      publicKey,
      privateKey,
      updatedAt: new Date().toISOString(),
    }, IDENTITY_KEY);
  },

  async getKeyPair() {
    const db = await getDB();
    const result = await db.get("keypairs", IDENTITY_KEY);

    if (!result) {
      return null;
    }

    return {
      publicKey: result.publicKey,
      privateKey: result.privateKey,
    };
  },

  async saveChannelKey(channelId, key) {
    const db = await getDB();

    // Channel keys are persisted for offline reads, so application CSP and script hygiene matter here.
    await db.put("channelkeys", {
      key,
      updatedAt: new Date().toISOString(),
    }, channelId);
  },

  async getChannelKey(channelId) {
    const db = await getDB();
    const result = await db.get("channelkeys", channelId);

    return result?.key ?? null;
  },

  async clearAll() {
    const db = await getDB();
    const tx = db.transaction(["keypairs", "channelkeys"], "readwrite");
    await Promise.all([
      tx.objectStore("keypairs").clear(),
      tx.objectStore("channelkeys").clear(),
    ]);
    await tx.done;
  },
};
