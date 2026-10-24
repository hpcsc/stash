import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:stash/src/api/client.dart';

http.Response jsonResponse(Object body, {int status = 200}) => http.Response(
      jsonEncode(body),
      status,
      headers: {'content-type': 'application/json'},
    );

ApiClient apiClientWith(http.Client client) =>
    ApiClient(client: client, baseUrl: 'http://test');
