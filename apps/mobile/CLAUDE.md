# CLAUDE.md - Mobile App

This file provides guidance to Claude Code when working with the Cairn mobile application.

## Service Overview

The Cairn mobile app is a React Native application built with Expo that provides a read-it-later experience for iOS and Android. It integrates with the Cairn backend services (User, Explore, and Read services) to provide personalized content recommendations and article management.

**Key Features:**
- 📱 Cross-platform (iOS, Android, Web via Expo)
- 🔐 Authentication with device ID or email/password
- 📖 Article reading and management (Read Later list)
- 🔍 Personalized content discovery (Explore feed)
- ⭐ Favorites and archive functionality
- 🌓 Dark mode support (follows system preference)
- 💾 Local persistence with AsyncStorage
- 🔄 Backend integration with JWT authentication

## Project Structure

```
apps/mobile/
├── App.tsx                          # Entry point with providers
├── app.json                         # Expo configuration
├── package.json                     # Dependencies and scripts
├── tsconfig.json                    # TypeScript configuration
├── .eslintrc.js                     # ESLint configuration
├── assets/                          # Images, fonts, splash screens
│   ├── icon.png                     # App icon
│   ├── splash.png                   # Splash screen
│   └── adaptive-icon.png            # Android adaptive icon
└── src/
    ├── components/                  # Reusable UI components
    │   ├── common/                  # Shared components
    │   │   ├── ArticleCard.tsx      # Card display for articles
    │   │   ├── ArticleRow.tsx       # List row for articles
    │   │   ├── Button.tsx           # Primary button component
    │   │   ├── CustomTabBar.tsx     # Custom tab bar
    │   │   ├── EmptyState.tsx       # Empty state placeholder
    │   │   └── IconButton.tsx       # Icon-only button
    │   ├── AddLinkModal.tsx         # Modal for adding URLs
    │   └── ArticleListScreen.tsx    # Reusable article list
    ├── config/                      # App configuration
    │   └── api.ts                   # API endpoint configuration
    ├── constants/                   # Constants and theme
    │   ├── globalStyles.ts          # Global style definitions
    │   ├── theme.ts                 # Colors, spacing, fonts
    │   └── index.ts                 # Exports
    ├── contexts/                    # React contexts
    │   └── AuthContext.tsx          # Authentication state
    ├── navigation/                  # Navigation setup
    │   ├── RootNavigator.tsx        # Stack navigator (root)
    │   └── TabNavigator.tsx         # Bottom tab navigator
    ├── screens/                     # App screens
    │   ├── AddArticleScreen.tsx     # Add new article/URL
    │   ├── ArchiveScreen.tsx        # Archived articles
    │   ├── ArticleDetailScreen.tsx  # Read article details
    │   ├── ExploreArticleDetailScreen.tsx  # Explore article view
    │   ├── ExploreScreen.tsx        # Content discovery feed
    │   ├── FavoritesScreen.tsx      # Favorited articles
    │   ├── LoginScreen.tsx          # Authentication
    │   ├── ReadScreen.tsx           # Reading list (main)
    │   ├── SettingsScreen.tsx       # App settings
    │   └── index.ts                 # Exports
    ├── services/                    # Service layer (API clients)
    │   ├── auth.ts                  # Authentication service
    │   ├── explore.ts               # Explore/recommendations API
    │   ├── read.ts                  # Read service API
    │   ├── storage.ts               # Local storage (AsyncStorage)
    │   └── index.ts                 # Exports
    ├── types/                       # TypeScript type definitions
    │   ├── article.ts               # Article interface
    │   ├── auth.ts                  # Auth types
    │   ├── navigation.ts            # Navigation param types
    │   ├── read.ts                  # Read service types
    │   └── index.ts                 # Exports
    └── utils/                       # Utility functions
        ├── helpers.ts               # Helper functions
        └── index.ts                 # Exports
```

## Architecture and Patterns

### Navigation Architecture
The app uses React Navigation with a stack + tabs pattern:

```
RootNavigator (Stack)
├── MainTabs (Bottom Tabs)
│   ├── Explore Tab → ExploreScreen
│   ├── Read Tab → ReadScreen
│   └── Settings Tab → SettingsScreen
├── ArticleDetail (Modal)
├── ExploreArticleDetail (Modal)
└── AddArticle (Modal)
```

