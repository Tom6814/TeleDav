import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:telegram_webdav_web/api_client.dart';

class _StubClient extends http.BaseClient {
  _StubClient(this.responses);

  final Map<String, http.Response> responses;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final key = '${request.method} ${request.url.path}'
        '${request.url.hasQuery ? '?${request.url.query}' : ''}';
    final response = responses[key] ??
        http.Response(
          'not stubbed: $key',
          500,
          headers: {'content-type': 'application/json'},
        );
    return http.StreamedResponse(
      Stream.value(utf8.encode(response.body)),
      response.statusCode,
      headers: response.headers,
    );
  }
}

void main() {
  test('fetchJobs treats null payload as empty list', () async {
    final api = ApiClient(
      httpClient: _StubClient({
        'GET /api/jobs': http.Response(
          'null',
          200,
          headers: {'content-type': 'application/json'},
        ),
      }),
    );

    final jobs = await api.fetchJobs();

    expect(jobs, isEmpty);
  });

  test('fetchTelegramChannels treats null payload as empty list', () async {
    final api = ApiClient(
      httpClient: _StubClient({
        'GET /api/telegram/channels': http.Response(
          'null',
          200,
          headers: {'content-type': 'application/json'},
        ),
      }),
    );

    final channels = await api.fetchTelegramChannels();

    expect(channels, isEmpty);
  });
}
