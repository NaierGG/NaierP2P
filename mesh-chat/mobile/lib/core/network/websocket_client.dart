import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

class MeshWebSocketClient {
  MeshWebSocketClient({
    required this.getToken,
    required this.baseUrl,
  });

  final String? Function() getToken;
  final String baseUrl;

  final StreamController<Map<String, dynamic>> _events =
      StreamController<Map<String, dynamic>>.broadcast();

  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  Timer? _reconnectTimer;
  int _reconnectAttempts = 0;
  bool _manuallyClosed = false;

  Stream<Map<String, dynamic>> get events => _events.stream;

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
}
