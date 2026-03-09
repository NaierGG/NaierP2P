import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../shared/models/session.dart';
import '../storage/secure_storage.dart';
import 'runtime_config.dart';
import 'sync_client.dart';
import 'websocket_client.dart';

final storageProvider = Provider<SecureStorageService>((ref) {
  throw UnimplementedError('storageProvider must be overridden');
});

final authSessionProvider =
    StateNotifierProvider<AuthSessionController, AuthSession>((ref) {
  final storage = ref.watch(storageProvider);
  return AuthSessionController(storage)..hydrate();
});

final dioProvider = Provider<Dio>((ref) {
  final controller = ref.watch(authSessionProvider.notifier);
  const configuredApiBaseUrl = String.fromEnvironment('MESH_API_BASE_URL');
  final apiBaseUrl =
      configuredApiBaseUrl.isNotEmpty ? configuredApiBaseUrl : defaultApiBaseUrl();

  final dio = Dio(
    BaseOptions(
      baseUrl: apiBaseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 15),
      headers: {'Content-Type': 'application/json'},
    ),
  );

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        final token = ref.read(authSessionProvider).accessToken;
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        final request = error.requestOptions;
        if (error.response?.statusCode != 401 || request.extra['retried'] == true) {
          handler.next(error);
          return;
        }

        final refreshToken = ref.read(authSessionProvider).refreshToken;
        if (refreshToken == null || refreshToken.isEmpty) {
          await controller.clear();
          handler.next(error);
          return;
        }

        try {
          final refreshResponse = await dio.post<Map<String, dynamic>>(
            '/auth/refresh',
            data: {'refresh_token': refreshToken},
          );
          final data = refreshResponse.data ?? const <String, dynamic>{};

          await controller.setSession(
            AuthSession(
              userId: ref.read(authSessionProvider).userId,
              username: ref.read(authSessionProvider).username,
              accessToken: data['access_token'] as String?,
              refreshToken: data['refresh_token'] as String?,
              isHydrated: true,
            ),
          );

          request.extra['retried'] = true;
          request.headers['Authorization'] =
              'Bearer ${data['access_token'] as String? ?? ''}';
          final retried = await dio.fetch(request);
          handler.resolve(retried);
        } catch (_) {
          await controller.clear();
          handler.next(error);
        }
      },
    ),
  );

  return dio;
});

final wsBaseUrlProvider = Provider<String>((ref) {
  const configuredWsBaseUrl = String.fromEnvironment('MESH_WS_BASE_URL');
  if (configuredWsBaseUrl.isNotEmpty) {
    return configuredWsBaseUrl;
  }

  final apiBaseUrl = ref.watch(dioProvider).options.baseUrl;
  return defaultWsBaseUrl(apiBaseUrl);
});

final syncClientProvider = Provider<MeshSyncClient>((ref) {
  return MeshSyncClient(
    dio: ref.watch(dioProvider),
    storage: ref.watch(storageProvider),
  );
});

final websocketClientProvider = Provider<MeshWebSocketClient>((ref) {
  final syncClient = ref.watch(syncClientProvider);
  final client = MeshWebSocketClient(
    getToken: () => ref.read(authSessionProvider).accessToken,
    baseUrl: ref.watch(wsBaseUrlProvider),
    onConnected: () => syncClient.syncEvents(),
  );

  ref.onDispose(client.dispose);
  return client;
});

class AuthSessionController extends StateNotifier<AuthSession> {
  AuthSessionController(this._storage) : super(const AuthSession());

  final SecureStorageService _storage;

  Future<void> hydrate() async {
    final stored = await _storage.getSession();
    if (stored == null) {
      state = state.copyWith(isHydrated: true);
      return;
    }

    state = AuthSession(
      userId: stored['userId'],
      username: stored['username'],
      accessToken: stored['accessToken'],
      refreshToken: stored['refreshToken'],
      isHydrated: true,
    );
  }

  Future<void> setSession(AuthSession session) async {
    state = session.copyWith(isHydrated: true);
    await _storage.saveSession({
      'userId': state.userId,
      'username': state.username,
      'accessToken': state.accessToken,
      'refreshToken': state.refreshToken,
    });
  }

  Future<void> clear() async {
    state = const AuthSession(isHydrated: true);
    await _storage.clearSession();
  }
}
