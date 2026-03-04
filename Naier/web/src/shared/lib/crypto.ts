import nacl from "tweetnacl";
import * as naclUtil from "tweetnacl-util";

const AES_KEY_LENGTH = 32;
const AES_GCM_IV_LENGTH = 12;
const BACKUP_SALT_LENGTH = 16;
const BACKUP_PBKDF2_ITERATIONS = 210_000;

export interface SigningKeyPair {
  publicKey: string;
  privateKey: string;
}

export interface ExchangeKeyPair {
  publicKey: string;
  privateKey: string;
}

export interface KeyBundle {
  identity: {
    signing: SigningKeyPair;
    exchange: ExchangeKeyPair;
  };
  device: {
    signing: SigningKeyPair;
    exchange: ExchangeKeyPair;
  };
}

export interface EncryptedBackupBlob {
  version: number;
  algorithm: "AES-GCM";
  kdf: "PBKDF2-SHA-256";
  iterations: number;
  salt: string;
  iv: string;
  ciphertext: string;
  created_at: string;
}

export async function generateKeyPair(): Promise<ExchangeKeyPair> {
  const seed = crypto.getRandomValues(new Uint8Array(32));
  const keyPair = nacl.box.keyPair.fromSecretKey(seed);

  return {
    publicKey: encodeBase64(keyPair.publicKey),
    privateKey: encodeBase64(seed),
  };
}

export async function generateSigningKeyPair(): Promise<SigningKeyPair> {
  const seed = crypto.getRandomValues(new Uint8Array(32));
  const keyPair = nacl.sign.keyPair.fromSeed(seed);

  return {
    publicKey: encodeBase64(keyPair.publicKey),
    privateKey: encodeBase64(seed),
  };
}

export async function generateKeyBundle(): Promise<KeyBundle> {
  const [identitySigning, identityExchange, deviceSigning, deviceExchange] =
    await Promise.all([
      generateSigningKeyPair(),
      generateKeyPair(),
      generateSigningKeyPair(),
      generateKeyPair(),
    ]);

  return {
    identity: {
      signing: identitySigning,
      exchange: identityExchange,
    },
    device: {
      signing: deviceSigning,
      exchange: deviceExchange,
    },
  };
}

export async function signChallenge(
  challenge: string,
  privateKey: string
): Promise<string> {
  const seed = decodeBase64(privateKey);
  if (seed.length !== 32) {
    throw new Error("Private key seed must be 32 bytes");
  }

  const signingKeyPair = nacl.sign.keyPair.fromSeed(seed);
  const message = naclUtil.decodeUTF8(challenge);
  const signature = nacl.sign.detached(message, signingKeyPair.secretKey);

  return encodeBase64(signature);
}

export async function encryptMessage(
  plaintext: string,
  channelKey: string
): Promise<{ ciphertext: string; iv: string }> {
  const rawKey = normalizeAESKey(decodeBase64(channelKey));
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(rawKey),
    { name: "AES-GCM" },
    false,
    ["encrypt"]
  );

  const iv = crypto.getRandomValues(new Uint8Array(AES_GCM_IV_LENGTH));
  const encoded = naclUtil.decodeUTF8(plaintext);
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: toArrayBuffer(iv) },
    cryptoKey,
    toArrayBuffer(encoded)
  );

  return {
    ciphertext: encodeBase64(new Uint8Array(ciphertext)),
    iv: encodeBase64(iv),
  };
}

export async function decryptMessage(
  ciphertext: string,
  iv: string,
  channelKey: string
): Promise<string> {
  const rawKey = normalizeAESKey(decodeBase64(channelKey));
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(rawKey),
    { name: "AES-GCM" },
    false,
    ["decrypt"]
  );

  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: toArrayBuffer(decodeBase64(iv)) },
    cryptoKey,
    toArrayBuffer(decodeBase64(ciphertext))
  );

  return naclUtil.encodeUTF8(new Uint8Array(plaintext));
}

