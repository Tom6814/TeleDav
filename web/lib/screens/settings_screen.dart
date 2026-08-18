import 'package:flutter/material.dart';

import '../models.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({
    super.key,
    required this.config,
    required this.busy,
    required this.chatIdController,
    required this.chunkSizeController,
    required this.maxStagingController,
    required this.downloadTtlController,
    required this.sessionBlobController,
    required this.onRefresh,
    required this.onSave,
    this.errorMessage,
    this.statusMessage,
  });

  final StorageConfig config;
  final bool busy;
  final String? errorMessage;
  final String? statusMessage;
  final TextEditingController chatIdController;
  final TextEditingController chunkSizeController;
  final TextEditingController maxStagingController;
  final TextEditingController downloadTtlController;
  final TextEditingController sessionBlobController;
  final Future<void> Function() onRefresh;
  final Future<void> Function() onSave;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            FilledButton.icon(
              onPressed: busy ? null : () => onSave(),
              icon: const Icon(Icons.save_outlined),
              label: const Text('Save'),
            ),
            OutlinedButton.icon(
              onPressed: busy ? null : () => onRefresh(),
              icon: const Icon(Icons.refresh),
              label: const Text('Reload'),
            ),
          ],
        ),
        const SizedBox(height: 16),
        ListTile(
          title: const Text('Telegram session'),
          subtitle: Text(
            config.telegramSessionReady ? 'Session saved' : 'Session not saved',
          ),
        ),
        ListTile(
          title: const Text('Application password'),
          subtitle: Text(
            config.applicationPasswordSet ? 'Configured' : 'Not configured',
          ),
        ),
        TextField(
          controller: chatIdController,
          keyboardType: TextInputType.number,
          decoration: const InputDecoration(
            labelText: 'Telegram target chat ID',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: chunkSizeController,
          keyboardType: TextInputType.number,
          decoration: const InputDecoration(
            labelText: 'Default chunk size',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: maxStagingController,
          keyboardType: TextInputType.number,
          decoration: const InputDecoration(
            labelText: 'Max staging bytes',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: downloadTtlController,
          keyboardType: TextInputType.number,
          decoration: const InputDecoration(
            labelText: 'Download cache TTL seconds',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: sessionBlobController,
          minLines: 4,
          maxLines: 8,
          decoration: const InputDecoration(
            labelText: 'Telegram session blob',
            alignLabelWithHint: true,
            border: OutlineInputBorder(),
            helperText: 'Paste a saved session blob to persist it server-side.',
          ),
        ),
        if (statusMessage != null) ...[
          const SizedBox(height: 12),
          Text(
            statusMessage!,
            style: TextStyle(color: Theme.of(context).colorScheme.primary),
          ),
        ],
        if (errorMessage != null) ...[
          const SizedBox(height: 12),
          Text(
            errorMessage!,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ],
      ],
    );
  }
}
