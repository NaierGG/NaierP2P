import { useCallback, useRef } from "react";

import {
  decryptMessage,
  deriveSharedKey,
  encryptMessage,
  generateKeyPair,
  signChallenge,
} from "@/shared/lib/crypto";
import { keyStore } from "@/shared/lib/keystore";

type KeyPair = { publicKey: string; privateKey: string };

const channelKeyCache = new Map<string, string>();

export function useEncryption() {
  const identityRef = useRef<KeyPair | null>(null);

  const createAndStoreKeyPair = useCallback(async () => {
    const nextKeyPair = await generateKeyPair();
    identityRef.current = nextKeyPair;
    await keyStore.saveKeyPair(nextKeyPair.publicKey, nextKeyPair.privateKey);
    return nextKeyPair;
  }, []);

  const loadKeyPair = useCallback(async () => {
    if (identityRef.current) {
      return identityRef.current;
    }

    const stored = await keyStore.getKeyPair();
    identityRef.current = stored;
    return stored;
  }, []);

  const saveChannelKey = useCallback(async (channelId: string, key: string) => {
    channelKeyCache.set(channelId, key);
    await keyStore.saveChannelKey(channelId, key);
  }, []);

  const loadChannelKey = useCallback(async (channelId: string) => {
    const cached = channelKeyCache.get(channelId);
    if (cached) {
      return cached;
    }

    const stored = await keyStore.getChannelKey(channelId);
    if (stored) {
      channelKeyCache.set(channelId, stored);
    }

    return stored;
  }, []);

  const encryptForChannel = useCallback(async (channelId: string, plaintext: string) => {
    const channelKey = await loadChannelKey(channelId);
    if (!channelKey) {
      throw new Error(`Missing channel key for ${channelId}`);
    }

    return encryptMessage(plaintext, channelKey);
  }, [loadChannelKey]);

  const decryptForChannel = useCallback(async (
    channelId: string,
    ciphertext: string,
    iv: string
  ) => {
    const channelKey = await loadChannelKey(channelId);
    if (!channelKey) {
      throw new Error(`Missing channel key for ${channelId}`);
    }

    return decryptMessage(ciphertext, iv, channelKey);
  }, [loadChannelKey]);

  const deriveDMChannelKey = useCallback(async (theirPublicKey: string) => {
    const keyPair = await loadKeyPair();
    if (!keyPair) {
      throw new Error("Missing local identity keypair");
    }

    return deriveSharedKey(keyPair.privateKey, theirPublicKey);
  }, [loadKeyPair]);

  const signLoginChallenge = useCallback(async (challenge: string) => {
    const keyPair = await loadKeyPair();
    if (!keyPair) {
      throw new Error("Missing local identity keypair");
    }

    return signChallenge(challenge, keyPair.privateKey);
  }, [loadKeyPair]);

  const clearStoredKeys = useCallback(async () => {
    identityRef.current = null;
    channelKeyCache.clear();
    await keyStore.clearAll();
  }, []);

  return {
    createAndStoreKeyPair,
    loadKeyPair,
    saveChannelKey,
    loadChannelKey,
    encryptForChannel,
    decryptForChannel,
    deriveDMChannelKey,
    signLoginChallenge,
    clearStoredKeys,
  };
}
