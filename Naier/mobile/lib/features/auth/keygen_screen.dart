import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/crypto/keypair.dart';
import '../../core/network/api_client.dart';
import '../../shared/models/session.dart';

class KeygenScreen extends ConsumerStatefulWidget {
  const KeygenScreen({super.key});

  @override
  ConsumerState<KeygenScreen> createState() => _KeygenScreenState();
}

class _KeygenScreenState extends ConsumerState<KeygenScreen> {
  final TextEditingController _usernameController = TextEditingController();
  final TextEditingController _displayNameController = TextEditingController();
  bool _submitting = false;
  String? _error;

  @override
  void dispose() {
    _usernameController.dispose();
    _displayNameController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Key generation')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Your keypair becomes your identity',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _usernameController,
              decoration: const InputDecoration(
                labelText: 'Username',
                hintText: 'naier',
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _displayNameController,
              decoration: const InputDecoration(
                labelText: 'Display name',
                hintText: 'Naier',
              ),
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
              onPressed: _submitting ? null : _generateAndRegister,
              child: Text(_submitting ? 'Creating...' : 'Generate keys and register'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _generateAndRegister() async {
    final username = _usernameController.text.trim();
    final displayName = _displayNameController.text.trim();
    if (username.isEmpty || displayName.isEmpty) {
      setState(() => _error = 'Username and display name are required.');
      return;
    }

    setState(() {
      _submitting = true;
      _error = null;
    });

    final dio = ref.read(dioProvider);
    final storage = ref.read(storageProvider);
    final auth = ref.read(authSessionProvider.notifier);
    final keys = const KeyPairService();

    try {
      final bundle = await keys.generateKeyBundle();
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

      final deviceSignature = await keys.signChallenge(
        challenge,
        bundle.device.signingPrivateKey,
      );
      final identitySignatureOverDevice = await keys.signChallenge(
        '${bundle.device.signingPublicKey}:${bundle.device.exchangePublicKey}',
        bundle.identity.signingPrivateKey,
      );

      final response = await dio.post<Map<String, dynamic>>(
        '/auth/register',
        data: {
          'username': username,
          'display_name': displayName,
          'identity_signing_key': bundle.identity.signingPublicKey,
          'identity_exchange_key': bundle.identity.exchangePublicKey,
          'device_signing_key': bundle.device.signingPublicKey,
          'device_exchange_key': bundle.device.exchangePublicKey,
          'device_signature': deviceSignature,
          'identity_signature_over_device': identitySignatureOverDevice,
          'device_name': 'Naier mobile',
          'platform': 'android',
        },
      );

      final data = response.data ?? const <String, dynamic>{};
      final user = data['user'] as Map<String, dynamic>? ?? const <String, dynamic>{};
      await storage.saveIdentityKeyPair(bundle.toJson());
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
                'Registration failed.')
            : 'Registration failed.';
      });
      return;
    } catch (_) {
      setState(() {
        _submitting = false;
        _error = 'Registration failed.';
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
