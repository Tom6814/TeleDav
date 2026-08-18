import 'dart:convert';

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
}
