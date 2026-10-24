import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/testing.dart';
import 'package:stash/main.dart';
import 'package:stash/src/auth/session.dart';

import 'fakes.dart';
import 'http_testing.dart';

void main() {
  testWidgets('redirects to sign-in when signed out', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          tokenStoreProvider.overrideWithValue(FakeTokenStore()),
          idTokenSourceProvider.overrideWithValue(FakeIdTokenSource('google-token')),
          apiClientProvider.overrideWithValue(apiClientWith(
            MockClient((request) async =>
                jsonResponse({'successful': true, 'messages': <String>[]})),
          )),
        ],
        child: const StashApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Continue with Google'), findsOneWidget);
  });

  testWidgets('stays on the inbox when signed in', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          tokenStoreProvider.overrideWithValue(FakeTokenStore('session-token')),
          idTokenSourceProvider.overrideWithValue(FakeIdTokenSource('google-token')),
          apiClientProvider.overrideWithValue(apiClientWith(
            MockClient((request) async =>
                jsonResponse({'successful': true, 'messages': <String>[]})),
          )),
        ],
        child: const StashApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Inbox'), findsOneWidget);
  });
}
