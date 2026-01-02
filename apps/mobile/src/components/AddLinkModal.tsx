import React, { useState } from 'react';
import {
  Modal,
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  useColorScheme,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../constants';
import { Button } from './common/Button';

interface AddLinkModalProps {
  visible: boolean;
  onClose: () => void;
  onAddArticle: (url: string) => Promise<void>;
  onFindFeed: (url: string) => Promise<void>;
}

export const AddLinkModal: React.FC<AddLinkModalProps> = ({
  visible,
  onClose,
  onAddArticle,
  onFindFeed,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isValidUrl = (urlString: string): boolean => {
    try {
      const trimmedUrl = urlString.trim();
      if (!trimmedUrl) return false;

      // Add https:// if no protocol is specified
      const urlToTest = trimmedUrl.match(/^https?:\/\//)
        ? trimmedUrl
        : `https://${trimmedUrl}`;

      new URL(urlToTest);
      return true;
    } catch {
      return false;
    }
  };

  const normalizeUrl = (urlString: string): string => {
    const trimmedUrl = urlString.trim();
    return trimmedUrl.match(/^https?:\/\//)
      ? trimmedUrl
      : `https://${trimmedUrl}`;
  };

  const handleAddPress = async () => {
    if (!isValidUrl(url)) {
      setError('Please enter a valid URL');
      return;
    }

    setError(null);
    setLoading(true);

    try {
      const normalizedUrl = normalizeUrl(url);
      await onAddArticle(normalizedUrl);
      setUrl('');
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add article');
    } finally {
      setLoading(false);
    }
  };

  const handleFindFeedPress = async () => {
    if (!isValidUrl(url)) {
      setError('Please enter a valid URL');
      return;
    }

    setError(null);
    setLoading(true);

    try {
      const normalizedUrl = normalizeUrl(url);
      await onFindFeed(normalizedUrl);
      setUrl('');
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to find feed');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading) {
      setUrl('');
      setError(null);
      onClose();
    }
  };

  return (
    <Modal
      visible={visible}
      transparent
      animationType="slide"
      onRequestClose={handleClose}
    >
      <KeyboardAvoidingView
        style={styles.overlay}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <TouchableOpacity
          style={styles.overlay}
          activeOpacity={1}
          onPress={handleClose}
        >
          <TouchableOpacity
            activeOpacity={1}
            style={[
              styles.modalContent,
              { backgroundColor: colors.background }
            ]}
            onPress={(e) => e.stopPropagation()}
          >
            <View style={styles.header}>
              <Text style={[styles.title, { color: colors.text }]}>
                Add link
              </Text>
              <TouchableOpacity
                onPress={handleClose}
                disabled={loading}
                style={styles.closeButton}
              >
                <Text style={[styles.closeButtonText, { color: colors.textSecondary }]}>
                  ✕
                </Text>
              </TouchableOpacity>
            </View>

            <View style={styles.inputContainer}>
              <TextInput
                style={[
                  styles.input,
                  {
                    backgroundColor: colors.card,
                    color: colors.text,
                    borderColor: error ? colors.error : colors.border,
                  }
                ]}
                placeholder="Add link"
                placeholderTextColor={colors.textSecondary}
                value={url}
                onChangeText={(text) => {
                  setUrl(text);
                  if (error) setError(null);
                }}
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="url"
                editable={!loading}
                returnKeyType="done"
                onSubmitEditing={handleAddPress}
              />
              {error && (
                <Text style={[styles.errorText, { color: colors.error }]}>
                  {error}
                </Text>
              )}
            </View>

            <View style={styles.buttonContainer}>
              <View style={styles.buttonWrapper}>
                <TouchableOpacity
                  style={[
                    styles.primaryButton,
                    {
                      backgroundColor: colors.primary,
                      opacity: loading || !url.trim() ? 0.5 : 1,
                    }
                  ]}
                  onPress={handleAddPress}
                  disabled={loading || !url.trim()}
                  activeOpacity={0.7}
                >
                  {loading ? (
                    <ActivityIndicator color="#FFFFFF" size="small" />
                  ) : (
                    <Text style={styles.primaryButtonText}>Add</Text>
                  )}
                </TouchableOpacity>
              </View>

              <View style={styles.buttonWrapper}>
                <TouchableOpacity
                  style={[
                    styles.secondaryButton,
                    {
                      backgroundColor: colors.card,
                      borderColor: colors.border,
                      opacity: loading || !url.trim() ? 0.5 : 1,
                    }
                  ]}
                  onPress={handleFindFeedPress}
                  disabled={loading || !url.trim()}
                  activeOpacity={0.7}
                >
                  <Text style={[styles.secondaryButtonText, { color: colors.text }]}>
                    Find feed
                  </Text>
                </TouchableOpacity>
              </View>
            </View>
          </TouchableOpacity>
        </TouchableOpacity>
      </KeyboardAvoidingView>
    </Modal>
  );
};

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    justifyContent: 'flex-end',
  },
  modalContent: {
    borderTopLeftRadius: BorderRadius.xl,
    borderTopRightRadius: BorderRadius.xl,
    paddingTop: Spacing.lg,
    paddingBottom: Platform.OS === 'ios' ? Spacing.xxl : Spacing.lg,
    paddingHorizontal: Spacing.md,
    minHeight: 200,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: Spacing.lg,
  },
  title: {
    fontSize: FontSizes.xl,
    fontFamily: FontFamily.headingSemiBold,
  },
  closeButton: {
    padding: Spacing.sm,
  },
  closeButtonText: {
    fontSize: FontSizes.xl,
    fontFamily: FontFamily.default,
  },
  inputContainer: {
    marginBottom: Spacing.lg,
  },
  input: {
    height: 48,
    borderRadius: BorderRadius.md,
    borderWidth: 1,
    paddingHorizontal: Spacing.md,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  errorText: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    marginTop: Spacing.sm,
    marginLeft: Spacing.xs,
  },
  buttonContainer: {
    flexDirection: 'row',
    gap: Spacing.md,
  },
  buttonWrapper: {
    flex: 1,
  },
  primaryButton: {
    height: 48,
    borderRadius: BorderRadius.md,
    justifyContent: 'center',
    alignItems: 'center',
  },
  primaryButtonText: {
    color: '#FFFFFF',
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultSemiBold,
  },
  secondaryButton: {
    height: 48,
    borderRadius: BorderRadius.md,
    borderWidth: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  secondaryButtonText: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultSemiBold,
  },
});
