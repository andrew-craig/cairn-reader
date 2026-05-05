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

const LOGIN_FONT_SIZE_TITLE = 56;
const LOGIN_FONT_SIZE_SUBTITLE = 26;
const LOGIN_GAP_SECTIONS = 64;
const LOGIN_GAP_BUTTONS = 12;
const LOGIN_MAX_WIDTH_HEADER = 240;
const LOGIN_MAX_WIDTH_BUTTONS = 280;
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
    gap: LOGIN_GAP_SECTIONS,
  },
  header: {
    alignItems: 'center',
    gap: Spacing.sm,
    maxWidth: LOGIN_MAX_WIDTH_HEADER,
    width: '100%',
  },
  title: {
    fontSize: LOGIN_FONT_SIZE_TITLE,
    fontFamily: FontFamily.heading,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: LOGIN_FONT_SIZE_SUBTITLE,
    fontFamily: FontFamily.heading,
    textAlign: 'center',
    lineHeight: 32,
  },
  buttonContainer: {
    gap: LOGIN_GAP_BUTTONS,
    width: '100%',
    maxWidth: LOGIN_MAX_WIDTH_BUTTONS,
  },
  emailForm: {
    gap: Spacing.md,
    width: '100%',
    maxWidth: LOGIN_MAX_WIDTH_BUTTONS,
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