**Navigation Types:**
- All navigation params are typed in `src/types/navigation.ts`
- Use `RootStackParamList` for stack navigation
- Use `MainTabParamList` for tab navigation

### State Management
The app uses React hooks and Context API for state management:

1. **AuthContext** (`src/contexts/AuthContext.tsx`):
   - Manages authentication state
   - Provides login, logout, and user info
   - Handles token refresh automatically

2. **Local State**:
   - Component-level state with `useState`
   - Side effects with `useEffect`

3. **AsyncStorage**:
   - Persistent local storage for articles, tokens, and user data
   - Accessed via `StorageService` (see Services section)

### Service Layer Pattern
All backend communication goes through service classes:

- **AuthService** (`src/services/auth.ts`): Authentication and token management
- **ExploreService** (`src/services/explore.ts`): Content recommendations and voting
- **ReadService** (`src/services/read.ts`): Article storage and management
- **StorageService** (`src/services/storage.ts`): Local data persistence

**Key Patterns:**
- Services are static classes (no instantiation needed)
- All API calls include automatic token refresh on 401 errors
- Services transform backend models to mobile `Article` interface
- Error handling with try/catch and console logging

### Component Patterns
- **Functional components** with hooks (no class components)
- **TypeScript** interfaces for all props
- **Styled components** using React Native StyleSheet
- **Theme system** for colors, spacing, and typography

## Key Components and Screens

### Common Components

**ArticleCard.tsx** - Card-based article display
```typescript
interface ArticleCardProps {
  article: Article;
  onPress: () => void;
  onToggleFavorite?: () => void;
  onArchive?: () => void;
}
```

**ArticleRow.tsx** - List row for articles
```typescript
interface ArticleRowProps {
  article: Article;
  onPress: () => void;
  showOptions?: boolean;
}
```

**Button.tsx** - Primary button component
```typescript
interface ButtonProps {
  title: string;
  onPress: () => void;
  variant?: 'primary' | 'secondary' | 'outline';
  disabled?: boolean;
}
```

**EmptyState.tsx** - Empty state placeholder
```typescript
interface EmptyStateProps {
  icon: string;              // Ionicons name
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}
```

### Main Screens

**ExploreScreen.tsx** - Content discovery feed
- Fetches recommendations from Explore service
- Displays articles in card format
- Supports upvote/downvote actions
- Pull-to-refresh functionality

**ReadScreen.tsx** - Reading list (main screen)
- Displays user's saved articles
- Tabs for Unread/Archive/Favorites
- Search and filter functionality
- Integrates with Read service

**ArticleDetailScreen.tsx** - Full article view
- Displays article content in WebView
- Reading progress tracking
- Mark as read/unread
- Favorite/unfavorite actions
- Archive functionality

**LoginScreen.tsx** - Authentication
- Device ID login (automatic on first launch)
- Email/password login/registration
- Account upgrade (device → email/password)

**SettingsScreen.tsx** - App settings
- User profile information
- Logout functionality
- App information
- Future: Reading preferences, notifications

## Data Models and Types

### Core Article Interface
Located in `src/types/article.ts`:

```typescript
interface Article {
  id: string;              // Unique identifier
  url: string;             // Source URL
  title: string;
  description?: string;
  imageUrl?: string;
  author?: string;
  publishedDate?: string;  // ISO 8601 date string
  readingTime?: number;    // Estimated minutes
  tags: string[];
  isRead: boolean;
  isFavorite: boolean;
  addedAt: number;         // Unix timestamp (ms)
  readAt?: number;         // Unix timestamp (ms)
  notes?: string;
}
```

### Navigation Types
Located in `src/types/navigation.ts`:

```typescript
type RootStackParamList = {
  MainTabs: undefined;
  ArticleDetail: { article: Article };
  ExploreArticleDetail: { article: Article };
  AddArticle: undefined;
};

type MainTabParamList = {
  Explore: undefined;
  Read: undefined;
  Settings: undefined;
};
```

### Authentication Types
Located in `src/types/auth.ts`:

