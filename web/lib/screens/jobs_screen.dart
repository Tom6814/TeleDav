import 'package:flutter/material.dart';

import '../models.dart';

class JobsScreen extends StatelessWidget {
  const JobsScreen({super.key, required this.jobs});

  final List<PendingJob> jobs;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: jobs
          .map(
            (job) => ListTile(
              leading: const Icon(Icons.sync),
              title: Text('Job #${job.id}'),
              subtitle: Text('${job.stage}${job.lastError.isEmpty ? '' : ' - ${job.lastError}'}'),
            ),
          )
          .toList(),
    );
  }
}
