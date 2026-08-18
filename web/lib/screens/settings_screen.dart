import 'package:flutter/material.dart';

import '../models.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key, required this.config});

  final StorageConfig config;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        ListTile(
          title: const Text('Telegram Target Chat ID'),
          subtitle: Text('${config.telegramTargetChatId}'),
        ),
        ListTile(
          title: const Text('Default Chunk Size'),
          subtitle: Text('${config.defaultChunkSize}'),
        ),
        ListTile(
          title: const Text('Max Staging Bytes'),
          subtitle: Text('${config.maxStagingBytes}'),
        ),
      ],
    );
  }
}
