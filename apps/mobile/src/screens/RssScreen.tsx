import React, { useState } from 'react';
import { SubscriptionListScreen } from '../components/SubscriptionListScreen';
import { AddLinkModal } from '../components/AddLinkModal';
import { IconButton } from '../components/common/IconButton';
import { UnifiedSubscription } from '@cairn/shared';

const rssFilter = (s: UnifiedSubscription) => s.type !== 'email';

const getRssSubtitle = (s: UnifiedSubscription): string | undefined =>
  s.rss_data?.feed_url ?? s.description;

export const RssScreen: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);

  return (
    <>
      <SubscriptionListScreen
        title="RSS"
        filter={rssFilter}
        getSubtitle={getRssSubtitle}
        headerActions={
          <IconButton icon="add" onPress={() => setModalVisible(true)} size={24} accessibilityLabel="Add feed" />
        }
        emptyMessage="No RSS feeds yet"
        reloadKey={reloadKey}
      />
      <AddLinkModal
        mode="feed"
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        onSuccess={() => setReloadKey((k) => k + 1)}
      />
    </>
  );
};
