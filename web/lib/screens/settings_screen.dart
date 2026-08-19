import 'package:flutter/material.dart';

import '../models.dart';
import 'telegram_connect_screen.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({
    super.key,
    required this.config,
    required this.telegramAuth,
    required this.telegramChannels,
    required this.busy,
    required this.phoneController,
    required this.codeController,
    required this.telegramPasswordController,
    required this.newChannelController,
    required this.chunkSizeController,
    required this.maxStagingController,
    required this.downloadTtlController,
    required this.onRefresh,
    required this.onSave,
    required this.onStartTelegramAuth,
    required this.onVerifyTelegramCode,
    required this.onVerifyTelegramPassword,
    required this.onRefreshTelegramChannels,
    required this.onSelectTelegramChannel,
    required this.onCreateTelegramChannel,
    required this.onDisconnectTelegram,
    this.errorMessage,
    this.statusMessage,
  });

  final StorageConfig config;
  final TelegramAuthStatus telegramAuth;
  final List<TelegramChannel> telegramChannels;
  final bool busy;
  final String? errorMessage;
  final String? statusMessage;
  final TextEditingController phoneController;
  final TextEditingController codeController;
  final TextEditingController telegramPasswordController;
  final TextEditingController newChannelController;
  final TextEditingController chunkSizeController;
  final TextEditingController maxStagingController;
  final TextEditingController downloadTtlController;
  final Future<void> Function() onRefresh;
  final Future<void> Function() onSave;
  final Future<void> Function() onStartTelegramAuth;
  final Future<void> Function() onVerifyTelegramCode;
  final Future<void> Function() onVerifyTelegramPassword;
  final Future<void> Function() onRefreshTelegramChannels;
  final Future<void> Function(TelegramChannel channel) onSelectTelegramChannel;
  final Future<void> Function() onCreateTelegramChannel;
  final Future<void> Function() onDisconnectTelegram;

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
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: TelegramConnectScreen(
              status: telegramAuth,
              channels: telegramChannels,
              busy: busy,
              phoneController: phoneController,
              codeController: codeController,
              passwordController: telegramPasswordController,
              newChannelController: newChannelController,
              onStart: onStartTelegramAuth,
              onVerifyCode: onVerifyTelegramCode,
              onVerifyPassword: onVerifyTelegramPassword,
              onRefreshChannels: onRefreshTelegramChannels,
              onSelectChannel: onSelectTelegramChannel,
              onCreateChannel: onCreateTelegramChannel,
              onDisconnect: onDisconnectTelegram,
              errorMessage: errorMessage,
              statusMessage: statusMessage,
            ),
          ),
        ),
        const SizedBox(height: 16),
        ListTile(
          title: const Text('Telegram status'),
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
      ],
    );
  }
}
