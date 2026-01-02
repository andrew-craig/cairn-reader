// API Configuration
// In production, these would come from environment variables via Expo's config
// For now, using the production API endpoints

export const API_CONFIG = {
  // User service base URL
  USER_SERVICE_URL: 'https://cairn.seatrain.net',

  // Recommender service base URL
  RECOMMENDER_SERVICE_URL: 'https://cairn.seatrain.net',

  // Read service base URL (Content Service)
  READ_SERVICE_URL: 'https://cairn.seatrain.net',

  // Request timeout in milliseconds
  REQUEST_TIMEOUT: 30000,
};

// For development, you can override these values:
// export const API_CONFIG = {
//   USER_SERVICE_URL: 'http://localhost:8080',
//   RECOMMENDER_SERVICE_URL: 'http://localhost:8081',
//   REQUEST_TIMEOUT: 30000,
// };
