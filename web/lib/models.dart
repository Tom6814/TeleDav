class LoginRequest {
  const LoginRequest({required this.password});

  final String password;

  Map<String, dynamic> toJson() => {'password': password};
}

class StorageConfig {
  const StorageConfig({
    this.telegramTargetChatId = 0,
    this.defaultChunkSize = 0,
    this.maxStagingBytes = 0,
  });

  final int telegramTargetChatId;
  final int defaultChunkSize;
  final int maxStagingBytes;

  factory StorageConfig.fromJson(Map<String, dynamic> json) {
    return StorageConfig(
      telegramTargetChatId: json['telegram_target_chat_id'] as int? ?? 0,
      defaultChunkSize: json['default_chunk_size'] as int? ?? 0,
      maxStagingBytes: json['max_staging_bytes'] as int? ?? 0,
    );
  }
}

class PendingJob {
  const PendingJob({
    required this.id,
    required this.stage,
    required this.lastError,
  });

  final int id;
  final String stage;
  final String lastError;

  factory PendingJob.fromJson(Map<String, dynamic> json) {
    return PendingJob(
      id: json['id'] as int? ?? 0,
      stage: json['stage'] as String? ?? '',
      lastError: json['last_error'] as String? ?? '',
    );
  }
}
