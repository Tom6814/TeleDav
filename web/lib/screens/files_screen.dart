import 'package:flutter/material.dart';

class FilesScreen extends StatelessWidget {
  const FilesScreen({
    super.key,
    required this.directories,
    required this.files,
  });

  final List<String> directories;
  final List<String> files;

  @override
  Widget build(BuildContext context) {
    return ListView(
      children: [
        const ListTile(title: Text('Virtual Filesystem')),
        ...directories.map(
          (directory) => ListTile(
            leading: const Icon(Icons.folder_outlined),
            title: Text(directory),
          ),
        ),
        ...files.map(
          (file) => ListTile(
            leading: const Icon(Icons.insert_drive_file_outlined),
            title: Text(file),
          ),
        ),
      ],
    );
  }
}
