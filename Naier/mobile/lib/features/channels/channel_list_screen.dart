import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class ChannelListScreen extends StatelessWidget {
  const ChannelListScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Channels')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: ListTile(
              title: const Text('General'),
              subtitle: const Text('Realtime and encrypted messaging UI comes next.'),
              onTap: () => context.go('/app/channel/demo'),
            ),
          ),
        ],
      ),
    );
  }
}
