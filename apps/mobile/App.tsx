import React from 'react';
import { Platform, useColorScheme } from 'react-native';
import { NavigationContainer } from '@react-navigation/native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { StatusBar } from 'expo-status-bar';
import { useFonts } from 'expo-font';
import {
  Inter_400Regular,
  Inter_500Medium,
  Inter_600SemiBold,
  Inter_700Bold,
} from '@expo-google-fonts/inter';
import {
  CrimsonPro_400Regular,
  CrimsonPro_500Medium,
  CrimsonPro_600SemiBold,
  CrimsonPro_700Bold,
} from '@expo-google-fonts/crimson-pro';
import { configureStorage, configureDefaultServerUrl } from '@cairn/shared';
import RootNavigator from './src/navigation/RootNavigator';
import { AuthProvider } from './src/contexts/AuthContext';
import { TopBlurGradient } from './src/components/common';
import { asyncStorageAdapter, DEFAULT_SERVER_URL } from './src/config/storage';

// Wire the shared API config layer to mobile's persistence + default backend
// before any service reads the server URL (decision_9b2d ADR).
configureStorage(asyncStorageAdapter);
configureDefaultServerUrl(() => DEFAULT_SERVER_URL);

export default function App() {
  const colorScheme = useColorScheme();
  const [fontsLoaded] = useFonts({
    Inter_400Regular,
    Inter_500Medium,
    Inter_600SemiBold,
    Inter_700Bold,
    CrimsonPro_400Regular,
    CrimsonPro_500Medium,
    CrimsonPro_600SemiBold,
    CrimsonPro_700Bold,
  });

  if (!fontsLoaded) {
    return null;
  }

  return (
    <SafeAreaProvider>
      <AuthProvider>
        <NavigationContainer>
          <RootNavigator />
          <TopBlurGradient />
          <StatusBar
            style={colorScheme === 'dark' ? 'light' : 'dark'}
            translucent={Platform.OS === 'android'}
            backgroundColor="transparent"
          />
        </NavigationContainer>
      </AuthProvider>
    </SafeAreaProvider>
  );
}
