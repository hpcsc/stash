import 'package:flutter/material.dart';

import 'sign_out_button.dart';

class InboxScreen extends StatelessWidget {
  const InboxScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Inbox'),
        actions: const [SignOutButton()],
      ),
      body: const Center(child: Text('Inbox — APP-002')),
    );
  }
}
