import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

class MeshWebSocketClient {
  MeshWebSocketClient({
    required this.getToken,
    required this.baseUrl,
    this.onConnected,
  });

  final String? Function() getToken;
  final String baseUrl;
  final Future<void> Function()? onConnected;

  final StreamController<Map<String, dynamic>> _events =
      StreamController<Map<String, dynamic>>.broadcast();
  final List<Map<String, dynamic>> _queuedEvents = <Map<String, dynamic>>[];

  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  Timer? _reconnectTimer;
  int _reconnectAttempts = 0;
  bool _manuallyClosed = false;

  Stream<Map<String, dynamic>> get events => _events.stream;
  bool get isConnected => _channel != null;

  void connect() {
    if (_channel != null) {
      return;
    }

    final token = getToken();
    if (token == null || token.isEmpty) {
      return;
    }

    _manuallyClosed = false;
    final uri = Uri.parse('$baseUrl?token=${Uri.encodeQueryComponent(token)}');
    _channel = WebSocketChannel.connect(uri);
    unawaited(_handleConnected());

    _subscription = _channel!.stream.listen(
      (event) {
        _reconnectAttempts = 0;
        if (event is String && event.isNotEmpty) {
          _events.add(jsonDecode(event) as Map<String, dynamic>);
        }
      },
      onDone: _handleDisconnect,
      onError: (_) => _handleDisconnect(),
      cancelOnError: false,
    );
  }

  void disconnect() {
    _manuallyClosed = true;
    _reconnectTimer?.cancel();
    _subscription?.cancel();
    _subscription = null;
    _channel?.sink.close();
    _channel = null;
  }

  void send(Map<String, dynamic> event) {
    if (_channel == null) {
      _queuedEvents.add(event);
      return;
    }
    final payload = jsonEncode(event);
    _channel?.sink.add(payload);
  }

  void onAppLifecycleChanged(bool isForeground) {
    if (isForeground) {
      connect();
    } else {
      disconnect();
    }
  }

  void dispose() {
    disconnect();
    _events.close();
  }

  void _handleDisconnect() {
    _subscription?.cancel();
    _subscription = null;
    _channel = null;

    if (_manuallyClosed) {
      return;
    }

    _reconnectAttempts += 1;
    final delaySeconds =
        _reconnectAttempts > 5 ? 30 : (1 << (_reconnectAttempts - 1)).clamp(1, 16);
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(Duration(seconds: delaySeconds), connect);
  }

  Future<void> _handleConnected() async {
    for (final event in List<Map<String, dynamic>>.from(_queuedEvents)) {
      _channel?.sink.add(jsonEncode(event));
    }
    _queuedEvents.clear();
    await onConnected?.call();
  }
}
