import nacl from "tweetnacl";
import * as naclUtil from "tweetnacl-util";

const AES_KEY_LENGTH = 32;
const AES_GCM_IV_LENGTH = 12;

export async function generateKeyPair(): Promise<{
  publicKey: string;
  privateKey: string;
}> {
  const seed = crypto.getRandomValues(new Uint8Array(32));
  const keyPair = nacl.box.keyPair.fromSecretKey(seed);

  return {
    publicKey: encodeBase64(keyPair.publicKey),
    privateKey: encodeBase64(seed),
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
      info: toArrayBuffer(naclUtil.decodeUTF8("meshchat-channel-key")),
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

function normalizeAESKey(input: Uint8Array): Uint8Array {
  if (input.length === AES_KEY_LENGTH) {
    return input;
  }

  throw new Error("Channel key must be 32 bytes");
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
