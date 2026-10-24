import 'package:flutter/material.dart';

import 'sign_out_button.dart';

class TrashScreen extends StatelessWidget {
  const TrashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Trash'),
        actions: const [SignOutButton()],
      ),
      body: const Center(child: Text('Trash — APP-004')),
    );
  }
}
