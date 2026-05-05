import React, { useState } from 'react';
import { Text, StyleSheet, useColorScheme } from 'react-native';
import { IconButton } from '../components/common/IconButton';
import { HeaderPopover } from '../components/common/HeaderPopover';
import { SubscriptionListScreen } from '../components/SubscriptionListScreen';
import { Colors, FontSizes, FontFamily } from '../constants/theme';
import { UnifiedSubscription } from '../types/read';

const newslettersFilter = (s: UnifiedSubscription) => s.type === 'email';

const getNewslettersSubtitle = (s: UnifiedSubscription): string | undefined =>
  s.email_data?.email_address ?? s.description;

const AddNewsletterModal: React.FC<{ visible: boolean; onClose: () => void; emailAddress: string }> = ({
  visible,
  onClose,
  emailAddress,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  return (
    <HeaderPopover visible={visible} onClose={onClose}>
      <Text style={[styles.modalLabel, { color: colors.text }]}>
        Send or forward emails to
      </Text>
      <Text style={[styles.modalEmail, { color: colors.text }]}>
        {emailAddress}
      </Text>
      <Text style={[styles.modalDescription, { color: colors.textSecondary }]}>
        New subscriptions will be automatically added to your reading list.
      </Text>
    </HeaderPopover>
  );
};

export const NewslettersScreen: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false);

  const headerActions = (
    <>
      <IconButton icon="add" onPress={() => setModalVisible(true)} size={24} />
      <IconButton icon="search-outline" onPress={() => {}} size={24} />
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
        emailAddress="rose-parrot-74@cairnreader.app"
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
  modalEmail: {
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
