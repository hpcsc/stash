import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config.dart';

class ApiException implements Exception {
  ApiException(this.message, {this.statusCode});

  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

class ApiClient {
  ApiClient({http.Client? client, String? baseUrl})
      : _client = client ?? http.Client(),
        _baseUrl = baseUrl ?? apiBaseUrl;

  final http.Client _client;
  final String _baseUrl;

  String? token;

  Future<Map<String, dynamic>> _request(
    String method,
    String path, {
    Map<String, dynamic>? body,
  }) async {
    final request = http.Request(method, Uri.parse('$_baseUrl$path'));
    request.headers['Content-Type'] = 'application/json';
    if (token != null) {
      request.headers['Authorization'] = 'Bearer $token';
    }
    if (body != null) {
      request.body = jsonEncode(body);
    }

    final response =
        await _client.send(request).then(http.Response.fromStream);
    return _decode(response);
  }

  Map<String, dynamic> _decode(http.Response response) {
    final Map<String, dynamic> envelope;
    try {
      envelope =
          jsonDecode(utf8.decode(response.bodyBytes)) as Map<String, dynamic>;
    } catch (_) {
      throw ApiException('request failed', statusCode: response.statusCode);
    }

    final messages =
        (envelope['messages'] as List?)?.cast<String>() ?? const <String>[];
    if (envelope['successful'] != true) {
      throw ApiException(
        messages.isEmpty ? 'request failed' : messages.join('\n'),
        statusCode: response.statusCode,
      );
    }
    return envelope;
  }

  Future<String> signIn({required String idToken}) async {
    final envelope = await _request('POST', '/auth/signin', body: {
      'provider': 'google',
      'idToken': idToken,
    });
    return (envelope['data'] as Map<String, dynamic>)['token'] as String;
  }

  Future<void> signOut() async {
    await _request('POST', '/auth/signout');
  }
}
