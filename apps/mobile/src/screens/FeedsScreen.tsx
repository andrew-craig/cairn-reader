import React, { useCallback } from 'react';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { IconButton } from '../components/common/IconButton';
import { SubscriptionListScreen } from '../components/SubscriptionListScreen';
import { RootStackParamList } from '../types';
import { UnifiedSubscription } from '../types/read';

type FeedsScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Feeds'>;

const feedsFilter = (s: UnifiedSubscription) => s.type !== 'email';

const getFeedsSubtitle = (s: UnifiedSubscription): string | undefined =>
  s.rss_data?.feed_url ?? s.description;

export const FeedsScreen: React.FC = () => {
  const navigation = useNavigation<FeedsScreenNavigationProp>();

  const headerActions = (
    <>
      <IconButton icon="add" onPress={() => {}} size={24} />
      <IconButton icon="search-outline" onPress={() => {}} size={24} />
    </>
  );

  return (
    <SubscriptionListScreen
      title="Feeds"
      filter={feedsFilter}
      getSubtitle={getFeedsSubtitle}
      headerActions={headerActions}
      emptyMessage="No feeds yet"
    />
  );
};
