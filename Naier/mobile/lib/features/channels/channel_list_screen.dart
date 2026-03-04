import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_client.dart';

class ChannelListScreen extends ConsumerStatefulWidget {
  const ChannelListScreen({super.key});

  @override
  ConsumerState<ChannelListScreen> createState() => _ChannelListScreenState();
}

class _ChannelListScreenState extends ConsumerState<ChannelListScreen> {
  late Future<List<_ChannelSummary>> _channelsFuture;

  @override
  void initState() {
    super.initState();
    _channelsFuture = _loadChannels();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Channels')),
      body: FutureBuilder<List<_ChannelSummary>>(
        future: _channelsFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }

          if (snapshot.hasError) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Text('Failed to load channels.'),
                    const SizedBox(height: 12),
                    FilledButton(
                      onPressed: () {
                        setState(() {
                          _channelsFuture = _loadChannels();
                        });
                      },
                      child: const Text('Retry'),
                    ),
                  ],
                ),
              ),
            );
          }

          final channels = snapshot.data ?? const <_ChannelSummary>[];
          if (channels.isEmpty) {
            return const Center(
              child: Text('No channels yet. Create a DM or join a room.'),
            );
          }

          return RefreshIndicator(
            onRefresh: () async {
              final next = _loadChannels();
              setState(() => _channelsFuture = next);
              await next;
            },
            child: ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: channels.length,
              separatorBuilder: (_, __) => const SizedBox(height: 12),
              itemBuilder: (context, index) {
                final channel = channels[index];
                return Card(
                  child: ListTile(
                    title: Text(channel.title),
                    subtitle: Text(channel.subtitle),
                    trailing: Text(channel.trailing),
                    onTap: () => context.go('/app/channel/${channel.id}'),
                  ),
                );
              },
            ),
          );
        },
      ),
    );
  }

  Future<List<_ChannelSummary>> _loadChannels() async {
    try {
      final dio = ref.read(dioProvider);
      final response = await dio.get<Map<String, dynamic>>('/channels');
      final channels = (response.data?['channels'] as List<dynamic>? ?? const [])
          .whereType<Map<String, dynamic>>()
          .map(_ChannelSummary.fromJson)
          .toList();
      if (channels.isNotEmpty) {
        return channels;
      }
    } catch (_) {
      // Fallback to demo data.
    }

    return const [
      _ChannelSummary(
        id: 'demo',
        title: 'General',
        subtitle: 'Realtime and encrypted messaging UI comes next.',
        trailing: 'demo',
      ),
    ];
  }
}

class _ChannelSummary {
  const _ChannelSummary({
    required this.id,
    required this.title,
    required this.subtitle,
    required this.trailing,
  });

  final String id;
  final String title;
  final String subtitle;
  final String trailing;

  factory _ChannelSummary.fromJson(Map<String, dynamic> json) {
    final lastMessage = json['last_message'] as Map<String, dynamic>?;
    return _ChannelSummary(
      id: json['id']?.toString() ?? 'unknown',
      title: json['name']?.toString().isNotEmpty == true
          ? json['name'].toString()
          : 'Untitled channel',
      subtitle: lastMessage?['content']?.toString() ??
          json['description']?.toString() ??
          '',
      trailing: json['type']?.toString() ?? '',
    );
  }
}
