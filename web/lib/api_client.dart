import 'dart:convert';
import 'dart:typed_data';

import 'package:http/http.dart' as http;

import 'models.dart';

class ApiClient {
  ApiClient({http.Client? httpClient, String baseUrl = ''})
      : _httpClient = httpClient ?? http.Client(),
        _baseUrl = baseUrl;

  final http.Client _httpClient;
  final String _baseUrl;

  Future<void> login(String password) async {
    final response = await _httpClient.post(
      Uri.parse('$_baseUrl/api/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(LoginRequest(password: password).toJson()),
    );
    if (response.statusCode != 204) {
      throw Exception('login failed: ${response.statusCode}');
    }
  }

  Future<Map<String, dynamic>> fetchTree() async {
    final response = await _httpClient.get(Uri.parse('$_baseUrl/api/fs/tree'));
    if (response.statusCode != 200) {
      throw Exception('tree request failed: ${response.statusCode}');
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  Future<TreeResponse> fetchDirectory({int? parentId}) async {
    final uri = Uri.parse('$_baseUrl/api/fs/tree').replace(
      queryParameters: parentId == null ? null : {'parent_id': '$parentId'},
    );
    final response = await _httpClient.get(uri);
    if (response.statusCode != 200) {
      throw Exception('tree request failed: ${response.statusCode}');
    }
    return TreeResponse.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<StorageConfig> fetchStorageConfig() async {
    final response =
        await _httpClient.get(Uri.parse('$_baseUrl/api/config/storage'));
    if (response.statusCode != 200) {
      throw Exception('config request failed: ${response.statusCode}');
    }
    return StorageConfig.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<List<PendingJob>> fetchJobs() async {
    final response = await _httpClient.get(Uri.parse('$_baseUrl/api/jobs'));
    if (response.statusCode != 200) {
      throw Exception('jobs request failed: ${response.statusCode}');
    }
    final data = jsonDecode(response.body) as List<dynamic>;
    return data
        .map((entry) => PendingJob.fromJson(entry as Map<String, dynamic>))
        .toList();
  }

  Future<void> createDirectory({
    required int parentId,
    required String name,
  }) async {
    final response = await _httpClient.post(
      Uri.parse('$_baseUrl/api/fs/mkdir'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'parent_id': parentId, 'name': name}),
    );
    if (response.statusCode != 200) {
      throw Exception('mkdir failed: ${response.statusCode}');
    }
  }

  Future<void> uploadBytes({
    required int parentId,
    required String filename,
    required Uint8List bytes,
  }) async {
    final request = http.MultipartRequest(
      'POST',
      Uri.parse('$_baseUrl/api/fs/upload'),
    );
    request.fields['parent_id'] = '$parentId';
    request.files.add(
      http.MultipartFile.fromBytes('file', bytes, filename: filename),
    );
    final response = await http.Response.fromStream(await _httpClient.send(request));
    if (response.statusCode != 200) {
      throw Exception('upload failed: ${response.statusCode}');
    }
  }

  Future<void> updateStorageConfig({
    int? telegramTargetChatId,
    int? defaultChunkSize,
    int? maxStagingBytes,
    int? downloadCacheTtlSeconds,
    String? telegramSessionBlob,
  }) async {
    final body = <String, dynamic>{};
    if (telegramTargetChatId != null) {
      body['telegram_target_chat_id'] = telegramTargetChatId;
    }
    if (defaultChunkSize != null) {
      body['default_chunk_size'] = defaultChunkSize;
    }
    if (maxStagingBytes != null) {
      body['max_staging_bytes'] = maxStagingBytes;
    }
    if (downloadCacheTtlSeconds != null) {
      body['download_cache_ttl_seconds'] = downloadCacheTtlSeconds;
    }
    if (telegramSessionBlob != null) {
      body['telegram_session_blob'] = telegramSessionBlob;
    }
    final response = await _httpClient.patch(
      Uri.parse('$_baseUrl/api/config/storage'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(body),
    );
    if (response.statusCode != 200) {
      throw Exception('config update failed: ${response.statusCode}');
    }
  }

  Future<void> retryJob(int jobId) async {
    final response = await _httpClient.post(
      Uri.parse('$_baseUrl/api/jobs/$jobId/retry'),
    );
    if (response.statusCode != 204) {
      throw Exception('retry failed: ${response.statusCode}');
    }
  }
}
