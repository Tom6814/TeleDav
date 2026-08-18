import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';

import 'api_client.dart';
import 'models.dart';
import 'screens/files_screen.dart';
import 'screens/jobs_screen.dart';
import 'screens/login_screen.dart';
import 'screens/settings_screen.dart';

class NetdiskApp extends StatefulWidget {
  const NetdiskApp({super.key});

  @override
  State<NetdiskApp> createState() => _NetdiskAppState();
}

class _NetdiskAppState extends State<NetdiskApp> {
  final ApiClient _api = ApiClient();
  final TextEditingController _passwordController = TextEditingController();
  final TextEditingController _newDirectoryController = TextEditingController();
  final TextEditingController _chatIdController = TextEditingController();
  final TextEditingController _chunkSizeController = TextEditingController();
  final TextEditingController _maxStagingController = TextEditingController();
  final TextEditingController _downloadTtlController = TextEditingController();
  final TextEditingController _sessionBlobController = TextEditingController();

  bool _authenticated = false;
  bool _busy = false;
  String? _errorMessage;
  String? _statusMessage;
  TreeResponse? _tree;
  List<PendingJob> _jobs = const [];
  StorageConfig _config = const StorageConfig();

  @override
  void dispose() {
    _passwordController.dispose();
    _newDirectoryController.dispose();
    _chatIdController.dispose();
    _chunkSizeController.dispose();
    _maxStagingController.dispose();
    _downloadTtlController.dispose();
    _sessionBlobController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    await _runAction(() async {
      await _api.login(_passwordController.text);
      final tree = await _api.fetchDirectory();
      final config = await _api.fetchStorageConfig();
      final jobs = await _api.fetchJobs();
      setState(() {
        _authenticated = true;
        _tree = tree;
        _config = config;
        _jobs = jobs;
        _errorMessage = null;
      });
      _syncConfigControllers(config);
    });
  }

