import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/testing.dart';
import 'package:stash/src/auth/session.dart';

import '../fakes.dart';
import '../http_testing.dart';

void main() {
  group('AuthController', () {
    test('starts signed out when no token is stored', () async {
      final container = containerWith(
        tokenStore: FakeTokenStore(),
        httpClient: MockClient((request) async =>
            jsonResponse({'successful': true, 'messages': <String>[]})),
      );
      addTearDown(container.dispose);

      final state = await container.read(authControllerProvider.future);

      expect(state, isA<SignedOut>());
    });

    test('starts signed in with the stored token', () async {
      final container = containerWith(
        tokenStore: FakeTokenStore('session-token'),
        httpClient: MockClient((request) async =>
            jsonResponse({'successful': true, 'messages': <String>[]})),
      );
      addTearDown(container.dispose);

      final state = await container.read(authControllerProvider.future);

      expect(state, isA<SignedIn>());
      expect(container.read(apiClientProvider).token, 'session-token');
    });

    test('signIn stores the session token and updates the client', () async {
      final tokenStore = FakeTokenStore();
      final container = containerWith(
        tokenStore: tokenStore,
        httpClient: MockClient((request) async {
          expect(jsonDecode(request.body)['idToken'], 'google-token');
          return jsonResponse({
            'successful': true,
            'messages': <String>[],
            'data': {'token': 'session-token'},
          });
        }),
      );
      addTearDown(container.dispose);
      await container.read(authControllerProvider.future);

      await container.read(authControllerProvider.notifier).signIn();

      expect(container.read(authControllerProvider).value, isA<SignedIn>());
      expect(tokenStore.stored, 'session-token');
      expect(container.read(apiClientProvider).token, 'session-token');
    });

    test('signOut clears the token and signs out', () async {
      final tokenStore = FakeTokenStore('session-token');
      final container = containerWith(
        tokenStore: tokenStore,
        httpClient: MockClient((request) async {
          expect(request.headers['Authorization'], 'Bearer session-token');
          return jsonResponse({'successful': true, 'messages': <String>[]});
        }),
      );
      addTearDown(container.dispose);
      await container.read(authControllerProvider.future);

      await container.read(authControllerProvider.notifier).signOut();

      expect(container.read(authControllerProvider).value, isA<SignedOut>());
      expect(tokenStore.stored, isNull);
      expect(container.read(apiClientProvider).token, isNull);
    });
  });
}

ProviderContainer containerWith({
  required FakeTokenStore tokenStore,
  required MockClient httpClient,
}) {
  return ProviderContainer(
    overrides: [
      tokenStoreProvider.overrideWithValue(tokenStore),
      idTokenSourceProvider.overrideWithValue(FakeIdTokenSource('google-token')),
      apiClientProvider.overrideWithValue(apiClientWith(httpClient)),
    ],
  );
}
