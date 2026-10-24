import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:stash/main.dart';

void main() {
  testWidgets('renders the inbox', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: StashApp()));

    expect(find.text('Inbox'), findsOneWidget);
  });
}
