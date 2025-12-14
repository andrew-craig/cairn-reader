# ReadItLater

A modern read-it-later mobile application built with React Native and Expo, targeting both iOS and Android platforms.

## Features

- 📚 Save articles for later reading
- ⭐ Mark articles as favorites
- ✅ Track read/unread status
- 🗂️ Organize articles in archive
- 🔗 Open articles in browser
- 📤 Share articles
- 🌓 Dark mode support
- 💾 Local storage with AsyncStorage

## Tech Stack

- **Framework**: React Native with Expo
- **Navigation**: React Navigation (Stack & Bottom Tabs)
- **Language**: TypeScript
- **Storage**: AsyncStorage
- **UI**: React Native core components
- **Icons**: Expo Vector Icons (Ionicons)

## Project Structure

```
cairn/
├── src/
│   ├── components/        # Reusable UI components
│   │   └── common/        # Common components (ArticleCard, Button, etc.)
│   ├── screens/           # App screens
│   │   ├── HomeScreen.tsx
│   │   ├── AddArticleScreen.tsx
│   │   ├── ArticleDetailScreen.tsx
│   │   ├── FavoritesScreen.tsx
│   │   ├── ArchiveScreen.tsx
│   │   └── SettingsScreen.tsx
│   ├── navigation/        # Navigation configuration
│   │   ├── RootNavigator.tsx
│   │   └── TabNavigator.tsx
│   ├── services/          # Services (Storage, API, etc.)
│   │   └── storage.ts
│   ├── types/             # TypeScript type definitions
│   │   ├── article.ts
│   │   └── navigation.ts
│   ├── utils/             # Utility functions
│   │   └── helpers.ts
│   └── constants/         # App constants (theme, colors, etc.)
│       └── theme.ts
├── assets/                # Images, fonts, and other assets
├── App.tsx                # Entry point
├── app.json               # Expo configuration
├── package.json           # Dependencies
└── tsconfig.json          # TypeScript configuration
```

## Getting Started

### Prerequisites

- Node.js (v16 or later)
- npm or yarn
- Expo CLI (optional, but recommended)

### Installation

1. Install dependencies:

```bash
npm install
```

2. Start the development server:

```bash
npm start
```

3. Run on iOS simulator:

```bash
npm run ios
```

4. Run on Android emulator:

```bash
npm run android
```

5. Run on web browser:

```bash
npm run web
```

### Using Expo Go

1. Install Expo Go on your iOS or Android device
2. Run `npm start` to start the development server
3. Scan the QR code with:
   - iOS: Camera app
   - Android: Expo Go app

## App Screens

### Reading List (Home)
- Displays all unread articles
- Pull to refresh
- Tap to view article details
- Heart icon to toggle favorites

### Favorites
- Shows all favorited articles
- Same interactions as Reading List

### Archive
- Displays all read articles
- Helps keep your reading list clean

### Settings
- App information
- Clear all articles option

### Add Article
- Add new articles by URL
- Paste from clipboard
- Manual title entry

### Article Detail
- View article information
- Open in browser
- Mark as read/unread
- Toggle favorite status
- Share article
- Delete article

## Data Model

### Article Interface

```typescript
interface Article {
  id: string;
  url: string;
  title: string;
  description?: string;
  imageUrl?: string;
  author?: string;
  publishedDate?: string;
  readingTime?: number;
  tags: string[];
  isRead: boolean;
  isFavorite: boolean;
  addedAt: number;
  readAt?: number;
  notes?: string;
}
```

## Theme

The app supports both light and dark modes, automatically following the system preference. Color schemes are defined in `src/constants/theme.ts`.

## Future Enhancements

- [ ] Article metadata extraction from URLs
- [ ] Tags and categories
- [ ] Search functionality
- [ ] Export/import data
- [ ] Reading progress tracking
- [ ] Offline reading mode
- [ ] Cloud sync
- [ ] Multiple user accounts
- [ ] RSS feed integration

## Development

### Type Checking

```bash
npm run type-check
```

### Linting

```bash
npm run lint
```

## Building for Production

### iOS

```bash
expo build:ios
```

### Android

```bash
expo build:android
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
