import React, { useState, useEffect, useRef } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  useColorScheme,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../constants';
import { HeaderPopover } from './common/HeaderPopover';
import { ReadService } from '../services/read';
import { DetectURLResponse } from '@cairn/shared';

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
  const [discovering, setDiscovering] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detectionResult, setDetectionResult] = useState<DetectURLResponse | null>(null);

  // Handle to the in-flight detection so a dismiss can abandon it (detection is
  // not user-initiated — it fires on every URL change — so it must never block
  // closing the modal).
  const detectRunRef = useRef<{ cancelled: boolean } | null>(null);

  // Handle to the in-flight Find-feed request so a dismiss can abandon it —
  // same rationale as detectRunRef: a hung network request must never trap
  // the user, and a late response must not setState or pop an Alert.
  const discoverRunRef = useRef<{ cancelled: boolean } | null>(null);

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
    const run = { cancelled: false };
    detectRunRef.current = run;

    const detectURL = async () => {
      setDetecting(true);
      try {
        const result = await ReadService.detectURL(normalizedUrl);
        if (!run.cancelled) {
          setDetectionResult(result);
        }
      } catch (err) {
        console.error('URL detection error:', err);
        // Silently fail - user can still submit
        if (!run.cancelled) {
          setDetectionResult({
            url: normalizedUrl,
            type: 'unknown',
            title: null,
          });
        }
      } finally {
        if (!run.cancelled) {
          setDetecting(false);
        }
      }
    };

    detectURL();

    return () => {
      run.cancelled = true;
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
    setDiscovering(true);
    const run = { cancelled: false };
    discoverRunRef.current = run;

    try {
      const normalizedUrl = normalizeUrl(url);
      const { feeds } = await ReadService.discoverFeed(normalizedUrl);

      if (run.cancelled) return;

      if (feeds.length === 0) {
        setError('No RSS feed found for this site');
        return;
      }

      if (feeds.length === 1) {
        // Single feed: update URL input; useEffect will re-detect as feed
        // and the UI will merge the buttons into "Add Feed".
        setUrl(feeds[0].url);
        return;
      }

      // Multiple feeds: let the user pick one via Alert.
      Alert.alert(
        'Multiple feeds found',
        'Select a feed to use:',
        [
          ...feeds.map((feed) => ({
            text: feed.title || feed.url,
            onPress: () => setUrl(feed.url),
          })),
          { text: 'Cancel', style: 'cancel' as const },
        ],
        { cancelable: true }
      );
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      if (!run.cancelled) setError(errorMessage);
    } finally {
      if (!run.cancelled) setDiscovering(false);
    }
  };

  const handleClose = () => {
    // Only a real submit in flight blocks dismissal. `detecting` / `discovering`
    // are advisory background work and must not trap the user.
    if (loading) return;

    // Abandon any in-flight detection so a late response can't touch state.
    if (detectRunRef.current) detectRunRef.current.cancelled = true;
    // Same for an in-flight Find-feed request.
    if (discoverRunRef.current) discoverRunRef.current.cancelled = true;
    setDetecting(false);
    setDiscovering(false);
    setUrl('');
    setError(null);
    setDetectionResult(null);
    onClose();
  };

  // Button text changes based on detection
  const getSubmitButtonText = () => {
    if (loading) return 'Adding...';
    if (detecting) return 'Detecting...';
    if (detectionResult?.type === 'feed') return 'Add Feed';
    return 'Add';
  };

  return (
    <HeaderPopover visible={visible} onClose={handleClose}>
          <View>
            <View style={styles.inputContainer}>
              <TextInput
                style={[
                  styles.input,
                  {
                    backgroundColor: colors.background,
                    color: colors.text,
                    borderColor: error ? colors.error : 'transparent',
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

              {detectionResult?.type !== 'feed' && (
                <View style={styles.buttonWrapper}>
                  <TouchableOpacity
                    style={[
                      styles.secondaryButton,
                      {
                        backgroundColor: colors.background,
                        borderColor: colors.textSecondary,
                        opacity: loading || discovering || !url.trim() ? 0.5 : 1,
                      }
                    ]}
                    onPress={handleFindFeedPress}
                    disabled={loading || discovering || !url.trim()}
                    activeOpacity={0.7}
                  >
                    {discovering ? (
                      <ActivityIndicator color={colors.text} size="small" />
                    ) : (
                      <Text style={[styles.secondaryButtonText, { color: colors.text }]}>
                        Find feed
                      </Text>
                    )}
                  </TouchableOpacity>
                </View>
              )}
            </View>
          </View>
    </HeaderPopover>
  );
};

const styles = StyleSheet.create({
  inputContainer: {
    width: '100%',
  },
  input: {
    height: 48,
    borderRadius: BorderRadius.md,
    borderWidth: 0,
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
    gap: 12,
    width: '100%',
  },
  buttonWrapper: {
    flex: 1,
  },
  primaryButton: {
    height: 48,
    borderRadius: BorderRadius.lg,
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
    borderRadius: BorderRadius.lg,
    borderWidth: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  secondaryButtonText: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultSemiBold,
  },
});
