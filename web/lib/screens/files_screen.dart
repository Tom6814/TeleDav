import 'package:flutter/material.dart';

import '../models.dart';

class FilesScreen extends StatelessWidget {
  const FilesScreen({
    super.key,
    required this.tree,
    required this.busy,
    required this.newDirectoryController,
    required this.onRefresh,
    required this.onOpenRoot,
    required this.onOpenParent,
    required this.onOpenDirectory,
    required this.onCreateDirectory,
    required this.onUpload,
    this.errorMessage,
    this.statusMessage,
  });

  final TreeResponse? tree;
  final bool busy;
  final String? errorMessage;
  final String? statusMessage;
  final TextEditingController newDirectoryController;
  final Future<void> Function() onRefresh;
  final Future<void> Function() onOpenRoot;
  final Future<void> Function() onOpenParent;
  final Future<void> Function(DirectoryEntry directory) onOpenDirectory;
  final Future<void> Function() onCreateDirectory;
  final Future<void> Function() onUpload;

  @override
  Widget build(BuildContext context) {
    final current = tree?.directory;
    final directories = tree?.directories ?? const <DirectoryEntry>[];
    final files = tree?.files ?? const <FileEntryModel>[];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        ListTile(
          title: const Text('Virtual Filesystem'),
          subtitle: Text(current == null ? '/' : current.path),
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            FilledButton.icon(
              onPressed: busy ? null : () => onRefresh(),
              icon: const Icon(Icons.refresh),
              label: const Text('Refresh'),
            ),
            FilledButton.icon(
              onPressed: busy ? null : () => onUpload(),
              icon: const Icon(Icons.upload_file),
              label: const Text('Upload'),
            ),
            OutlinedButton.icon(
              onPressed: busy ? null : () => onOpenRoot(),
              icon: const Icon(Icons.home_outlined),
              label: const Text('Root'),
            ),
            OutlinedButton.icon(
              onPressed: busy ? null : () => onOpenParent(),
              icon: const Icon(Icons.arrow_upward),
              label: const Text('Parent'),
            ),
          ],
        ),
        const SizedBox(height: 16),
        TextField(
          controller: newDirectoryController,
          decoration: InputDecoration(
            labelText: 'New folder name',
            border: const OutlineInputBorder(),
            suffixIcon: IconButton(
              onPressed: busy ? null : () => onCreateDirectory(),
              icon: const Icon(Icons.create_new_folder_outlined),
            ),
          ),
          onSubmitted: (_) => onCreateDirectory(),
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
        const SizedBox(height: 16),
        const ListTile(title: Text('Folders')),
        ...directories.map(
          (directory) => ListTile(
            leading: const Icon(Icons.folder_outlined),
            title: Text(directory.name),
            subtitle: Text(directory.path),
            trailing: IconButton(
              onPressed: busy ? null : () => onOpenDirectory(directory),
              icon: const Icon(Icons.chevron_right),
            ),
          ),
        ),
        if (directories.isEmpty)
          const ListTile(
            leading: Icon(Icons.folder_off_outlined),
            title: Text('No folders yet'),
          ),
        const ListTile(title: Text('Files')),
        ...files.map(
          (file) => ListTile(
            leading: const Icon(Icons.insert_drive_file_outlined),
            title: Text(file.name),
            subtitle: Text(
              'size=${file.size} bytes · status=${file.status} · source=${file.source}',
            ),
          ),
        ),
        if (files.isEmpty)
          const ListTile(
            leading: Icon(Icons.file_copy_outlined),
            title: Text('No files in this folder'),
          ),
      ],
    );
  }
}
