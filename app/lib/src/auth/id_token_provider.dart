import 'package:google_sign_in/google_sign_in.dart';

import '../config.dart';

abstract class IdTokenSource {
  Future<String> idToken();
}

class SignInCancelled implements Exception {
  const SignInCancelled();
}

class SignInFailed implements Exception {
  const SignInFailed(this.message);

  final String message;
}

class GoogleIdTokenSource implements IdTokenSource {
  Future<void>? _initialized;

  Future<void> _ensureInitialized() {
    return _initialized ??= GoogleSignIn.instance.initialize(
      clientId: googleClientId,
      serverClientId: googleClientId,
    );
  }

  @override
  Future<String> idToken() async {
    if (googleClientId.isEmpty) {
      throw const SignInFailed('Google sign-in is not configured');
    }

    await _ensureInitialized();

    final GoogleSignInAccount account;
    try {
      account = await GoogleSignIn.instance.authenticate();
    } on GoogleSignInException catch (e) {
      if (e.code == GoogleSignInExceptionCode.canceled ||
          e.code == GoogleSignInExceptionCode.interrupted ||
          e.code == GoogleSignInExceptionCode.uiUnavailable) {
        throw const SignInCancelled();
      }
      throw const SignInFailed('Google sign-in failed');
    }

    final token = account.authentication.idToken;
    if (token == null) {
      throw const SignInFailed('Google sign-in failed');
    }
    return token;
  }
}
