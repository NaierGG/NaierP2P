import 'package:dio/dio.dart';

import '../storage/secure_storage.dart';

class MeshSyncClient {
  const MeshSyncClient({
    required this.dio,
    required this.storage,
  });

  final Dio dio;
  final SecureStorageService storage;

  Future<Map<String, dynamic>> syncEvents({
    String streamName = 'global',
    String? after,
    int limit = 200,
  }) async {
    final cursor = after ?? await storage.getSyncCursor(streamName);
    final response = await dio.get<Map<String, dynamic>>(
      '/events/sync',
      queryParameters: {
        if (cursor != null && cursor.isNotEmpty) 'after': cursor,
        'limit': limit,
      },
    );

    final data = response.data ?? <String, dynamic>{};
    final lastEventId = data['last_event_id']?.toString();
    if (lastEventId != null && lastEventId.isNotEmpty) {
      await storage.saveSyncCursor(streamName, lastEventId);
    }

    return data;
  }

  Future<Map<String, dynamic>> syncChannel({
    required String channelId,
    String? after,
    int limit = 200,
  }) async {
    final streamName = 'channel:$channelId';
    final cursor = after ?? await storage.getSyncCursor(streamName);
    final response = await dio.get<Map<String, dynamic>>(
      '/channels/$channelId/sync',
      queryParameters: {
        if (cursor != null && cursor.isNotEmpty) 'after': cursor,
        'limit': limit,
      },
    );

    final data = response.data ?? <String, dynamic>{};
    final lastEventId = data['last_event_id']?.toString();
    if (lastEventId != null && lastEventId.isNotEmpty) {
      await storage.saveSyncCursor(streamName, lastEventId);
    }

    return data;
  }
}