```typescript
interface User {
  id: string;
  email?: string;
  expo_device_id?: string;
  created_at: string;
  updated_at: string;
}

interface AuthTokens {
  accessToken: string;
  refreshToken: string;
}

interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}
```

## Services

### StorageService (`src/services/storage.ts`)
Local persistence using AsyncStorage.

**Methods:**
```typescript
StorageService.getArticles(): Promise<Article[]>
StorageService.saveArticles(articles: Article[]): Promise<void>
StorageService.addArticle(article: Article): Promise<void>
StorageService.updateArticle(id: string, updates: Partial<Article>): Promise<void>
StorageService.deleteArticle(id: string): Promise<void>
StorageService.clearAllArticles(): Promise<void>
```

**Storage Key:**
- Articles: `@readitlater:articles`

### AuthService (`src/services/auth.ts`)
Authentication and token management.

**Methods:**
```typescript
AuthService.initialize(): Promise<void>
AuthService.loginWithDevice(): Promise<LoginResponse>
AuthService.registerWithDevice(): Promise<LoginResponse>
AuthService.loginWithEmail(credentials: LoginRequest): Promise<LoginResponse>
AuthService.registerWithEmail(credentials: RegisterRequest): Promise<LoginResponse>
AuthService.logout(): Promise<void>
AuthService.getAccessToken(): Promise<string | null>
AuthService.isAuthenticated(): Promise<boolean>
AuthService.refreshAccessToken(): Promise<void>
AuthService.getUser(): Promise<User | null>
AuthService.getUserId(): Promise<string | null>
```

**Storage Keys:**
- Access Token: `@cairn:access_token`
- Refresh Token: `@cairn:refresh_token`
- User Data: `@cairn:user`

**Device ID:**
- iOS: Uses `expo-application.getIosIdForVendorAsync()`
- Android: Uses `expo-application.getAndroidId()`

### ExploreService (`src/services/explore.ts`)
Content discovery and recommendations.

**Methods:**
```typescript
ExploreService.getRecommendations(): Promise<Article[]>
ExploreService.markAsRead(articleId: string): Promise<void>
ExploreService.upvoteArticle(articleId: string): Promise<void>
ExploreService.downvoteArticle(articleId: string): Promise<void>
ExploreService.removeVote(articleId: string): Promise<void>
ExploreService.getVoteCounts(articleId: string): Promise<{upvotes, downvotes, user_vote?}>
```

**Backend Integration:**
- Endpoint: `${RECOMMENDER_SERVICE_URL}/api/v1/explore/...`
- Requires JWT authentication
- Transforms `BackendArticle` → `Article` interface

### ReadService (`src/services/read.ts`)
Article storage and reading list management.

**Methods:**
```typescript
ReadService.listUserContents(params?: ListContentsParams): Promise<UserContentsListResponse>
ReadService.searchUserContents(params: SearchParams): Promise<UserContentsListResponse>
ReadService.addContentToUser(request: AddContentToUserRequest): Promise<UserContentResponse>
ReadService.updateUserContent(contentId: string, updates: UpdateUserContentRequest): Promise<UserContentResponse>
ReadService.deleteUserContent(contentId: string): Promise<void>
ReadService.detectURL(url: string): Promise<DetectURLResponse>
ReadService.addURL(request: AddURLRequest): Promise<AddURLResponse>
```

**Backend Integration:**
- Endpoint: `${READ_SERVICE_URL}/api/v1/content/...`
- Requires JWT authentication
- Supports pagination (limit/offset)
- URL detection (feed vs page) with 10s timeout

## Navigation

### Setup
Navigation is configured in two files:

1. **RootNavigator.tsx** - Stack navigator (modals)
2. **TabNavigator.tsx** - Bottom tabs (main app)

### Navigation Hooks
```typescript
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { RootStackParamList } from '../types/navigation';

type NavigationProp = StackNavigationProp<RootStackParamList>;

const navigation = useNavigation<NavigationProp>();
navigation.navigate('ArticleDetail', { article });
```

### Route Params
```typescript
import { useRoute, RouteProp } from '@react-navigation/native';

type ArticleDetailRouteProp = RouteProp<RootStackParamList, 'ArticleDetail'>;

const route = useRoute<ArticleDetailRouteProp>();
const { article } = route.params;
```

