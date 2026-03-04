import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/crypto/keypair.dart';
import '../../core/network/api_client.dart';
import '../../shared/models/session.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final TextEditingController _usernameController = TextEditingController();
  bool _submitting = false;
  String? _error;

  @override
  void dispose() {
    _usernameController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Naier')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Challenge-based login',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _usernameController,
              decoration: const InputDecoration(
                labelText: 'Username',
                hintText: 'naier',
              ),
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _login(),
            ),
            const SizedBox(height: 16),
            if (_error != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Text(
                  _error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            FilledButton(
              onPressed: _submitting ? null : _login,
              child: Text(_submitting ? 'Signing in...' : 'Sign in'),
            ),
            const SizedBox(height: 12),
            TextButton(
              onPressed: _submitting ? null : () => context.go('/auth/keygen'),
              child: const Text('Create account'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _login() async {
    final username = _usernameController.text.trim();
    if (username.isEmpty) {
      setState(() => _error = 'Username is required.');
      return;
    }

    setState(() {
      _submitting = true;
      _error = null;
    });

    final storage = ref.read(storageProvider);
    final rawKeyBundle = await storage.getIdentityKeyPair();
    if (rawKeyBundle == null) {
      if (!mounted) {
        return;
      }
      setState(() {
        _submitting = false;
        _error = 'No local keys found. Generate a device identity first.';
      });
      return;
    }

    final bundle = KeyBundle.fromJson(rawKeyBundle);
    final dio = ref.read(dioProvider);
    final auth = ref.read(authSessionProvider.notifier);
    final signer = const KeyPairService();

    try {
      final challengeResponse = await dio.post<Map<String, dynamic>>(
        '/auth/challenge',
        data: {
          'username': username,
          'device_signing_key': bundle.device.signingPublicKey,
          'device_name': 'Naier mobile',
          'platform': 'android',
        },
      );
      final challenge = challengeResponse.data?['challenge']?.toString() ?? '';
      if (challenge.isEmpty) {
        throw const FormatException('Missing challenge');
      }

      final deviceSignature = await signer.signChallenge(
        challenge,
        bundle.device.signingPrivateKey,
      );

      final loginResponse = await dio.post<Map<String, dynamic>>(
        '/auth/login',
        data: {
          'username': username,
          'challenge': challenge,
          'device_signature': deviceSignature,
          'device_signing_key': bundle.device.signingPublicKey,
          'device_name': 'Naier mobile',
          'platform': 'android',
        },
      );

      final data = loginResponse.data ?? const <String, dynamic>{};
      final user = data['user'] as Map<String, dynamic>? ?? const <String, dynamic>{};
      await auth.setSession(
        AuthSession(
          userId: user['id']?.toString(),
          username: user['username']?.toString(),
          accessToken: data['access_token']?.toString(),
          refreshToken: data['refresh_token']?.toString(),
          isHydrated: true,
        ),
      );

      if (mounted) {
        context.go(await _resolvePostAuthRoute(dio));
      }
    } on DioException catch (error) {
      setState(() {
        _submitting = false;
        _error = error.response?.data is Map<String, dynamic>
            ? (error.response?.data['message']?.toString() ??
                error.response?.data['error']?.toString() ??
                'Login failed.')
            : 'Login failed.';
      });
      return;
    } catch (_) {
      setState(() {
        _submitting = false;
        _error = 'Login failed.';
      });
      return;
    }

    if (mounted) {
      setState(() => _submitting = false);
    }
  }

  Future<String> _resolvePostAuthRoute(Dio dio) async {
    try {
      final response = await dio.get<Map<String, dynamic>>('/channels');
      final channels = response.data?['channels'] as List<dynamic>? ?? const [];
      if (channels.isNotEmpty) {
        final first = channels.first as Map<String, dynamic>;
        final channelId = first['id']?.toString();
        if (channelId != null && channelId.isNotEmpty) {
          return '/app/channel/$channelId';
        }
      }
    } catch (_) {
      // Fall back to channel list.
    }

    return '/app';
  }
}
