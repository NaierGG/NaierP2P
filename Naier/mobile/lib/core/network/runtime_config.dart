import 'dart:io';

String defaultApiBaseUrl() {
  if (Platform.isAndroid) {
    return 'http://10.0.2.2:8080/api/v1';
  }

  return 'http://localhost:8080/api/v1';
}

String defaultWsBaseUrl(String apiBaseUrl) {
  final baseUri = Uri.parse(apiBaseUrl);
  final scheme = baseUri.scheme == 'https' ? 'wss' : 'ws';
  final basePath = baseUri.path.replaceFirst(RegExp(r'/api/v1/?$'), '');
  return baseUri.replace(scheme: scheme, path: '$basePath/ws').toString();
}

String currentClientPlatform() {
  if (Platform.isIOS) {
    return 'ios';
  }
  if (Platform.isAndroid) {
    return 'android';
  }

  return 'web';
}
