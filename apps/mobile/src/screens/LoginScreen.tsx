import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  StyleSheet,
  useColorScheme,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  Alert,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Button } from '../components/common';
import { LogoMark } from '../components/LogoMark';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../constants';
import { AuthService } from '../services';

interface LoginScreenProps {
  onLoginSuccess: () => void;
}

export const LoginScreen: React.FC<LoginScreenProps> = ({ onLoginSuccess }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  const [showEmailLogin, setShowEmailLogin] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleGetStarted = async () => {
    setIsLoading(true);
    try {
      // Try to login first
      try {
        await AuthService.loginWithDevice();
        onLoginSuccess();
      } catch (loginError) {
        // If login fails, try to register
        await AuthService.registerWithDevice();
        onLoginSuccess();
      }
    } catch (error) {
      Alert.alert(
        'Error',
        error instanceof Error ? error.message : 'Failed to authenticate with device',
      );
    } finally {
      setIsLoading(false);
    }
  };

  const handleEmailLogin = async () => {
    if (!email.trim() || !password.trim()) {
      Alert.alert('Error', 'Please enter both email and password');
      return;
    }

    setIsLoading(true);
    try {
      await AuthService.loginWithEmail({ email: email.trim(), password });
      onLoginSuccess();
    } catch (error) {
      Alert.alert(
        'Login Failed',
        error instanceof Error ? error.message : 'Failed to login with email',
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView
      style={[styles.container, { backgroundColor: colors.background }]}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView
        contentContainerStyle={[
          styles.scrollContent,
          {
            paddingTop: insets.top,
            paddingBottom: insets.bottom,
          },
        ]}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.content}>
          <View style={styles.header}>
            <LogoMark size={90} />
            <Text style={[styles.title, { color: colors.text }]}>Cairn</Text>
            <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
              Read and discover{'\n'}what you love
            </Text>
          </View>

          {!showEmailLogin ? (
            <View style={styles.buttonContainer}>
              <Button
                title="Explore"
                onPress={handleGetStarted}
                loading={isLoading}
                disabled={isLoading}
              />
              <Button
                title="Login"
                onPress={() => setShowEmailLogin(true)}
                variant="outline"
                disabled={isLoading}
              />
            </View>
          ) : (
            <View style={styles.emailForm}>
              <TextInput
                style={[
                  styles.input,
                  {
                    backgroundColor: colors.card,
                    color: colors.text,
                    borderColor: colors.border,
                  },
                ]}
                placeholder="Email"
                placeholderTextColor={colors.textSecondary}
                value={email}
                onChangeText={setEmail}
                autoCapitalize="none"
                keyboardType="email-address"
                autoComplete="email"
                editable={!isLoading}
              />
              <TextInput
                style={[
                  styles.input,
                  {
                    backgroundColor: colors.card,
                    color: colors.text,
                    borderColor: colors.border,
                  },
                ]}
                placeholder="Password"
                placeholderTextColor={colors.textSecondary}
                value={password}
                onChangeText={setPassword}
                secureTextEntry
                autoCapitalize="none"
                autoComplete="password"
                editable={!isLoading}
              />
              <View style={styles.buttonContainer}>
                <Button
                  title="Login"
                  onPress={handleEmailLogin}
                  loading={isLoading}
                  disabled={isLoading}
                />
                <Button
                  title="Back"
                  onPress={() => setShowEmailLogin(false)}
                  variant="secondary"
                  disabled={isLoading}
                />
              </View>
            </View>
          )}

        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: 'center',
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: Spacing.xl,
    paddingVertical: Spacing.xxl,
    gap: 64,
  },
  header: {
    alignItems: 'center',
    gap: Spacing.sm,
    maxWidth: 240,
    width: '100%',
  },
  title: {
    fontSize: 56,
    fontFamily: FontFamily.heading,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 26,
    fontFamily: FontFamily.heading,
    textAlign: 'center',
    lineHeight: 32,
  },
  buttonContainer: {
    gap: 12,
    width: '100%',
    maxWidth: 280,
  },
  emailForm: {
    gap: Spacing.md,
    width: '100%',
    maxWidth: 280,
  },
  input: {
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.md,
    borderRadius: BorderRadius.lg,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    borderWidth: 1,
    minHeight: 48,
  },
  helpText: {
    fontSize: FontSizes.sm,
    textAlign: 'center',
    marginTop: Spacing.lg,
  },
});
