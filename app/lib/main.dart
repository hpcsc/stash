import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'src/routes.dart';
import 'src/theme.dart';

void main() {
  runApp(const ProviderScope(child: StashApp()));
}

class StashApp extends StatelessWidget {
  const StashApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Stash',
      theme: stashTheme,
      routerConfig: router,
    );
  }
}