## Styling and Theming

### Theme System
Theme constants defined in `src/constants/theme.ts`:

**Colors:**
```typescript
Colors.light.primary        // #0F0C0B
Colors.light.background     // #FDFCFC
Colors.light.text           // #0F0C0B
Colors.light.textSecondary  // #696563

Colors.dark.primary         // #FDFCFC
Colors.dark.background      // #0F0C0B
Colors.dark.text            // #FDFCFC
Colors.dark.textSecondary   // #8E8E93
```

**Spacing:**
```typescript
Spacing.xs    // 4
Spacing.sm    // 8
Spacing.md    // 16
Spacing.lg    // 24
Spacing.xl    // 32
Spacing.xxl   // 48
```

**Typography:**
```typescript
FontSizes.xs   // 12
FontSizes.sm   // 14
FontSizes.md   // 16
FontSizes.lg   // 18
FontSizes.xl   // 24
FontSizes.xxl  // 32

FontFamily.default          // Inter_400Regular
FontFamily.defaultMedium    // Inter_500Medium
FontFamily.defaultSemiBold  // Inter_600SemiBold
FontFamily.defaultBold      // Inter_700Bold
FontFamily.heading          // CrimsonPro_400Regular
FontFamily.headingBold      // CrimsonPro_700Bold
```

**Border Radius:**
```typescript
BorderRadius.sm    // 4
BorderRadius.md    // 8
BorderRadius.lg    // 12
BorderRadius.xl    // 16
BorderRadius.full  // 9999
```

### Using Theme
```typescript
import { Colors, Spacing, FontSizes, FontFamily } from '../constants/theme';
import { useColorScheme } from 'react-native';

function MyComponent() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? 'light'];

  const styles = StyleSheet.create({
    container: {
      backgroundColor: colors.background,
      padding: Spacing.md,
    },
    text: {
      color: colors.text,
      fontSize: FontSizes.md,
      fontFamily: FontFamily.default,
    },
  });

  return <View style={styles.container}>...</View>;
}
```

## Safe Area Strategy

The app uses an edge-to-edge display strategy where content can scroll behind OS UI elements (status bar, home indicator) while keeping interactive UI within safe areas.

### Core Principles

1. **Content scrolls behind OS UI** - Article content, lists, and scrollable areas extend behind the status bar and tab bar/action menu when scrolling
2. **Initial render within safe areas** - Headers, titles, and initial content position below the status bar
3. **Floating UI accounts for safe areas** - Tab bar and bottom action menu add bottom inset to their positioning

### Implementation Pattern

**DO NOT use `SafeAreaView`** for wrapping entire screens. Instead:

1. Use plain `View` containers
2. Import `useSafeAreaInsets` from `react-native-safe-area-context`
3. Apply insets to specific content areas

```typescript
import { View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Layout, Spacing } from '../constants';

function MyScreen() {
  const insets = useSafeAreaInsets();

  // For tab screens: account for tab bar + bottom safe area
  const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;

  // For detail screens: account for action menu + bottom safe area
  const detailBottomPadding = Layout.bottomActionMenuHeight + insets.bottom + Spacing.md;

  return (
    <View style={styles.container}>
      <FlatList
        ListHeaderComponent={() => (
          <View style={{ paddingTop: insets.top + Spacing.md }}>
            <Text>Header renders below status bar</Text>
          </View>
        )}
        contentContainerStyle={{ paddingBottom: bottomPadding }}
        // Content can scroll behind status bar and tab bar
      />
    </View>
  );
}
```

### Layout Constants

Use the `Layout` constants from `src/constants/theme.ts`:

```typescript
Layout.tabBarHeight           // 70 - Height of floating tab bar (54px pill + 16px padding)
Layout.bottomActionMenuHeight // 70 - Height of floating action menu
Layout.headerHeight           // 64 - Standard header height
```

### Screen-Specific Guidelines

