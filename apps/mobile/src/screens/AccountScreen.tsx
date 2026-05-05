import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  StyleSheet,
  ScrollView,
  useColorScheme,
  KeyboardAvoidingView,
  Platform,
  Alert,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useAuth } from '../contexts/AuthContext';
import { AuthService } from '../services/auth';
import { Button } from '../components/common';
import { ScreenHeader } from '../components/common/ScreenHeader';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../constants/theme';
import { RootStackParamList } from '../types';

type AccountScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Account'>;

export const AccountScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const colors = isDark ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<AccountScreenNavigationProp>();
  const { user, login } = useAuth();

  const isAnonymous = !user?.email;

  // Save account form state
  const [showSaveForm, setShowSaveForm] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  // Change password form state
  const [showChangePassword, setShowChangePassword] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  const handleSaveAccount = async () => {
    if (!email.trim()) {
      Alert.alert('Error', 'Please enter an email address');
      return;
    }
    if (!password.trim()) {
      Alert.alert('Error', 'Please enter a password');
      return;
    }
    if (password.length < 8) {
      Alert.alert('Error', 'Password must be at least 8 characters');
      return;
    }

    setIsSaving(true);
    try {
      await AuthService.upgradeAccount(email.trim(), password);
      login();
      Alert.alert('Success', 'Your account has been saved');
      setShowSaveForm(false);
      setEmail('');
      setPassword('');
    } catch (error) {
      Alert.alert(
        'Error',
        error instanceof Error ? error.message : 'Failed to save account',
      );
    } finally {
      setIsSaving(false);
    }
  };

  const handleChangePassword = async () => {
    if (!currentPassword.trim()) {
      Alert.alert('Error', 'Please enter your current password');
      return;
    }
    if (!newPassword.trim()) {
      Alert.alert('Error', 'Please enter a new password');
      return;
    }
    if (newPassword.length < 8) {
      Alert.alert('Error', 'New password must be at least 8 characters');
      return;
    }
    if (newPassword !== confirmPassword) {
      Alert.alert('Error', 'Passwords do not match');
      return;
    }

    setIsChangingPassword(true);
    try {
      await AuthService.changePassword(currentPassword, newPassword);
      Alert.alert('Success', 'Your password has been changed');
      setShowChangePassword(false);
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (error) {
      Alert.alert(
        'Error',
        error instanceof Error ? error.message : 'Failed to change password',
      );
    } finally {
      setIsChangingPassword(false);
    }
  };

  const inputStyle = [
    styles.input,
    {
      backgroundColor: colors.card,
      color: colors.text,
      borderColor: colors.border,
    },
  ];

  return (
    <KeyboardAvoidingView
      style={[styles.container, { backgroundColor: colors.background }]}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={[styles.scrollContent, { paddingBottom: insets.bottom + Spacing.lg }]}
        keyboardShouldPersistTaps="handled"
      >
        <ScreenHeader title="Account" onBack={() => navigation.goBack()} />

        <View style={styles.content}>
          <Text style={[styles.accountLabel, { color: colors.textSecondary }]}>
            {isAnonymous ? 'Anonymous' : user?.email}
          </Text>

          {isAnonymous && !showSaveForm && (
            <View style={styles.actionSection}>
              <Button
                title="Save account"
                onPress={() => setShowSaveForm(true)}
              />
            </View>
          )}

          {isAnonymous && showSaveForm && (
            <View style={styles.formSection}>
              <TextInput
                style={inputStyle}
                placeholder="Email"
                placeholderTextColor={colors.textSecondary}
                value={email}
                onChangeText={setEmail}
                autoCapitalize="none"
                keyboardType="email-address"
                autoComplete="email"
                editable={!isSaving}
              />
              <TextInput
                style={inputStyle}
                placeholder="Password"
                placeholderTextColor={colors.textSecondary}
                value={password}
                onChangeText={setPassword}
                secureTextEntry
                autoCapitalize="none"
                autoComplete="password"
                editable={!isSaving}
              />
              <View style={styles.buttonGroup}>
                <Button
                  title="Save"
                  onPress={handleSaveAccount}
                  loading={isSaving}
                  disabled={isSaving}
                />
                <Button
                  title="Cancel"
                  onPress={() => {
                    setShowSaveForm(false);
                    setEmail('');
                    setPassword('');
                  }}
                  variant="secondary"
                  disabled={isSaving}
                />
              </View>
            </View>
          )}

          {!isAnonymous && !showChangePassword && (
            <View style={styles.actionSection}>
              <Button
                title="Change password"
                onPress={() => setShowChangePassword(true)}
                variant="secondary"
              />
            </View>
          )}

          {!isAnonymous && showChangePassword && (
            <View style={styles.formSection}>
              <TextInput
                style={inputStyle}
                placeholder="Current password"
                placeholderTextColor={colors.textSecondary}
                value={currentPassword}
                onChangeText={setCurrentPassword}
                secureTextEntry
                autoCapitalize="none"
                autoComplete="password"
                editable={!isChangingPassword}
              />
              <TextInput
                style={inputStyle}
                placeholder="New password"
                placeholderTextColor={colors.textSecondary}
                value={newPassword}
                onChangeText={setNewPassword}
                secureTextEntry
                autoCapitalize="none"
                autoComplete="new-password"
                editable={!isChangingPassword}
              />
              <TextInput
                style={inputStyle}
                placeholder="Confirm new password"
                placeholderTextColor={colors.textSecondary}
                value={confirmPassword}
                onChangeText={setConfirmPassword}
                secureTextEntry
                autoCapitalize="none"
                autoComplete="new-password"
                editable={!isChangingPassword}
              />
              <View style={styles.buttonGroup}>
                <Button
                  title="Change password"
                  onPress={handleChangePassword}
                  loading={isChangingPassword}
                  disabled={isChangingPassword}
                />
                <Button
                  title="Cancel"
                  onPress={() => {
                    setShowChangePassword(false);
                    setCurrentPassword('');
                    setNewPassword('');
                    setConfirmPassword('');
                  }}
                  variant="secondary"
                  disabled={isChangingPassword}
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
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
  },
  content: {
    paddingHorizontal: Spacing.md,
    paddingTop: Spacing.lg,
  },
  accountLabel: {
    fontSize: FontSizes.lg,
    fontFamily: FontFamily.defaultMedium,
  },
  actionSection: {
    marginTop: Spacing.lg,
  },
  formSection: {
    marginTop: Spacing.lg,
    gap: Spacing.md,
  },
  input: {
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.md,
    borderRadius: BorderRadius.md,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    borderWidth: 1,
    minHeight: 48,
  },
  buttonGroup: {
    gap: Spacing.md,
    marginTop: Spacing.sm,
  },
});
