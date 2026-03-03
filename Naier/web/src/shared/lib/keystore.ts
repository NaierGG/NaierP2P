import { openDB, type DBSchema, type IDBPDatabase } from "idb";

import { deriveSigningPublicKey, type KeyBundle } from "@/shared/lib/crypto";

interface NaierKeyDB extends DBSchema {
  keypairs: {
    key: string;
    value: {
      version?: number;
      publicKey?: string;
      privateKey?: string;
      identity?: {
        signingPublicKey: string;
        signingPrivateKey: string;
        exchangePublicKey: string;
        exchangePrivateKey: string;
      };
      device?: {
        signingPublicKey: string;
        signingPrivateKey: string;
        exchangePublicKey: string;
        exchangePrivateKey: string;
      };
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
  saveKeyBundle(keyBundle: KeyBundle): Promise<void>;
  getKeyBundle(): Promise<KeyBundle | null>;
  saveChannelKey(channelId: string, key: string): Promise<void>;
  getChannelKey(channelId: string): Promise<string | null>;
  clearAll(): Promise<void>;
}

const DB_NAME = "naier-keys";
const DB_VERSION = 2;
const IDENTITY_KEY = "identity";

let dbPromise: Promise<IDBPDatabase<NaierKeyDB>> | null = null;

async function getDB() {
  if (!dbPromise) {
    dbPromise = openDB<NaierKeyDB>(DB_NAME, DB_VERSION, {
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
  async saveKeyBundle(keyBundle) {
    const db = await getDB();

    // IndexedDB persistence improves UX, but these values are still exposed to any successful XSS.
    await db.put("keypairs", {
      version: DB_VERSION,
      identity: {
        signingPublicKey: keyBundle.identity.signing.publicKey,
        signingPrivateKey: keyBundle.identity.signing.privateKey,
        exchangePublicKey: keyBundle.identity.exchange.publicKey,
        exchangePrivateKey: keyBundle.identity.exchange.privateKey,
      },
      device: {
        signingPublicKey: keyBundle.device.signing.publicKey,
        signingPrivateKey: keyBundle.device.signing.privateKey,
        exchangePublicKey: keyBundle.device.exchange.publicKey,
        exchangePrivateKey: keyBundle.device.exchange.privateKey,
      },
      updatedAt: new Date().toISOString(),
    }, IDENTITY_KEY);
  },

  async getKeyBundle() {
    const db = await getDB();
    const result = await db.get("keypairs", IDENTITY_KEY);

    if (!result) {
      return null;
    }

    if (result.identity && result.device) {
      return {
        identity: {
          signing: {
            publicKey: result.identity.signingPublicKey,
            privateKey: result.identity.signingPrivateKey,
          },
          exchange: {
            publicKey: result.identity.exchangePublicKey,
            privateKey: result.identity.exchangePrivateKey,
          },
        },
        device: {
          signing: {
            publicKey: result.device.signingPublicKey,
            privateKey: result.device.signingPrivateKey,
          },
          exchange: {
            publicKey: result.device.exchangePublicKey,
            privateKey: result.device.exchangePrivateKey,
          },
        },
      };
    }

    if (!result.privateKey) {
      return null;
    }

    const signingPublicKey =
      result.publicKey ?? deriveSigningPublicKey(result.privateKey);
    return {
      identity: {
        signing: {
          publicKey: signingPublicKey,
          privateKey: result.privateKey,
        },
        exchange: {
          publicKey: signingPublicKey,
          privateKey: result.privateKey,
        },
      },
      device: {
        signing: {
          publicKey: signingPublicKey,
          privateKey: result.privateKey,
        },
        exchange: {
          publicKey: signingPublicKey,
          privateKey: result.privateKey,
        },
      },
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
