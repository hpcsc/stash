import 'package:go_router/go_router.dart';

import 'screens/inbox_screen.dart';
import 'screens/library_screen.dart';
import 'screens/sign_in_screen.dart';
import 'screens/trash_screen.dart';

final router = GoRouter(
  initialLocation: '/inbox',
  routes: [
    GoRoute(path: '/sign-in', builder: (_, _) => const SignInScreen()),
    GoRoute(path: '/inbox', builder: (_, _) => const InboxScreen()),
    GoRoute(path: '/library', builder: (_, _) => const LibraryScreen()),
    GoRoute(path: '/trash', builder: (_, _) => const TrashScreen()),
  ],
);
