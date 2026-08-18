import 'package:flutter/material.dart';

import '../models.dart';

class JobsScreen extends StatelessWidget {
  const JobsScreen({
    super.key,
    required this.jobs,
    required this.busy,
    required this.onRefresh,
    required this.onRetry,
    this.errorMessage,
    this.statusMessage,
  });

  final List<PendingJob> jobs;
  final bool busy;
  final String? errorMessage;
  final String? statusMessage;
  final Future<void> Function() onRefresh;
  final Future<void> Function(PendingJob job) onRetry;

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
              onPressed: busy ? null : () => onRefresh(),
              icon: const Icon(Icons.refresh),
              label: const Text('Refresh'),
            ),
          ],
        ),
        if (statusMessage != null) ...[
          const SizedBox(height: 12),
          Text(
            statusMessage!,
            style: TextStyle(color: Theme.of(context).colorScheme.primary),
          )
        ],
        if (errorMessage != null) ...[
          const SizedBox(height: 12),
          Text(
            errorMessage!,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ],
        const SizedBox(height: 16),
        ...jobs.map(
          (job) => ListTile(
            leading: const Icon(Icons.sync),
            title: Text('Job #${job.id}'),
            subtitle: Text(
              'stage=${job.stage} · file=${job.fileId} · last_chunk=${job.lastChunkIndex}'
              '${job.lastError.isEmpty ? '' : '\n${job.lastError}'}',
            ),
            trailing: job.retryable
                ? IconButton(
                    onPressed: busy ? null : () => onRetry(job),
                    icon: const Icon(Icons.replay_outlined),
                  )
                : null,
          ),
        ),
        if (jobs.isEmpty)
          const ListTile(
            leading: Icon(Icons.inbox_outlined),
            title: Text('No pending jobs'),
          ),
      ],
    );
  }
}