| Screen Type | Top Handling | Bottom Handling |
|-------------|--------------|-----------------|
| **Tab screens** (Explore, Read, You) | Header with `paddingTop: insets.top + Spacing.md` | `paddingBottom: Layout.tabBarHeight + insets.bottom + Spacing.md` |
| **Detail screens** (ArticleContent) | Content with `paddingTop: insets.top + Spacing.md` | `paddingBottom: Layout.bottomActionMenuHeight + insets.bottom + Spacing.md` |
| **Modal screens** (AddArticle, Login) | Content with `paddingTop: insets.top` | Content with `paddingBottom: insets.bottom` |
| **Floating UI** (TabBar, ActionMenu) | N/A | Position absolutely, add `insets.bottom + Spacing.sm` to paddingBottom |

### StatusBar Configuration

The app configures StatusBar in `App.tsx` for edge-to-edge display:

```typescript
<StatusBar
  style={colorScheme === 'dark' ? 'light' : 'dark'}
  translucent={Platform.OS === 'android'}
  backgroundColor="transparent"
/>
```

Android also has `androidStatusBar.translucent: true` in `app.json`.

### Common Mistakes to Avoid

1. **Don't wrap screens in SafeAreaView** - This prevents content from scrolling behind OS UI
2. **Don't use hardcoded padding values** (e.g., `paddingBottom: 100`) - Use dynamic values based on `useSafeAreaInsets()` and `Layout` constants
3. **Don't forget bottom padding on scrollable content** - Content will be hidden behind floating tab bar or action menu

## Development Workflow

### Initial Setup
```bash
cd apps/mobile

# Install dependencies
npm install

# Start development server
npm start
```

### Running on Devices

**iOS Simulator:**
```bash
npm run ios
# or press 'i' in Expo dev server
```

**Android Emulator:**
```bash
npm run android
# or press 'a' in Expo dev server
```

**Physical Device:**
1. Install Expo Go app from App Store or Google Play
2. Scan QR code from Expo dev server
3. App will load on your device

**Web:**
```bash
npm run web
# or press 'w' in Expo dev server
```

### Code Quality

**Type Checking:**
```bash
npm run type-check
# Runs: tsc --noEmit
```

**Linting:**
```bash
npm run lint
# Runs: eslint .
```

**Fix Linting Issues:**
```bash
npx eslint . --fix
```

### Development Tips

1. **Hot Reload**: Changes auto-reload in Expo. Shake device or press 'r' to reload manually.

2. **Debug Menu**: Shake device or press `Cmd+D` (iOS) / `Cmd+M` (Android) to open debug menu.

3. **Console Logs**: Use React Native Debugger or browser DevTools (press 'j' to open).

4. **AsyncStorage**: Use Expo DevTools or React Native Debugger to inspect storage.

5. **Network Requests**: Monitor in React Native Debugger Network tab.

## Testing

### Running Tests
```bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage
```

### Testing Patterns

**Component Testing:**
```typescript
import { render, fireEvent } from '@testing-library/react-native';
import MyComponent from './MyComponent';

test('renders correctly', () => {
  const { getByText } = render(<MyComponent />);
  expect(getByText('Hello')).toBeTruthy();
});
```

**Service Testing:**
```typescript
import { AuthService } from '../services/auth';

jest.mock('@react-native-async-storage/async-storage');

test('saves tokens correctly', async () => {
  await AuthService.saveTokens({
    accessToken: 'access',
    refreshToken: 'refresh',
  });

  const token = await AuthService.getAccessToken();
  expect(token).toBe('access');
});
```

## Adding New Features

### Adding a New Screen
1. Create screen component in `src/screens/`:
```typescript
// src/screens/NewScreen.tsx
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

export default function NewScreen() {
  return (
    <View style={styles.container}>
      <Text>New Screen</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
```

2. Add to navigation in `src/navigation/RootNavigator.tsx` or `TabNavigator.tsx`:
```typescript
import NewScreen from '../screens/NewScreen';

// In Stack.Navigator or Tab.Navigator:
<Stack.Screen name="NewScreen" component={NewScreen} />
```

3. Update navigation types in `src/types/navigation.ts`:
```typescript
export type RootStackParamList = {
  // ... existing routes
  NewScreen: { param1: string };  // Add params if needed
};
```

4. Export screen from `src/screens/index.ts`:
```typescript
export { default as NewScreen } from './NewScreen';
```

