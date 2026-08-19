import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:telegram_webdav_web/api_client.dart';
import 'package:telegram_webdav_web/app.dart';

class _StubClient extends http.BaseClient {
  _StubClient(this.responses);

  final Map<String, http.Response> responses;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final key = '${request.method} ${request.url.path}'
        '${request.url.hasQuery ? '?${request.url.query}' : ''}';
    final response = responses[key] ??
        http.Response('not stubbed: $key', 500,
            headers: {'content-type': 'application/json'});
    return http.StreamedResponse(
      Stream.value(utf8.encode(response.body)),
      response.statusCode,
      headers: response.headers,
    );
  }
}

void main() {
  testWidgets('app renders login shell first', (tester) async {
    await tester.pumpWidget(const NetdiskApp());
    expect(find.text('Telegram WebDAV Netdisk'), findsWidgets);
    expect(find.text('Sign In'), findsWidgets);
  });

  testWidgets(
    'settings surface exposes the full set of controls once authenticated',
    (tester) async {
      final stub = _StubClient({
        'POST /api/login': http.Response('', 204),
        'GET /api/fs/tree': http.Response(
          jsonEncode({
            'root': {
              'id': 1,
              'name': 'root',
              'path': '/',
              'parent_id': null,
            },
            'directory': {
              'id': 1,
              'name': 'root',
              'path': '/',
              'parent_id': null,
            },
            'listing': {
              'directories': <dynamic>[],
              'files': <dynamic>[],
            },
          }),
          200,
          headers: {'content-type': 'application/json'},
        ),
        'GET /api/config/storage': http.Response(
          jsonEncode({
            'telegram_target_chat_id': 0,
            'default_chunk_size': 1024,
            'max_staging_bytes': 2048,
            'download_cache_ttl_seconds': 60,
            'telegram_session_ready': true,
            'application_password_set': true,
          }),
          200,
          headers: {'content-type': 'application/json'},
        ),
        'GET /api/jobs': http.Response('[]', 200,
            headers: {'content-type': 'application/json'}),
        'GET /api/telegram/auth/status': http.Response(
          jsonEncode({
            'step': 'disconnected',
            'connected': false,
          }),
          200,
          headers: {'content-type': 'application/json'},
        ),
        'GET /api/telegram/channels': http.Response('[]', 200,
            headers: {'content-type': 'application/json'}),
      });

      await tester.pumpWidget(
        NetdiskApp(api: ApiClient(httpClient: stub)),
      );
      final passwordFields = find.byType(TextField);
      expect(passwordFields, findsOneWidget);
      await tester.enterText(passwordFields, 'whatever');
      await tester.tap(find.widgetWithText(FilledButton, 'Sign In'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));

      await tester.tap(find.widgetWithText(Tab, 'Settings'));
      await tester.pumpAndSettle();

      // The fixed TelegramConnectScreen now uses a Column inside the
      // outer ListView, so every settings control should be present in
      // the widget tree (the test viewport may scroll, but the widgets
      // themselves are real).
      Future<void> ensureVisible(Finder finder) async {
        await tester.scrollUntilVisible(
          finder,
          200,
          scrollable: find.byType(Scrollable).first,
        );
      }

      await ensureVisible(find.text('Save'));
      expect(find.text('Save'), findsOneWidget);

      await ensureVisible(find.text('Connect Telegram'));
      expect(find.text('Connect Telegram'), findsOneWidget);

      await ensureVisible(find.text('Telegram status'));
      expect(find.text('Telegram status'), findsOneWidget);
      expect(find.text('Session saved'), findsOneWidget);

      await ensureVisible(find.text('Application password'));
      expect(find.text('Application password'), findsOneWidget);

      await ensureVisible(find.text('Default chunk size'));
      expect(find.text('Default chunk size'), findsOneWidget);

      await ensureVisible(find.text('Max staging bytes'));
      expect(find.text('Max staging bytes'), findsOneWidget);

      await ensureVisible(find.text('Download cache TTL seconds'));
      expect(find.text('Download cache TTL seconds'), findsOneWidget);
    },
  );
}
