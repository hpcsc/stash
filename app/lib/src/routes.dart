import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'auth/session.dart';
import 'screens/inbox_screen.dart';
import 'screens/library_screen.dart';
import 'screens/sign_in_screen.dart';
import 'screens/trash_screen.dart';

final routerProvider = Provider<GoRouter>((ref) {
  final session = ref.watch(authControllerProvider);

  return GoRouter(
    initialLocation: '/inbox',
    redirect: (context, state) {
      if (session.isLoading) {
        return null;
      }
      final signedIn = session.value is SignedIn;
      final goingToSignIn = state.matchedLocation == '/sign-in';

      if (!signedIn && !goingToSignIn) {
        return '/sign-in';
      }
      if (signedIn && goingToSignIn) {
        return '/inbox';
      }
      return null;
    },
    routes: [
      GoRoute(path: '/sign-in', builder: (_, _) => const SignInScreen()),
      GoRoute(path: '/inbox', builder: (_, _) => const InboxScreen()),
      GoRoute(path: '/library', builder: (_, _) => const LibraryScreen()),
      GoRoute(path: '/trash', builder: (_, _) => const TrashScreen()),
    ],
  );
});