### Adding a New Component
1. Create component in `src/components/common/`:
```typescript
// src/components/common/NewComponent.tsx
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

interface NewComponentProps {
  title: string;
  onPress?: () => void;
}

export default function NewComponent({ title, onPress }: NewComponentProps) {
  return (
    <View style={styles.container}>
      <Text>{title}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
  },
});
```

2. Export from `src/components/common/index.ts`:
```typescript
export { default as NewComponent } from './NewComponent';
```

### Adding a New Service Method
1. Add method to service class:
```typescript
// src/services/myservice.ts
export class MyService {
  static async newMethod(param: string): Promise<Result> {
    try {
      const response = await fetch(`${API_URL}/endpoint`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ param }),
      });

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || 'Request failed');
      }

      return result.data;
    } catch (error) {
      console.error('Error in newMethod:', error);
      throw error;
    }
  }
}
```

2. Use in component:
```typescript
import { MyService } from '../services';

async function handleAction() {
  try {
    const result = await MyService.newMethod('param');
    // Handle success
  } catch (error) {
    // Handle error
  }
}
```

## Code Conventions

### TypeScript
- **Strict mode enabled** in `tsconfig.json`
- **Type all props** for components
- **Type all function parameters** and return values
- **Use interfaces** for object shapes
- **Use type aliases** for unions and simple types
- **Avoid `any`** - use `unknown` and type guards instead

### React/React Native
- **Functional components** with hooks (no class components)
- **Named exports** for components (except default for screens)
- **Destructure props** in function signature
- **Use `useCallback`** for functions passed to children
- **Use `useMemo`** for expensive computations
- **Use `useEffect`** for side effects only

### Naming Conventions
- **Components**: PascalCase (e.g., `ArticleCard.tsx`)
- **Services**: PascalCase classes (e.g., `AuthService`)
- **Types/Interfaces**: PascalCase (e.g., `Article`, `LoginResponse`)
- **Constants**: UPPER_SNAKE_CASE (e.g., `API_BASE_URL`)
- **Functions**: camelCase (e.g., `handlePress`)
- **Variables**: camelCase (e.g., `articleList`)

### File Organization
- **One component per file**
- **Co-locate types** with components (or use `src/types/`)
- **Group related files** in folders
- **Use index.ts** for exports
- **Keep files under 300 lines** (split if larger)

### Styling
- **Use StyleSheet.create()** for component styles
- **Define styles at bottom** of file
- **Use theme constants** (Colors, Spacing, FontSizes)
- **Follow platform conventions** (iOS vs Android)
- **Test dark mode** for all screens

### Error Handling
- **Try/catch for async operations**
- **Log errors** with `console.error()`
- **Show user-friendly messages** (no stack traces)
- **Handle network errors** gracefully
- **Validate user input** before submission

### Async Operations
- **Use async/await** (not .then() chains)
- **Handle loading states** (show spinners)
- **Handle error states** (show error messages)
- **Cleanup on unmount** (cancel pending requests)

## Environment Configuration

### API Configuration
API endpoints configured in `src/config/api.ts`:

```typescript
export const API_CONFIG = {
  USER_SERVICE_URL: 'https://cairn.seatrain.net',
  RECOMMENDER_SERVICE_URL: 'https://cairn.seatrain.net',
  READ_SERVICE_URL: 'https://cairn.seatrain.net',
  REQUEST_TIMEOUT: 30000,
};
```

**For Local Development:**
Update `src/config/api.ts` to point to local services:
```typescript
export const API_CONFIG = {
  USER_SERVICE_URL: 'http://localhost:8082',
  RECOMMENDER_SERVICE_URL: 'http://localhost:8081',
  READ_SERVICE_URL: 'http://localhost:8083',
  REQUEST_TIMEOUT: 30000,
};
```

**iOS Simulator Note:**
- Use `http://localhost:PORT` for local services

**Android Emulator Note:**
- Use `http://10.0.2.2:PORT` instead of `localhost`
- Or use your machine's IP address (e.g., `http://192.168.1.100:PORT`)

### Expo Configuration
App configuration in `app.json`:

```json
{
  "expo": {
    "name": "ReadItLater",
    "slug": "readitlater",
    "version": "1.0.0",
    "orientation": "portrait",
    "icon": "./assets/icon.png",
    "userInterfaceStyle": "automatic",
    "ios": {
      "supportsTablet": true,
      "bundleIdentifier": "com.readitlater.app"
    },
    "android": {
      "package": "com.readitlater.app"
    }
  }
}
```

