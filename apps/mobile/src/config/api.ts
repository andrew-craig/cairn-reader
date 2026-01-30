// API Configuration
// In production, these would come from environment variables via Expo's config

// Development configuration - using local backend services
export const API_CONFIG = {
  // User service base URL (port 8082)
  USER_SERVICE_URL: 'http://localhost:8082',

  // Recommender service base URL (port 8081)
  RECOMMENDER_SERVICE_URL: 'http://localhost:8081',

  // Read service base URL - Content Service (port 8083)
  READ_SERVICE_URL: 'http://localhost:8083',

  // Request timeout in milliseconds
  REQUEST_TIMEOUT: 30000,
};

// Production configuration:
// export const API_CONFIG = {
//   USER_SERVICE_URL: 'https://cairn.seatrain.net',
//   RECOMMENDER_SERVICE_URL: 'https://cairn.seatrain.net',
//   READ_SERVICE_URL: 'https://cairn.seatrain.net',
//   REQUEST_TIMEOUT: 30000,
// };
