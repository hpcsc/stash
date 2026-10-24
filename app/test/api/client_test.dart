import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/testing.dart';
import 'package:stash/src/api/client.dart';

import '../http_testing.dart';

void main() {
  group('ApiClient.signIn', () {
    test('returns the session token', () async {
      final client = apiClientWith(MockClient((request) async {
        expect(request.method, 'POST');
        expect(request.url.path, '/auth/signin');
        expect(
          jsonDecode(request.body),
          {'provider': 'google', 'idToken': 'google-token'},
        );
        return jsonResponse({
          'successful': true,
          'messages': <String>[],
          'data': {'token': 'session-token'},
        });
      }));

      final token = await client.signIn(idToken: 'google-token');

      expect(token, 'session-token');
    });

    test('throws the backend message when sign-in fails', () async {
      final client = apiClientWith(MockClient((request) async {
        return jsonResponse(
          {'successful': false, 'messages': ['sign-in failed']},
          status: 401,
        );
      }));

      await expectLater(
        client.signIn(idToken: 'bad-token'),
        throwsA(
          isA<ApiException>().having(
            (e) => e.message,
            'message',
            'sign-in failed',
          ),
        ),
      );
    });
  });

  group('ApiClient.signOut', () {
    test('sends the bearer token', () async {
      final client = apiClientWith(MockClient((request) async {
        expect(request.method, 'POST');
        expect(request.url.path, '/auth/signout');
        expect(request.headers['Authorization'], 'Bearer session-token');
        return jsonResponse({'successful': true, 'messages': <String>[]});
      }));
      client.token = 'session-token';

      await client.signOut();
    });
  });
}
