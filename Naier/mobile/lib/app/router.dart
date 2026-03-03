import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../core/network/api_client.dart';
import '../features/auth/keygen_screen.dart';
import '../features/auth/login_screen.dart';
import '../features/channels/channel_detail_screen.dart';
import '../features/channels/channel_list_screen.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  final session = ref.watch(authSessionProvider);

  return GoRouter(
    initialLocation: '/auth/login',
    redirect: (context, state) {
      if (!session.isHydrated) {
        return null;
      }

      final isAuthRoute = state.matchedLocation.startsWith('/auth');
      if (!session.isAuthenticated && !isAuthRoute) {
        return '/auth/login';
      }

      if (session.isAuthenticated && isAuthRoute) {
        return '/app';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/auth/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/auth/keygen',
        builder: (context, state) => const KeygenScreen(),
      ),
      GoRoute(
        path: '/app',
        builder: (context, state) => const ChannelListScreen(),
        routes: [
          GoRoute(
            path: 'channel/:id',
            builder: (context, state) => ChannelDetailScreen(
              channelId: state.pathParameters['id'] ?? 'unknown',
            ),
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => Scaffold(
      appBar: AppBar(title: const Text('Navigation error')),
      body: Center(child: Text(state.error.toString())),
    ),
  );
});
