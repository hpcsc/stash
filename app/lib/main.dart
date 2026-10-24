import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'src/auth/session.dart';
import 'src/routes.dart';
import 'src/theme.dart';

void main() {
  runApp(const ProviderScope(child: StashApp()));
}

class StashApp extends ConsumerWidget {
  const StashApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: 'Stash',
      theme: stashTheme,
      routerConfig: ref.watch(routerProvider),
      builder: (context, child) {
        if (ref.watch(authControllerProvider).isLoading) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        return child ?? const SizedBox.shrink();
      },
    );
  }
}
