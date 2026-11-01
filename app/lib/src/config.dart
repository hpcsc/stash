const apiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://127.0.0.1:3333',
);

const apiPrefix = '/api';

const googleClientId = String.fromEnvironment('GOOGLE_CLIENT_ID');
