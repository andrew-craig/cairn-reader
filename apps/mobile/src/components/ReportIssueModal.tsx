import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  useColorScheme,
  useWindowDimensions,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../constants';
import { HeaderPopover } from './common/HeaderPopover';
import { ReadService, ExtractionFeedbackCategory } from '../services/read';

const COMMENT_MAX_LENGTH = 2000;

const REASONS: { value: ExtractionFeedbackCategory; label: string }[] = [
  { value: 'body_missing_content', label: 'Part of the article is missing' },
  { value: 'body_extra_content', label: "Contains things that aren't part of the article (ads, menus, related links)" },
  { value: 'images', label: 'Images are missing or broken' },
  { value: 'formatting', label: 'Formatting is wrong (lists, tables, code, layout)' },
  { value: 'links', label: 'Links are missing or broken' },
  { value: 'title', label: 'Title is wrong' },
  { value: 'author', label: 'Author is wrong' },
  { value: 'description', label: 'Summary is wrong' },
  { value: 'lead_image', label: 'Header image is wrong' },
  { value: 'other', label: 'Something else' },
];

interface ReportIssueModalProps {
  visible: boolean;
  articleId: string;
  onClose: () => void;
}

export const ReportIssueModal: React.FC<ReportIssueModalProps> = ({
  visible,
  articleId,
  onClose,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const { height } = useWindowDimensions();

  const [selected, setSelected] = useState<ExtractionFeedbackCategory[]>([]);
  const [comment, setComment] = useState('');
  const [loadingExisting, setLoadingExisting] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Prefill from the user's existing report each time the dialog opens.
  useEffect(() => {
    if (!visible) return;
    let cancelled = false;
    setLoadingExisting(true);
    ReadService.getExtractionFeedback(articleId)
      .then((existing) => {
        if (cancelled || !existing) return;
        setSelected(existing.categories);
        setComment(existing.comment ?? '');
      })
      .catch((err) => console.error('Failed to load existing report:', err))
      .finally(() => {
        if (!cancelled) setLoadingExisting(false);
      });
    return () => {
      cancelled = true;
    };
  }, [visible, articleId]);

  const toggle = (value: ExtractionFeedbackCategory) => {
    setSelected((current) =>
      current.includes(value) ? current.filter((v) => v !== value) : [...current, value],
    );
    if (error) setError(null);
  };

  const handleClose = () => {
    if (sending) return;
    setSelected([]);
    setComment('');
    setError(null);
    onClose();
  };

  const handleSubmit = async () => {
    setError(null);
    setSending(true);
    try {
      // Keep the server's category order regardless of tap order.
      const categories = REASONS.map((r) => r.value).filter((v) => selected.includes(v));
      const trimmed = comment.trim();
      await ReadService.submitExtractionFeedback(articleId, {
        categories,
        ...(trimmed && { comment: trimmed }),
      });
      Alert.alert('Report sent', 'Thanks — your report was sent.', [{ text: 'OK' }]);
      setSending(false);
      handleClose();
    } catch (err) {
      console.error('Failed to send report:', err);
      setError(err instanceof Error ? err.message : 'Failed to send report');
      setSending(false);
    }
  };

  const submitDisabled = selected.length === 0 || sending || loadingExisting;

  return (
    <HeaderPopover visible={visible} onClose={handleClose}>
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Report an issue</Text>
        {loadingExisting && <ActivityIndicator color={colors.textSecondary} size="small" />}
      </View>

      <ScrollView
        style={{ maxHeight: height * 0.5 }}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        {REASONS.map(({ value, label }) => {
          const checked = selected.includes(value);
          return (
            <TouchableOpacity
              key={value}
              style={styles.row}
              onPress={() => toggle(value)}
              disabled={sending || loadingExisting}
              activeOpacity={0.7}
              accessibilityRole="checkbox"
              accessibilityState={{ checked, disabled: sending || loadingExisting }}
              accessibilityLabel={label}
            >
              <View
                style={[
                  styles.checkbox,
                  {
                    borderColor: checked ? colors.primary : colors.textSecondary,
                    backgroundColor: checked ? colors.primary : 'transparent',
                  },
                ]}
              >
                {checked && <Text style={[styles.checkmark, { color: colors.background }]}>✓</Text>}
              </View>
              <Text style={[styles.rowLabel, { color: colors.text }]}>{label}</Text>
            </TouchableOpacity>
          );
        })}

        <TextInput
          style={[
            styles.input,
            { backgroundColor: colors.background, color: colors.text },
          ]}
          placeholder="Details (optional)"
          placeholderTextColor={colors.textSecondary}
          value={comment}
          onChangeText={setComment}
          maxLength={COMMENT_MAX_LENGTH}
          multiline
          editable={!sending}
          accessibilityLabel="Details (optional)"
        />
      </ScrollView>

      {error && (
        <Text style={[styles.errorText, { color: colors.error }]}>{error}</Text>
      )}

      <View style={styles.buttonContainer}>
        <View style={styles.buttonWrapper}>
          <TouchableOpacity
            style={[
              styles.secondaryButton,
              {
                backgroundColor: colors.background,
                borderColor: colors.textSecondary,
                opacity: sending ? 0.5 : 1,
              },
            ]}
            onPress={handleClose}
            disabled={sending}
            activeOpacity={0.7}
            accessibilityRole="button"
          >
            <Text style={[styles.secondaryButtonText, { color: colors.text }]}>Cancel</Text>
          </TouchableOpacity>
        </View>
        <View style={styles.buttonWrapper}>
          <TouchableOpacity
            style={[
              styles.primaryButton,
              { backgroundColor: colors.primary, opacity: submitDisabled ? 0.5 : 1 },
            ]}
            onPress={handleSubmit}
            disabled={submitDisabled}
            activeOpacity={0.7}
            accessibilityRole="button"
            accessibilityState={{ disabled: submitDisabled }}
          >
            {sending ? (
              <ActivityIndicator color={colors.background} size="small" />
            ) : (
              <Text style={[styles.primaryButtonText, { color: colors.background }]}>Submit</Text>
            )}
          </TouchableOpacity>
        </View>
      </View>
    </HeaderPopover>
  );
};

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  title: {
    fontSize: FontSizes.lg,
    fontFamily: FontFamily.defaultSemiBold,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm + Spacing.xs,
    paddingVertical: Spacing.sm,
  },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: BorderRadius.sm,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkmark: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.defaultBold,
  },
  rowLabel: {
    flex: 1,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  input: {
    minHeight: 96,
    marginTop: Spacing.sm,
    borderRadius: BorderRadius.md,
    padding: Spacing.md,
    paddingTop: Spacing.md,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    textAlignVertical: 'top',
  },
  errorText: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
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
