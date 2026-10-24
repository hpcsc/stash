import 'package:stash/src/auth/id_token_provider.dart';
import 'package:stash/src/auth/token_store.dart';

class FakeTokenStore implements TokenStore {
  FakeTokenStore([this.stored]);

  String? stored;

  @override
  Future<String?> read() async => stored;

  @override
  Future<void> write(String token) async {
    stored = token;
  }

  @override
  Future<void> delete() async {
    stored = null;
  }
}

class FakeIdTokenSource implements IdTokenSource {
  FakeIdTokenSource(this.token);

  final String token;

  @override
  Future<String> idToken() async => token;
}

class CancellingIdTokenSource implements IdTokenSource {
  @override
  Future<String> idToken() async => throw const SignInCancelled();
}
