import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:stash/src/auth/id_token_provider.dart';
import 'package:stash/src/auth/session.dart';
import 'package:stash/src/screens/sign_in_screen.dart';

import '../fakes.dart';
import '../http_testing.dart';

void main() {
  testWidgets('signs in and stores the token', (tester) async {
    final tokenStore = FakeTokenStore();
    await tester.pumpWidget(signInScope(
      tokenStore: tokenStore,
      child: const MaterialApp(home: SignInScreen()),
    ));

    expect(find.text('Continue with Google'), findsOneWidget);

    await tester.tap(find.text('Continue with Google'));
    await tester.pumpAndSettle();

    expect(tokenStore.stored, 'session-token');
  });

  testWidgets('shows the backend message when sign-in fails', (tester) async {
    await tester.pumpWidget(signInScope(
      tokenStore: FakeTokenStore(),
      response: jsonResponse(
        {'successful': false, 'messages': ['sign-in failed']},
        status: 401,
      ),
      child: const MaterialApp(home: SignInScreen()),
    ));

    await tester.tap(find.text('Continue with Google'));
    await tester.pumpAndSettle();

    expect(find.text('sign-in failed'), findsOneWidget);
  });

  testWidgets('stays quiet when the user cancels', (tester) async {
    final tokenStore = FakeTokenStore();
    await tester.pumpWidget(signInScope(
      tokenStore: tokenStore,
      idTokenSource: CancellingIdTokenSource(),
      child: const MaterialApp(home: SignInScreen()),
    ));

    await tester.tap(find.text('Continue with Google'));
    await tester.pumpAndSettle();

    expect(tokenStore.stored, isNull);
    expect(find.text('Continue with Google'), findsOneWidget);
    expect(find.textContaining('sign-in'), findsNothing);
  });
}

Widget signInScope({
  required FakeTokenStore tokenStore,
  IdTokenSource? idTokenSource,
  http.Response? response,
  required Widget child,
}) {
  return ProviderScope(
    overrides: [
      tokenStoreProvider.overrideWithValue(tokenStore),
      idTokenSourceProvider.overrideWithValue(
        idTokenSource ?? FakeIdTokenSource('google-token'),
      ),
      apiClientProvider.overrideWithValue(
        apiClientWith(
          MockClient((request) async =>
              response ??
              jsonResponse({
                'successful': true,
                'messages': <String>[],
                'data': {'token': 'session-token'},
              })),
        ),
      ),
    ],
    child: child,
  );
}
