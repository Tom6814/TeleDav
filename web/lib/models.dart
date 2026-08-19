class LoginRequest {
  const LoginRequest({required this.password});

  final String password;

  Map<String, dynamic> toJson() => {'password': password};
}

class DirectoryEntry {
  const DirectoryEntry({
    required this.id,
    required this.name,
    required this.path,
    this.parentId,
  });

  final int id;
  final int? parentId;
  final String name;
  final String path;

  factory DirectoryEntry.fromJson(Map<String, dynamic> json) {
    return DirectoryEntry(
      id: (json['id'] as num?)?.toInt() ?? 0,
      parentId: (json['parent_id'] as num?)?.toInt(),
      name: json['name'] as String? ?? '',
      path: json['path'] as String? ?? '/',
    );
  }
}

class FileEntryModel {
  const FileEntryModel({
    required this.id,
    required this.parentId,
    required this.name,
    required this.size,
    required this.status,
    required this.source,
  });

  final int id;
  final int parentId;
  final String name;
  final int size;
  final String status;
  final String source;

  factory FileEntryModel.fromJson(Map<String, dynamic> json) {
    return FileEntryModel(
      id: (json['id'] as num?)?.toInt() ?? 0,
      parentId: (json['parent_id'] as num?)?.toInt() ?? 0,
      name: json['name'] as String? ?? '',
      size: (json['size'] as num?)?.toInt() ?? 0,
      status: json['status'] as String? ?? '',
      source: json['source'] as String? ?? '',
    );
  }
}

class TreeResponse {
  const TreeResponse({
    required this.root,
    required this.directory,
    required this.directories,
    required this.files,
  });

  final DirectoryEntry root;
  final DirectoryEntry directory;
  final List<DirectoryEntry> directories;
  final List<FileEntryModel> files;

  factory TreeResponse.fromJson(Map<String, dynamic> json) {
    Map<String, dynamic> asMap(Object? value) {
      if (value is Map<String, dynamic>) {
        return value;
      }
      return const <String, dynamic>{};
    }

    final listing = asMap(json['listing']);
    return TreeResponse(
      root: DirectoryEntry.fromJson(asMap(json['root'])),
      directory: DirectoryEntry.fromJson(
        json['directory'] == null ? asMap(json['root']) : asMap(json['directory']),
      ),
      directories: (listing['directories'] as List<dynamic>? ?? const [])
          .map((entry) => DirectoryEntry.fromJson(entry as Map<String, dynamic>))
          .toList(),
      files: (listing['files'] as List<dynamic>? ?? const [])
          .map((entry) => FileEntryModel.fromJson(entry as Map<String, dynamic>))
          .toList(),
    );
  }
}

class StorageConfig {
  const StorageConfig({
    this.telegramTargetChatId = 0,
    this.defaultChunkSize = 0,
    this.maxStagingBytes = 0,
    this.downloadCacheTtlSeconds = 0,
    this.telegramSessionReady = false,
    this.applicationPasswordSet = false,
  });

  final int telegramTargetChatId;
  final int defaultChunkSize;
  final int maxStagingBytes;
  final int downloadCacheTtlSeconds;
  final bool telegramSessionReady;
  final bool applicationPasswordSet;

  factory StorageConfig.fromJson(Map<String, dynamic> json) {
    return StorageConfig(
      telegramTargetChatId: (json['telegram_target_chat_id'] as num?)?.toInt() ?? 0,
      defaultChunkSize: (json['default_chunk_size'] as num?)?.toInt() ?? 0,
      maxStagingBytes: (json['max_staging_bytes'] as num?)?.toInt() ?? 0,
      downloadCacheTtlSeconds:
          (json['download_cache_ttl_seconds'] as num?)?.toInt() ?? 0,
      telegramSessionReady: json['telegram_session_ready'] as bool? ?? false,
      applicationPasswordSet: json['application_password_set'] as bool? ?? false,
    );
  }
}

class TelegramUser {
  const TelegramUser({
    this.id = 0,
    this.displayName = '',
    this.phoneMasked = '',
  });

  final int id;
  final String displayName;
  final String phoneMasked;

  factory TelegramUser.fromJson(Map<String, dynamic> json) {
    return TelegramUser(
      id: (json['id'] as num?)?.toInt() ?? 0,
      displayName: json['display_name'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
    );
  }
}

class TelegramAuthStatus {
  const TelegramAuthStatus({
    required this.step,
    this.connected = false,
    this.user = const TelegramUser(),
    this.phone = '',
    this.phoneMasked = '',
    this.selectedChannelId = 0,
    this.selectedChannelTitle = '',
  });

  final String step;
  final bool connected;
  final TelegramUser user;
  final String phone;
  final String phoneMasked;
  final int selectedChannelId;
  final String selectedChannelTitle;

  bool get needsPhone => step == 'disconnected';
  bool get needsCode => step == 'code_required';
  bool get needsPassword => step == 'password_required';

  factory TelegramAuthStatus.fromJson(Map<String, dynamic> json) {
    Map<String, dynamic> asMap(Object? value) {
      if (value is Map<String, dynamic>) {
        return value;
      }
      return const <String, dynamic>{};
    }

    return TelegramAuthStatus(
      step: json['step'] as String? ?? 'disconnected',
      connected: json['connected'] as bool? ?? false,
      user: TelegramUser.fromJson(asMap(json['user'])),
      phone: json['phone'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
      selectedChannelId: (json['selected_channel_id'] as num?)?.toInt() ?? 0,
      selectedChannelTitle: json['selected_channel_title'] as String? ?? '',
    );
  }
}

class TelegramChannel {
  const TelegramChannel({
    required this.id,
    required this.title,
    this.selected = false,
  });

  final int id;
  final String title;
  final bool selected;

  factory TelegramChannel.fromJson(Map<String, dynamic> json) {
    return TelegramChannel(
      id: (json['id'] as num?)?.toInt() ?? 0,
      title: json['title'] as String? ?? '',
      selected: json['selected'] as bool? ?? false,
    );
  }
}

class PendingJob {
  const PendingJob({
    required this.id,
    required this.fileId,
    required this.source,
    required this.stage,
    required this.lastError,
    required this.lastChunkIndex,
  });

  final int id;
  final int fileId;
  final String source;
  final String stage;
  final String lastError;
  final int lastChunkIndex;

  bool get retryable => stage == 'failed';

  factory PendingJob.fromJson(Map<String, dynamic> json) {
    return PendingJob(
      id: (json['id'] as num?)?.toInt() ?? 0,
      fileId: (json['file_id'] as num?)?.toInt() ?? 0,
      source: json['source'] as String? ?? '',
      stage: json['stage'] as String? ?? '',
      lastError: json['last_error'] as String? ?? '',
      lastChunkIndex: (json['last_chunk_index'] as num?)?.toInt() ?? 0,
    );
  }
}
