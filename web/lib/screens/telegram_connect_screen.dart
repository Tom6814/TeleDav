import 'package:flutter/material.dart';

import '../models.dart';

class TelegramConnectScreen extends StatelessWidget {
  const TelegramConnectScreen({
    super.key,
    required this.status,
    required this.channels,
    required this.busy,
    required this.phoneController,
    required this.codeController,
    required this.passwordController,
    required this.newChannelController,
    required this.onStart,
    required this.onVerifyCode,
    required this.onVerifyPassword,
    required this.onRefreshChannels,
    required this.onSelectChannel,
    required this.onCreateChannel,
    required this.onDisconnect,
    this.errorMessage,
    this.statusMessage,
  });

  final TelegramAuthStatus status;
  final List<TelegramChannel> channels;
  final bool busy;
  final String? errorMessage;
  final String? statusMessage;
  final TextEditingController phoneController;
  final TextEditingController codeController;
  final TextEditingController passwordController;
  final TextEditingController newChannelController;
  final Future<void> Function() onStart;
  final Future<void> Function() onVerifyCode;
  final Future<void> Function() onVerifyPassword;
  final Future<void> Function() onRefreshChannels;
  final Future<void> Function(TelegramChannel channel) onSelectChannel;
  final Future<void> Function() onCreateChannel;
  final Future<void> Function() onDisconnect;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('Connect Telegram', style: theme.textTheme.headlineSmall),
        const SizedBox(height: 12),
        if (status.connected) ...[
          ListTile(
            contentPadding: EdgeInsets.zero,
            leading: const Icon(Icons.verified_user_outlined),
            title: Text(
              status.user.displayName.isEmpty
                  ? 'Telegram connected'
                  : status.user.displayName,
            ),
            subtitle: Text(
              status.phoneMasked.isEmpty
                  ? 'Authorized session saved'
                  : status.phoneMasked,
            ),
            trailing: OutlinedButton(
              onPressed: busy ? null : () => onDisconnect(),
              child: const Text('Disconnect'),
            ),
          ),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('Current storage channel'),
            subtitle: Text(
              status.selectedChannelTitle.isEmpty
                  ? 'No channel selected yet'
                  : status.selectedChannelTitle,
            ),
          ),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: [
              FilledButton.icon(
                onPressed: busy ? null : () => onRefreshChannels(),
                icon: const Icon(Icons.refresh),
                label: const Text('Reload channels'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          ...channels.map(
            (channel) => ListTile(
              contentPadding: EdgeInsets.zero,
              leading: Icon(
                channel.selected
                    ? Icons.radio_button_checked
                    : Icons.radio_button_off,
              ),
              title: Text(channel.title),
              subtitle: Text('Channel #${channel.id}'),
              trailing: FilledButton(
                onPressed: busy ? null : () => onSelectChannel(channel),
                child: Text(channel.selected ? 'Selected' : 'Use'),
              ),
            ),
          ),
          if (channels.isEmpty)
            const ListTile(
              contentPadding: EdgeInsets.zero,
              title: Text('No channels loaded'),
              subtitle: Text('Reload or create a dedicated storage channel.'),
            ),
          const SizedBox(height: 12),
          TextField(
            controller: newChannelController,
            decoration: const InputDecoration(
              labelText: 'New storage channel title',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: busy ? null : () => onCreateChannel(),
            icon: const Icon(Icons.add),
            label: const Text('Create dedicated channel'),
          ),
        ] else if (status.needsCode) ...[
          Text(
            'Verification code sent to ${status.phoneMasked.isEmpty ? status.phone : status.phoneMasked}',
          ),
          const SizedBox(height: 12),
          TextField(
            controller: codeController,
            decoration: const InputDecoration(
              labelText: 'Verification code',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: busy ? null : () => onVerifyCode(),
            child: const Text('Verify code'),
          ),
        ] else if (status.needsPassword) ...[
          const Text('This Telegram account requires a two-step verification password.'),
          const SizedBox(height: 12),
          TextField(
            controller: passwordController,
            obscureText: true,
            decoration: const InputDecoration(
              labelText: 'Telegram password',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: busy ? null : () => onVerifyPassword(),
            child: const Text('Verify password'),
          ),
        ] else ...[
          const Text(
            'Sign in with your Telegram phone number. The app will then let you choose or create the storage channel.',
          ),
          const SizedBox(height: 12),
          TextField(
            controller: phoneController,
            keyboardType: TextInputType.phone,
            decoration: const InputDecoration(
              labelText: 'Phone number',
              hintText: '+8613800138000',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: busy ? null : () => onStart(),
            child: const Text('Send verification code'),
          ),
        ],
        if (statusMessage != null) ...[
          const SizedBox(height: 12),
          Text(
            statusMessage!,
            style: TextStyle(color: theme.colorScheme.primary),
          ),
        ],
        if (errorMessage != null) ...[
          const SizedBox(height: 12),
          Text(
            errorMessage!,
            style: TextStyle(color: theme.colorScheme.error),
          ),
        ],
      ],
    );
  }
}
