import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, useColorScheme } from 'react-native';
import * as Clipboard from 'expo-clipboard';
import { IconButton } from '../components/common/IconButton';
import { HeaderPopover } from '../components/common/HeaderPopover';
import { SubscriptionListScreen } from '../components/SubscriptionListScreen';
import { Colors, FontSizes, FontFamily, Spacing } from '../constants/theme';
import { UnifiedSubscription } from '@cairn/shared';
import { ReadService } from '../services/read';

const newslettersFilter = (s: UnifiedSubscription) => s.type === 'email';

const getNewslettersSubtitle = (s: UnifiedSubscription): string | undefined =>
  s.email_data?.email_address ?? s.description;

const AddNewsletterModal: React.FC<{
  visible: boolean;
  onClose: () => void;
  emailAddress: string | null;
  error: string | null;
}> = ({ visible, onClose, emailAddress, error }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);

  const handleCopy = async () => {
    if (!emailAddress) return;
    try {
      await Clipboard.setStringAsync(emailAddress);
      setCopied(true);
    } catch (err) {
      console.error('Failed to copy newsletter address:', err);
    }
  };

  return (
    <HeaderPopover visible={visible} onClose={onClose}>
      <Text style={[styles.modalLabel, { color: colors.text }]}>
        Send or forward emails to
      </Text>
      <View style={styles.emailRow}>
        <Text style={[styles.modalEmail, { color: colors.text }]} selectable>
          {error ?? emailAddress ?? 'Loading…'}
        </Text>
        {!error && emailAddress && (
          <IconButton
            icon={copied ? 'checkmark' : 'copy-outline'}
            onPress={handleCopy}
            size={20}
            accessibilityLabel={copied ? 'Copied' : 'Copy address'}
          />
        )}
      </View>
      <Text style={[styles.modalDescription, { color: colors.textSecondary }]}>
        New subscriptions will be automatically added to your reading list.
      </Text>
    </HeaderPopover>
  );
};

export const NewslettersScreen: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false);
  const [emailAddress, setEmailAddress] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    ReadService.getOrCreateEmailAddress()
      .then((address) => {
        if (!cancelled) setEmailAddress(address);
      })
      .catch((err) => {
        if (!cancelled) {
          console.error('Failed to load newsletter address:', err);
          setError('Unable to load address');
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const headerActions = (
    <>
      <IconButton icon="add" onPress={() => setModalVisible(true)} size={24} accessibilityLabel="Add newsletter" />
      <IconButton icon="search-outline" onPress={() => {}} size={24} accessibilityLabel="Search" />
    </>
  );

  return (
    <>
      <SubscriptionListScreen
        title="Newsletters"
        filter={newslettersFilter}
        getSubtitle={getNewslettersSubtitle}
        headerActions={headerActions}
        emptyMessage="No newsletters yet"
      />
      <AddNewsletterModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        emailAddress={emailAddress}
        error={error}
      />
    </>
  );
};

const styles = StyleSheet.create({
  modalLabel: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    textAlign: 'center',
  },
  emailRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.xs,
  },
  modalEmail: {
    flexShrink: 1,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultBold,
    textAlign: 'center',
  },
  modalDescription: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    textAlign: 'center',
  },
});
