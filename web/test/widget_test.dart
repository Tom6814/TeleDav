import 'package:flutter_test/flutter_test.dart';
import 'package:telegram_webdav_web/app.dart';

void main() {
  testWidgets('app renders login shell first', (tester) async {
    await tester.pumpWidget(const NetdiskApp());
    expect(find.text('Telegram WebDAV Netdisk'), findsOneWidget);
    expect(find.text('Sign In'), findsOneWidget);
  });
}
