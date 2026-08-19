import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';

import 'api_client.dart';
import 'models.dart';
import 'screens/files_screen.dart';
import 'screens/jobs_screen.dart';
import 'screens/login_screen.dart';
import 'screens/settings_screen.dart';

class NetdiskApp extends StatefulWidget {
  const NetdiskApp({super.key, this.api});

  final ApiClient? api;

  @override
  State<NetdiskApp> createState() => _NetdiskAppState();
}

class _NetdiskAppState extends State<NetdiskApp> {
  late final ApiClient _api = widget.api ?? ApiClient();
  final TextEditingController _passwordController = TextEditingController();
  final TextEditingController _newDirectoryController = TextEditingController();
  final TextEditingController _phoneController = TextEditingController();
  final TextEditingController _codeController = TextEditingController();
  final TextEditingController _telegramPasswordController = TextEditingController();
  final TextEditingController _newChannelController = TextEditingController();
  final TextEditingController _chunkSizeController = TextEditingController();
  final TextEditingController _maxStagingController = TextEditingController();
  final TextEditingController _downloadTtlController = TextEditingController();

  bool _authenticated = false;
  bool _busy = false;
  String? _errorMessage;
  String? _statusMessage;
  TreeResponse? _tree;
  List<PendingJob> _jobs = const [];
  StorageConfig _config = const StorageConfig();
  TelegramAuthStatus _telegramAuth = const TelegramAuthStatus(step: 'disconnected');
  List<TelegramChannel> _telegramChannels = const [];

  @override
  void dispose() {
    _passwordController.dispose();
    _newDirectoryController.dispose();
    _phoneController.dispose();
    _codeController.dispose();
    _telegramPasswordController.dispose();
    _newChannelController.dispose();
    _chunkSizeController.dispose();
    _maxStagingController.dispose();
    _downloadTtlController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    await _runAction(() async {
      await _api.login(_passwordController.text);
      final tree = await _api.fetchDirectory();
      final config = await _api.fetchStorageConfig();
      final jobs = await _api.fetchJobs();
      final telegramAuth = await _api.fetchTelegramAuthStatus();
      final telegramChannels = telegramAuth.connected
          ? await _api.fetchTelegramChannels()
          : const <TelegramChannel>[];
      setState(() {
        _authenticated = true;
        _tree = tree;
        _config = config;
        _jobs = jobs;
        _telegramAuth = telegramAuth;
        _telegramChannels = telegramChannels;
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
      await _refreshTelegramAuth();
    });
  }

  Future<void> _refreshTelegramAuth() async {
    final auth = await _api.fetchTelegramAuthStatus();
    final channels = auth.connected
        ? await _api.fetchTelegramChannels()
        : const <TelegramChannel>[];
    setState(() {
      _telegramAuth = auth;
      _telegramChannels = channels;
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

    final chunkSize = parseInt(_chunkSizeController);
    final maxStaging = parseInt(_maxStagingController);
    final downloadTtl = parseInt(_downloadTtlController);
    if ((_chunkSizeController.text.trim().isNotEmpty && chunkSize == null) ||
        (_maxStagingController.text.trim().isNotEmpty && maxStaging == null) ||
        (_downloadTtlController.text.trim().isNotEmpty && downloadTtl == null)) {
      setState(() {
        _errorMessage = 'Settings fields must be valid integers.';
      });
      return;
    }

    await _runAction(() async {
      await _api.updateStorageConfig(
        defaultChunkSize: chunkSize,
        maxStagingBytes: maxStaging,
        downloadCacheTtlSeconds: downloadTtl,
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

  Future<void> _startTelegramAuth() async {
    final phone = _phoneController.text.trim();
    if (phone.isEmpty) {
      setState(() {
        _errorMessage = 'Phone number is required.';
      });
      return;
    }
    await _runAction(() async {
      final auth = await _api.startTelegramAuth(phone);
      setState(() {
        _telegramAuth = auth;
        _statusMessage = 'Verification code sent.';
      });
    });
  }

  Future<void> _verifyTelegramCode() async {
    final code = _codeController.text.trim();
    if (code.isEmpty) {
      setState(() {
        _errorMessage = 'Verification code is required.';
      });
      return;
    }
    await _runAction(() async {
      final auth = await _api.verifyTelegramCode(code);
      _codeController.clear();
      setState(() {
        _telegramAuth = auth;
        _statusMessage = auth.connected
            ? 'Telegram connected.'
            : 'Verification code accepted.';
      });
      await _refreshConfig();
      await _refreshTelegramAuth();
    });
  }

  Future<void> _verifyTelegramPassword() async {
    final password = _telegramPasswordController.text.trim();
    if (password.isEmpty) {
      setState(() {
        _errorMessage = 'Telegram password is required.';
      });
      return;
    }
    await _runAction(() async {
      final auth = await _api.verifyTelegramPassword(password);
      _telegramPasswordController.clear();
      setState(() {
        _telegramAuth = auth;
        _statusMessage = 'Telegram connected.';
      });
      await _refreshConfig();
      await _refreshTelegramAuth();
    });
  }

  Future<void> _disconnectTelegram() async {
    await _runAction(() async {
      final auth = await _api.disconnectTelegram();
      setState(() {
        _telegramAuth = auth;
        _telegramChannels = const [];
        _statusMessage = 'Telegram disconnected.';
      });
      await _refreshConfig();
    });
  }

  Future<void> _refreshTelegramChannels() async {
    await _runAction(() async {
      await _refreshTelegramAuth();
      setState(() {
        _statusMessage = 'Telegram channels reloaded.';
      });
    });
  }

  Future<void> _selectTelegramChannel(TelegramChannel channel) async {
    await _runAction(() async {
      final auth = await _api.selectTelegramChannel(channel.id);
      setState(() {
        _telegramAuth = auth;
        _statusMessage = 'Storage channel updated.';
      });
      await _refreshConfig();
      await _refreshTelegramAuth();
    });
  }

  Future<void> _createTelegramChannel() async {
    final title = _newChannelController.text.trim();
    if (title.isEmpty) {
      setState(() {
        _errorMessage = 'Channel title is required.';
      });
      return;
    }
    await _runAction(() async {
      await _api.createTelegramChannel(title);
      _newChannelController.clear();
      await _refreshConfig();
      await _refreshTelegramAuth();
      setState(() {
        _statusMessage = 'Dedicated storage channel created.';
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
                      telegramAuth: _telegramAuth,
                      telegramChannels: _telegramChannels,
                      busy: _busy,
                      errorMessage: _errorMessage,
                      statusMessage: _statusMessage,
                      phoneController: _phoneController,
                      codeController: _codeController,
                      telegramPasswordController: _telegramPasswordController,
                      newChannelController: _newChannelController,
                      chunkSizeController: _chunkSizeController,
                      maxStagingController: _maxStagingController,
                      downloadTtlController: _downloadTtlController,
                      onRefresh: _refreshConfig,
                      onSave: _saveSettings,
                      onStartTelegramAuth: _startTelegramAuth,
                      onVerifyTelegramCode: _verifyTelegramCode,
                      onVerifyTelegramPassword: _verifyTelegramPassword,
                      onRefreshTelegramChannels: _refreshTelegramChannels,
                      onSelectTelegramChannel: _selectTelegramChannel,
                      onCreateTelegramChannel: _createTelegramChannel,
                      onDisconnectTelegram: _disconnectTelegram,
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
