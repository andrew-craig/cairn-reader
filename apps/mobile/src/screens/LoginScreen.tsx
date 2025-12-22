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
import { Button } from '../components/common';
import { Colors, Spacing, FontSizes, BorderRadius } from '../constants';
import { AuthService } from '../services';

interface LoginScreenProps {
  onLoginSuccess: () => void;
}

export const LoginScreen: React.FC<LoginScreenProps> = ({ onLoginSuccess }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

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
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.content}>
          <Text style={[styles.title, { color: colors.text }]}>Welcome to Cairn</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            Your personal read-it-later service
          </Text>

          {!showEmailLogin ? (
            <View style={styles.buttonContainer}>
              <Button
                title="Get Started"
                onPress={handleGetStarted}
                loading={isLoading}
                disabled={isLoading}
              />
              <View style={styles.divider}>
                <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
                <Text style={[styles.dividerText, { color: colors.textSecondary }]}>or</Text>
                <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
              </View>
              <Button
                title="Login with Email"
                onPress={() => setShowEmailLogin(true)}
                variant="secondary"
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

          <Text style={[styles.helpText, { color: colors.textSecondary }]}>
            {showEmailLogin
              ? 'Enter your email and password to access your account'
              : 'Get Started uses your device ID to create an account automatically'}
          </Text>
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
    paddingHorizontal: Spacing.lg,
    paddingVertical: Spacing.xxl,
  },
  title: {
    fontSize: FontSizes.xxl,
    fontWeight: 'bold',
    textAlign: 'center',
    marginBottom: Spacing.sm,
  },
  subtitle: {
    fontSize: FontSizes.md,
    textAlign: 'center',
    marginBottom: Spacing.xxl,
  },
  buttonContainer: {
    gap: Spacing.md,
    marginVertical: Spacing.lg,
  },
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    marginVertical: Spacing.md,
  },
  dividerLine: {
    flex: 1,
    height: 1,
  },
  dividerText: {
    marginHorizontal: Spacing.md,
    fontSize: FontSizes.sm,
  },
  emailForm: {
    gap: Spacing.md,
    marginTop: Spacing.lg,
  },
  input: {
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.md,
    borderRadius: BorderRadius.md,
    fontSize: FontSizes.md,
    borderWidth: 1,
    minHeight: 48,
  },
  helpText: {
    fontSize: FontSizes.sm,
    textAlign: 'center',
    marginTop: Spacing.lg,
  },
});
