import React, { useState, useEffect } from 'react';
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
  Alert,
} from 'react-native';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../constants';
import { Button } from './common/Button';
import { ReadService } from '../services/read';
import { DetectURLResponse } from '../types/read';

interface AddLinkModalProps {
  visible: boolean;
  onClose: () => void;
  onSuccess?: () => void; // Optional callback after successful add
}

export const AddLinkModal: React.FC<AddLinkModalProps> = ({
  visible,
  onClose,
  onSuccess,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detectionResult, setDetectionResult] = useState<DetectURLResponse | null>(null);

  // Trigger URL detection when URL changes
  useEffect(() => {
    if (!url.trim()) {
      setDetectionResult(null);
      return;
    }

    // Only detect if URL is valid
    if (!isValidUrl(url)) {
      setDetectionResult(null);
      return;
    }

    const normalizedUrl = normalizeUrl(url);
    let cancelled = false;

    const detectURL = async () => {
      setDetecting(true);
      try {
        const result = await ReadService.detectURL(normalizedUrl);
        if (!cancelled) {
          setDetectionResult(result);
        }
      } catch (err) {
        console.error('URL detection error:', err);
        // Silently fail - user can still submit
        if (!cancelled) {
          setDetectionResult({
            url: normalizedUrl,
            type: 'unknown',
            title: null,
          });
        }
      } finally {
        if (!cancelled) {
          setDetecting(false);
        }
      }
    };

    detectURL();

    return () => {
      cancelled = true;
    };
  }, [url]);

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
      const response = await ReadService.addURL({
        url: normalizedUrl,
        type: detectionResult?.type,
        title: detectionResult?.title ?? undefined,
      });

      // Show success message based on response type
      if (response.type === 'feed') {
        Alert.alert(
          'Success',
          `Subscribed to ${response.subscription.title}`,
          [{ text: 'OK' }]
        );
      } else {
        Alert.alert(
          'Success',
          'Article added to reading list',
          [{ text: 'OK' }]
        );
      }

      setUrl('');
      setDetectionResult(null);
      onClose();

      // Call success callback if provided
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';

      // Handle specific error messages
      if (errorMessage.includes('already subscribed')) {
        setError('Already subscribed to this feed');
      } else if (errorMessage.includes('Failed to subscribe')) {
        setError('Failed to subscribe to feed');
      } else if (errorMessage.includes('Failed to add')) {
        setError('Failed to add article');
      } else {
        setError(errorMessage);
      }
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

      // Force feed type for "Find feed" button
      const response = await ReadService.addURL({
        url: normalizedUrl,
        type: 'feed', // Always try to add as feed
        title: detectionResult?.title ?? undefined,
      });

      if (response.type === 'feed') {
        Alert.alert(
          'Success',
          `Subscribed to ${response.subscription.title}`,
          [{ text: 'OK' }]
        );
      } else {
        Alert.alert(
          'Info',
          'This URL is not an RSS feed. Added as an article instead.',
          [{ text: 'OK' }]
        );
      }

      setUrl('');
      setDetectionResult(null);
      onClose();

      // Call success callback if provided
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';

      if (errorMessage.includes('already subscribed')) {
        setError('Already subscribed to this feed');
      } else if (errorMessage.includes('invalid_feed')) {
        setError('This URL is not a valid RSS/Atom feed');
      } else {
        setError(errorMessage);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading && !detecting) {
      setUrl('');
      setError(null);
      setDetectionResult(null);
      onClose();
    }
  };

  // Button text changes based on detection
  const getSubmitButtonText = () => {
    if (loading) return 'Adding...';
    if (detecting) return 'Detecting...';
    if (detectionResult?.type === 'feed') return 'Add Feed';
    return 'Add';
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
                  {loading || detecting ? (
                    <ActivityIndicator color="#FFFFFF" size="small" />
                  ) : (
                    <Text style={styles.primaryButtonText}>{getSubmitButtonText()}</Text>
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