export async function deriveSharedKey(
  myPrivateKey: string,
  theirPublicKey: string
): Promise<string> {
  const secretKey = decodeBase64(myPrivateKey);
  const publicKey = decodeBase64(theirPublicKey);

  if (secretKey.length !== 32 || publicKey.length !== 32) {
    throw new Error("X25519 keys must be 32 bytes");
  }

  const sharedSecret = nacl.box.before(publicKey, secretKey);
  const hkdfKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(sharedSecret),
    "HKDF",
    false,
    ["deriveBits"]
  );

  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: "HKDF",
      hash: "SHA-256",
      salt: toArrayBuffer(new Uint8Array(32)),
      info: toArrayBuffer(naclUtil.decodeUTF8("naier-channel-key")),
    },
    hkdfKey,
    AES_KEY_LENGTH * 8
  );

  return encodeBase64(new Uint8Array(derivedBits));
}

export function deriveSigningPublicKey(privateKey: string): string {
  const seed = decodeBase64(privateKey);
  if (seed.length !== 32) {
    throw new Error("Private key seed must be 32 bytes");
  }

  const signingKeyPair = nacl.sign.keyPair.fromSeed(seed);
  return encodeBase64(signingKeyPair.publicKey);
}

export function toLegacyKeyPair(bundle: KeyBundle) {
  return {
    publicKey: bundle.identity.signing.publicKey,
    privateKey: bundle.identity.signing.privateKey,
  };
}

export async function encryptKeyBundleBackup(
  keyBundle: KeyBundle,
  passphrase: string
): Promise<EncryptedBackupBlob> {
  if (!passphrase.trim()) {
    throw new Error("Backup passphrase is required.");
  }

  const salt = crypto.getRandomValues(new Uint8Array(BACKUP_SALT_LENGTH));
  const iv = crypto.getRandomValues(new Uint8Array(AES_GCM_IV_LENGTH));
  const cryptoKey = await derivePassphraseKey(passphrase, salt);
  const payload = naclUtil.decodeUTF8(JSON.stringify(keyBundle));
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: toArrayBuffer(iv) },
    cryptoKey,
    toArrayBuffer(payload)
  );

  return {
    version: 1,
    algorithm: "AES-GCM",
    kdf: "PBKDF2-SHA-256",
    iterations: BACKUP_PBKDF2_ITERATIONS,
    salt: encodeBase64(salt),
    iv: encodeBase64(iv),
    ciphertext: encodeBase64(new Uint8Array(ciphertext)),
    created_at: new Date().toISOString(),
  };
}

export async function decryptKeyBundleBackup(
  backup: EncryptedBackupBlob,
  passphrase: string
): Promise<KeyBundle> {
  if (!passphrase.trim()) {
    throw new Error("Backup passphrase is required.");
  }

  const salt = decodeBase64(backup.salt);
  const iv = decodeBase64(backup.iv);
  const key = await derivePassphraseKey(passphrase, salt, backup.iterations);
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: toArrayBuffer(iv) },
    key,
    toArrayBuffer(decodeBase64(backup.ciphertext))
  );

  return JSON.parse(naclUtil.encodeUTF8(new Uint8Array(plaintext))) as KeyBundle;
}

function normalizeAESKey(input: Uint8Array): Uint8Array {
  if (input.length === AES_KEY_LENGTH) {
    return input;
  }

  throw new Error("Channel key must be 32 bytes");
}

async function derivePassphraseKey(
  passphrase: string,
  salt: Uint8Array,
  iterations = BACKUP_PBKDF2_ITERATIONS
) {
  const baseKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(naclUtil.decodeUTF8(passphrase)),
    "PBKDF2",
    false,
    ["deriveKey"]
  );

  return crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt: toArrayBuffer(salt),
      iterations,
      hash: "SHA-256",
    },
    baseKey,
    { name: "AES-GCM", length: AES_KEY_LENGTH * 8 },
    false,
    ["encrypt", "decrypt"]
  );
}

function encodeBase64(value: Uint8Array): string {
  return naclUtil.encodeBase64(value);
}

function decodeBase64(value: string): Uint8Array {
  return naclUtil.decodeBase64(value);
}

function toArrayBuffer(value: Uint8Array): ArrayBuffer {
  return value.buffer.slice(
    value.byteOffset,
    value.byteOffset + value.byteLength
  ) as ArrayBuffer;
}