## Backend Integration

### Authentication Flow
1. **Initial Launch** → Device ID login (automatic)
2. **Device ID Login** → `POST /api/v1/auth/login/mobile` or `POST /api/v1/auth/register/mobile`
3. **Store Tokens** → AsyncStorage (`@cairn:access_token`, `@cairn:refresh_token`)
4. **All API Calls** → Include `Authorization: Bearer <token>` header
5. **401 Error** → Refresh token → Retry request
6. **Refresh Fails** → Logout → Show login screen

### API Request Pattern
All authenticated requests follow this pattern:

```typescript
private static async fetchWithAuth(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const accessToken = await AuthService.getAccessToken();

  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${accessToken}`,
      ...options.headers,
    },
  });

  // Handle 401 - token expired
  if (response.status === 401) {
    await AuthService.refreshAccessToken();
    // Retry request with new token...
  }

  return response;
}
```

### Data Transformation
Backend models are transformed to mobile `Article` interface:

**Explore Service:**
```typescript
BackendArticle → Article
- id (same)
- link → url
- title (same)
- description || content → description
- author || feed_title → author
- published → publishedDate
- categories → tags
- Extract image from content → imageUrl
```

**Read Service:**
```typescript
UserContentResponse → Article
- content.id → id
- content.original_url → url
- content.title → title
- content.description || excerpt → description
- content.lead_image_url → imageUrl
- content.author || site_name → author
- content.published_at → publishedDate
- content.word_count / 200 → readingTime
- status === 'completed' → isRead
- is_favorite → isFavorite
```

## Troubleshooting

### Common Issues

**"No bundle URL present" error:**
- Restart Expo dev server (`npm start`)
- Clear cache: `npx expo start -c`

**"Cannot connect to Metro" error:**
- Check that dev server is running
- Ensure device/simulator is on same network
- Try restarting dev server

**"Unable to resolve module" error:**
- Run `npm install` to ensure all dependencies installed
- Clear node_modules and reinstall: `rm -rf node_modules && npm install`
- Clear Metro cache: `npx expo start -c`

**AsyncStorage errors:**
- Check if AsyncStorage is properly imported from `@react-native-async-storage/async-storage`
- Ensure async functions are awaited

**Authentication errors:**
- Check API endpoint configuration in `src/config/api.ts`
- Verify backend services are running
- Check device ID is being generated correctly
- Clear stored tokens: AsyncStorage → Delete `@cairn:*` keys

**Image not loading:**
- Check image URL is valid
- Verify network permissions
- Test with a known working image URL

**TypeScript errors:**
- Run `npm run type-check` to see all type errors
- Check that types are imported correctly
- Verify interface definitions match usage

**Navigation errors:**
- Check navigation types in `src/types/navigation.ts`
- Verify screen is registered in navigator
- Ensure params match type definition

### Debug Tools

**React Native Debugger:**
```bash
# Install
brew install --cask react-native-debugger

# Run
open "rndebugger://set-debugger-loc?host=localhost&port=8081"
```

**Expo DevTools:**
- Press `d` in Expo dev server terminal
- Opens browser with DevTools interface

**Console Logs:**
- Press `j` in Expo dev server to open console
- Use `console.log()`, `console.warn()`, `console.error()`

**Network Monitoring:**
- Use React Native Debugger Network tab
- Or Flipper for advanced debugging

## Related Documentation

- **Main README**: `/README.md` - Project overview
- **Root CLAUDE.md**: `/CLAUDE.md` - Project-wide guidance
- **Engineering Principles**: `/docs/ENGINEERING_PRINCIPLES.md` - Coding standards and architecture
- **User Service**: `/services/users/CLAUDE.md` - Authentication backend
- **Explore Service**: `/services/explore/CLAUDE.md` - Recommendations backend
- **Read Service**: `/services/read/CLAUDE.md` - Content storage backend
- **Expo Documentation**: https://docs.expo.dev/
- **React Navigation**: https://reactnavigation.org/docs/getting-started
- **React Native**: https://reactnative.dev/docs/getting-started
