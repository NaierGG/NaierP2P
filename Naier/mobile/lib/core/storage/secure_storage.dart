import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:hive_flutter/hive_flutter.dart';

class SecureStorageService {
  SecureStorageService(this._secureStorage);

  static const _identityKey = 'identity_keypair';
  static const _sessionKey = 'auth_session';
  static const _syncStateKey = 'sync_state';
  static const _encryptedBackupKey = 'encrypted_backup_blob';
  static const _channelBox = 'channel_keys';
  static const _messageBox = 'message_cache';

  final FlutterSecureStorage _secureStorage;

  static Future<SecureStorageService> bootstrap() async {
    await Hive.initFlutter();
    await Future.wait([
      Hive.openBox<String>(_channelBox),
      Hive.openBox<String>(_messageBox),
    ]);

    return SecureStorageService(const FlutterSecureStorage());
  }

  Future<void> saveIdentityKeyPair(Map<String, String> keyPair) async {
    await _secureStorage.write(key: _identityKey, value: jsonEncode(keyPair));
  }

  Future<Map<String, String>?> getIdentityKeyPair() async {
    final value = await _secureStorage.read(key: _identityKey);
    if (value == null || value.isEmpty) {
      return null;
    }

    final decoded = jsonDecode(value) as Map<String, dynamic>;
    return decoded.map(
      (key, value) => MapEntry(key, value.toString()),
    );
  }

  Future<void> clearIdentityKeyPair() async {
    await _secureStorage.delete(key: _identityKey);
  }

  Future<void> saveEncryptedBackup(String payload) async {
    await _secureStorage.write(key: _encryptedBackupKey, value: payload);
  }

  Future<String?> getEncryptedBackup() async {
    return _secureStorage.read(key: _encryptedBackupKey);
  }

  Future<void> clearEncryptedBackup() async {
    await _secureStorage.delete(key: _encryptedBackupKey);
  }

  Future<void> saveSession(Map<String, String?> session) async {
    await _secureStorage.write(key: _sessionKey, value: jsonEncode(session));
  }

  Future<Map<String, String?>?> getSession() async {
    final value = await _secureStorage.read(key: _sessionKey);
    if (value == null || value.isEmpty) {
      return null;
    }

    final decoded = jsonDecode(value) as Map<String, dynamic>;
    return decoded.map((key, value) => MapEntry(key, value?.toString()));
  }

  Future<void> clearSession() async {
    await _secureStorage.delete(key: _sessionKey);
  }

  Future<void> saveSyncCursor(String streamName, String eventId) async {
    final existing = await getSyncState();
    existing[streamName] = eventId;
    await _secureStorage.write(key: _syncStateKey, value: jsonEncode(existing));
  }

  Future<String?> getSyncCursor(String streamName) async {
    final state = await getSyncState();
    return state[streamName];
  }

  Future<Map<String, String>> getSyncState() async {
    final value = await _secureStorage.read(key: _syncStateKey);
    if (value == null || value.isEmpty) {
      return <String, String>{};
    }

    final decoded = jsonDecode(value) as Map<String, dynamic>;
    return decoded.map((key, value) => MapEntry(key, value.toString()));
  }

  Future<void> saveChannelKey(String channelId, String key) async {
    final box = Hive.box<String>(_channelBox);
    await box.put(channelId, key);
  }

  String? getChannelKey(String channelId) {
    final box = Hive.box<String>(_channelBox);
    return box.get(channelId);
  }

  Future<void> cacheMessages(String channelId, String payload) async {
    final box = Hive.box<String>(_messageBox);
    await box.put(channelId, payload);
  }

  String? getCachedMessages(String channelId) {
    final box = Hive.box<String>(_messageBox);
    return box.get(channelId);
  }
}
