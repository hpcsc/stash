import 'package:flutter/material.dart';

import 'sign_out_button.dart';

class LibraryScreen extends StatelessWidget {
  const LibraryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Library'),
        actions: const [SignOutButton()],
      ),
      body: const Center(child: Text('Library — APP-003')),
    );
  }
}
