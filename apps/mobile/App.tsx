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
import RootNavigator from './src/navigation/RootNavigator';
import { AuthProvider } from './src/contexts/AuthContext';
import { TopBlurGradient } from './src/components/common';

// The shared API config layer (storage + default server URL) is wired up in
// src/config/init.ts, imported first from index.js so it runs before any module
// that reads the server URL.

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
