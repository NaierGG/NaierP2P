import 'dart:async';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

class NotificationRouteIntent {
  const NotificationRouteIntent({
    required this.channelId,
    this.messageId,
  });

  final String channelId;
  final String? messageId;
}

class NotificationService {
  NotificationService({
    FirebaseMessaging? firebaseMessaging,
    FlutterLocalNotificationsPlugin? localNotifications,
  })  : _messaging = firebaseMessaging ?? FirebaseMessaging.instance,
        _localNotifications =
            localNotifications ?? FlutterLocalNotificationsPlugin();

  final FirebaseMessaging _messaging;
  final FlutterLocalNotificationsPlugin _localNotifications;
  final StreamController<NotificationRouteIntent> _tapIntents =
      StreamController<NotificationRouteIntent>.broadcast();

  Stream<NotificationRouteIntent> get tapIntents => _tapIntents.stream;

  Future<void> initialize() async {
    await _messaging.requestPermission(
      alert: true,
      badge: true,
      sound: true,
      provisional: false,
    );

    await _localNotifications.initialize(
      const InitializationSettings(
        android: AndroidInitializationSettings('@mipmap/ic_launcher'),
        iOS: DarwinInitializationSettings(),
      ),
      onDidReceiveNotificationResponse: (response) {
        final intent = _intentFromPayload(response.payload);
        if (intent != null) {
          _tapIntents.add(intent);
        }
      },
    );

    FirebaseMessaging.onMessage.listen(handleForegroundMessage);
    FirebaseMessaging.onMessageOpenedApp.listen((message) {
      final intent = _intentFromData(message.data);
      if (intent != null) {
        _tapIntents.add(intent);
      }
    });
  }

  Future<String?> getPushToken() {
    return _messaging.getToken();
  }

  Future<void> handleForegroundMessage(RemoteMessage message) async {
    final notification = message.notification;
    if (notification == null) {
      return;
    }

    await _localNotifications.show(
      notification.hashCode,
      notification.title,
      notification.body,
      const NotificationDetails(
        android: AndroidNotificationDetails(
          'meshchat_messages',
          'Messages',
          channelDescription: 'Realtime message notifications',
          importance: Importance.high,
          priority: Priority.high,
        ),
        iOS: DarwinNotificationDetails(),
      ),
      payload: _payloadFromData(message.data),
    );
  }

  Future<void> dispose() async {
    await _tapIntents.close();
  }

  String _payloadFromData(Map<String, dynamic> data) {
    final channelId = data['channelId']?.toString() ?? '';
    final messageId = data['messageId']?.toString() ?? '';
    return '$channelId|$messageId';
  }

  NotificationRouteIntent? _intentFromPayload(String? payload) {
    if (payload == null || payload.isEmpty) {
      return null;
    }

    final parts = payload.split('|');
    if (parts.isEmpty || parts.first.isEmpty) {
      return null;
    }

    return NotificationRouteIntent(
      channelId: parts.first,
      messageId: parts.length > 1 && parts[1].isNotEmpty ? parts[1] : null,
    );
  }

  NotificationRouteIntent? _intentFromData(Map<String, dynamic> data) {
    final channelId = data['channelId']?.toString();
    if (channelId == null || channelId.isEmpty) {
      return null;
    }

    return NotificationRouteIntent(
      channelId: channelId,
      messageId: data['messageId']?.toString(),
    );
  }
}
