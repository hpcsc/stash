import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../api/client.dart';
import 'id_token_provider.dart';
import 'token_store.dart';

final tokenStoreProvider = Provider<TokenStore>((ref) => SecureTokenStore());

final apiClientProvider = Provider<ApiClient>((ref) => ApiClient());

final idTokenSourceProvider = Provider<IdTokenSource>(
  (ref) => GoogleIdTokenSource(),
);

sealed class SessionState {
  const SessionState();
}

class SignedOut extends SessionState {
  const SignedOut();
}

class SignedIn extends SessionState {
  const SignedIn(this.token);

  final String token;
}

final authControllerProvider =
    AsyncNotifierProvider<AuthController, SessionState>(AuthController.new);

class AuthController extends AsyncNotifier<SessionState> {
  @override
  Future<SessionState> build() async {
    final token = await ref.watch(tokenStoreProvider).read();
    if (token == null) {
      return const SignedOut();
    }
    ref.read(apiClientProvider).token = token;
    return SignedIn(token);
  }

  Future<void> signIn() async {
    final idToken = await ref.read(idTokenSourceProvider).idToken();
    final token = await ref.read(apiClientProvider).signIn(idToken: idToken);
    await ref.read(tokenStoreProvider).write(token);
    ref.read(apiClientProvider).token = token;
    state = AsyncData(SignedIn(token));
  }

  Future<void> signOut() async {
    await ref.read(apiClientProvider).signOut();
    await ref.read(tokenStoreProvider).delete();
    ref.read(apiClientProvider).token = null;
    state = const AsyncData(SignedOut());
  }
}
