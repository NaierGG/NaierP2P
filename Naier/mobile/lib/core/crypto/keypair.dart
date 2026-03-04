import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

class IdentityKeyPair {
  const IdentityKeyPair({
    required this.exchangePublicKey,
    required this.exchangePrivateKey,
    required this.signingPublicKey,
    required this.signingPrivateKey,
  });

  final String exchangePublicKey;
  final String exchangePrivateKey;
  final String signingPublicKey;
  final String signingPrivateKey;

  Map<String, String> toJson() {
    return {
      'exchangePublicKey': exchangePublicKey,
      'exchangePrivateKey': exchangePrivateKey,
      'signingPublicKey': signingPublicKey,
      'signingPrivateKey': signingPrivateKey,
    };
  }

  factory IdentityKeyPair.fromJson(Map<String, String> json) {
    return IdentityKeyPair(
      exchangePublicKey: json['exchangePublicKey'] ?? '',
      exchangePrivateKey: json['exchangePrivateKey'] ?? '',
      signingPublicKey: json['signingPublicKey'] ?? '',
      signingPrivateKey: json['signingPrivateKey'] ?? '',
    );
  }
}

class KeyBundle {
  const KeyBundle({
    required this.identity,
    required this.device,
  });

  final IdentityKeyPair identity;
  final IdentityKeyPair device;

  Map<String, String> toJson() {
    return {
      'identityExchangePublicKey': identity.exchangePublicKey,
      'identityExchangePrivateKey': identity.exchangePrivateKey,
      'identitySigningPublicKey': identity.signingPublicKey,
      'identitySigningPrivateKey': identity.signingPrivateKey,
      'deviceExchangePublicKey': device.exchangePublicKey,
      'deviceExchangePrivateKey': device.exchangePrivateKey,
      'deviceSigningPublicKey': device.signingPublicKey,
      'deviceSigningPrivateKey': device.signingPrivateKey,
    };
  }

  factory KeyBundle.fromJson(Map<String, String> json) {
    return KeyBundle(
      identity: IdentityKeyPair(
        exchangePublicKey: json['identityExchangePublicKey'] ?? '',
        exchangePrivateKey: json['identityExchangePrivateKey'] ?? '',
        signingPublicKey: json['identitySigningPublicKey'] ?? '',
        signingPrivateKey: json['identitySigningPrivateKey'] ?? '',
      ),
      device: IdentityKeyPair(
        exchangePublicKey: json['deviceExchangePublicKey'] ?? '',
        exchangePrivateKey: json['deviceExchangePrivateKey'] ?? '',
        signingPublicKey: json['deviceSigningPublicKey'] ?? '',
        signingPrivateKey: json['deviceSigningPrivateKey'] ?? '',
      ),
    );
  }
}

class KeyPairService {
  const KeyPairService();

  Future<IdentityKeyPair> generateIdentity() async {
    final x25519 = X25519();
    final ed25519 = Ed25519();

    final exchange = await x25519.newKeyPair();
    final exchangePublic = await exchange.extractPublicKey();
    final exchangePrivate = await exchange.extractPrivateKeyBytes();

    final signing = await ed25519.newKeyPair();
    final signingPublic = await signing.extractPublicKey();
    final signingPrivate = await signing.extractPrivateKeyBytes();

    return IdentityKeyPair(
      exchangePublicKey: _encode(exchangePublic.bytes),
      exchangePrivateKey: _encode(exchangePrivate),
      signingPublicKey: _encode(signingPublic.bytes),
      signingPrivateKey: _encode(signingPrivate),
    );
  }

  Future<KeyBundle> generateKeyBundle() async {
    final identity = await generateIdentity();
    final device = await generateIdentity();
    return KeyBundle(identity: identity, device: device);
  }

  Future<String> signChallenge(String challenge, String signingPrivateKey) async {
    final algorithm = Ed25519();
    final keyPair = SimpleKeyPairData(
      _decode(signingPrivateKey),
      type: KeyPairType.ed25519,
    );

    final signature = await algorithm.sign(
      utf8.encode(challenge),
      keyPair: keyPair,
    );

    return _encode(signature.bytes);
  }

  Future<String> deriveSharedKey({
    required String myExchangePrivateKey,
    required String theirExchangePublicKey,
  }) async {
    final algorithm = X25519();
    final localKeyPair = SimpleKeyPairData(
      _decode(myExchangePrivateKey),
      type: KeyPairType.x25519,
    );
    final remoteKey = SimplePublicKey(
      _decode(theirExchangePublicKey),
      type: KeyPairType.x25519,
    );

    final sharedSecret = await algorithm.sharedSecretKey(
      keyPair: localKeyPair,
      remotePublicKey: remoteKey,
    );
    final hkdf = Hkdf(hmac: Hmac.sha256(), outputLength: 32);
    final derived = await hkdf.deriveKey(
      secretKey: sharedSecret,
      info: utf8.encode('naier-channel-key'),
      nonce: Uint8List(32),
    );

    return _encode(await derived.extractBytes());
  }

  String _encode(List<int> bytes) => base64Encode(bytes);
  Uint8List _decode(String value) => Uint8List.fromList(base64Decode(value));
}