  Future<void> _runAction(Future<void> Function() action) async {
    setState(() {
      _busy = true;
      _errorMessage = null;
      _statusMessage = null;
    });
    try {
      await action();
    } catch (error) {
      setState(() {
        _errorMessage = error.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _busy = false;
        });
      }
    }
  }

  void _syncConfigControllers(StorageConfig config) {
    _chatIdController.text =
        config.telegramTargetChatId == 0 ? '' : '${config.telegramTargetChatId}';
    _chunkSizeController.text =
        config.defaultChunkSize == 0 ? '' : '${config.defaultChunkSize}';
    _maxStagingController.text =
        config.maxStagingBytes == 0 ? '' : '${config.maxStagingBytes}';
    _downloadTtlController.text = config.downloadCacheTtlSeconds == 0
        ? ''
        : '${config.downloadCacheTtlSeconds}';
  }

  int get _currentDirectoryId => _tree?.directory.id ?? 0;

  Future<void> _refreshTree({int? parentId}) async {
    final tree = await _api.fetchDirectory(parentId: parentId);
    setState(() {
      _tree = tree;
    });
  }

  Future<void> _refreshJobs() async {
    final jobs = await _api.fetchJobs();
    setState(() {
      _jobs = jobs;
    });
  }

  Future<void> _refreshConfig() async {
    final config = await _api.fetchStorageConfig();
    setState(() {
      _config = config;
    });
    _syncConfigControllers(config);
  }

  Future<void> _refreshAll() async {
    await _runAction(() async {
      await _refreshTree(parentId: _tree?.directory.id);
      await _refreshConfig();
      await _refreshJobs();
    });
  }

  Future<void> _createDirectory() async {
    final name = _newDirectoryController.text.trim();
    if (name.isEmpty) {
      setState(() {
        _errorMessage = 'Folder name is required.';
      });
      return;
    }
    await _runAction(() async {
      await _api.createDirectory(parentId: _currentDirectoryId, name: name);
      _newDirectoryController.clear();
      await _refreshTree(parentId: _currentDirectoryId);
      setState(() {
        _statusMessage = 'Folder created.';
      });
    });
  }

  Future<void> _uploadFile() async {
    final result = await FilePicker.platform.pickFiles(withData: true);
    if (result == null || result.files.isEmpty) {
      return;
    }
    final file = result.files.single;
    if (file.bytes == null) {
      setState(() {
        _errorMessage = 'Selected file bytes are unavailable.';
      });
      return;
    }
    await _runAction(() async {
      await _api.uploadBytes(
        parentId: _currentDirectoryId,
        filename: file.name,
        bytes: file.bytes!,
      );
      await _refreshTree(parentId: _currentDirectoryId);
      await _refreshJobs();
      setState(() {
        _statusMessage = 'Upload completed.';
      });
    });
  }

  Future<void> _saveSettings() async {
    int? parseInt(TextEditingController controller) {
      final raw = controller.text.trim();
      if (raw.isEmpty) {
        return null;
      }
      return int.tryParse(raw);
    }

    final chatId = parseInt(_chatIdController);
    final chunkSize = parseInt(_chunkSizeController);
    final maxStaging = parseInt(_maxStagingController);
    final downloadTtl = parseInt(_downloadTtlController);
    if ((_chatIdController.text.trim().isNotEmpty && chatId == null) ||
        (_chunkSizeController.text.trim().isNotEmpty && chunkSize == null) ||
        (_maxStagingController.text.trim().isNotEmpty && maxStaging == null) ||
        (_downloadTtlController.text.trim().isNotEmpty && downloadTtl == null)) {
      setState(() {
        _errorMessage = 'Settings fields must be valid integers.';
      });
      return;
    }

    await _runAction(() async {
      await _api.updateStorageConfig(
        telegramTargetChatId: chatId,
        defaultChunkSize: chunkSize,
        maxStagingBytes: maxStaging,
        downloadCacheTtlSeconds: downloadTtl,
        telegramSessionBlob: _sessionBlobController.text.trim().isEmpty
            ? null
            : _sessionBlobController.text.trim(),
      );
      await _refreshConfig();
      setState(() {
        _statusMessage = 'Settings saved.';
      });
    });
  }

  Future<void> _retryJob(PendingJob job) async {
    await _runAction(() async {
      await _api.retryJob(job.id);
      await _refreshJobs();
      await _refreshTree(parentId: _tree?.directory.id);
      setState(() {
        _statusMessage = 'Job retried.';
      });
    });
  }

  Future<void> _openDirectory(DirectoryEntry directory) async {
    await _runAction(() async {
      await _refreshTree(parentId: directory.id);
    });
  }

  Future<void> _openRoot() async {
    await _runAction(() async {
      await _refreshTree();
    });
  }

  Future<void> _openParent() async {
    final parentId = _tree?.directory.parentId;
    if (parentId == null) {
      await _openRoot();
      return;
    }
    await _runAction(() async {
      await _refreshTree(parentId: parentId);
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Telegram WebDAV Netdisk',
      theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.blue),
      home: _authenticated
          ? DefaultTabController(
              length: 3,
              child: Scaffold(
                appBar: AppBar(
                  title: const Text('Telegram WebDAV Netdisk'),
                  actions: [
                    IconButton(
                      onPressed: _busy ? null : () => _refreshAll(),
                      icon: const Icon(Icons.refresh),
                    ),
                  ],
                  bottom: const TabBar(
                    tabs: [
                      Tab(text: 'Files'),
                      Tab(text: 'Settings'),
                      Tab(text: 'Jobs'),
                    ],
                  ),
                ),
                body: TabBarView(
                  children: [
                    FilesScreen(
                      tree: _tree,
                      busy: _busy,
                      errorMessage: _errorMessage,
                      statusMessage: _statusMessage,
                      newDirectoryController: _newDirectoryController,
                      onRefresh: _refreshAll,
                      onOpenRoot: _openRoot,
                      onOpenParent: _openParent,
                      onOpenDirectory: _openDirectory,
                      onCreateDirectory: _createDirectory,
                      onUpload: _uploadFile,
                    ),
                    SettingsScreen(
                      config: _config,
                      busy: _busy,
                      errorMessage: _errorMessage,
                      statusMessage: _statusMessage,
                      chatIdController: _chatIdController,
                      chunkSizeController: _chunkSizeController,
                      maxStagingController: _maxStagingController,
                      downloadTtlController: _downloadTtlController,
                      sessionBlobController: _sessionBlobController,
                      onRefresh: _refreshConfig,
                      onSave: _saveSettings,
                    ),
                    JobsScreen(
                      jobs: _jobs,
                      busy: _busy,
                      errorMessage: _errorMessage,
                      statusMessage: _statusMessage,
                      onRefresh: _refreshJobs,
                      onRetry: _retryJob,
                    ),
                  ],
                ),
              ),
            )
          : LoginScreen(
              controller: _passwordController,
              onSubmit: _login,
              errorMessage: _errorMessage,
            ),
    );
  }
}
