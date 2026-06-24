import React from 'react';
import { SubscriptionListScreen } from '../components/SubscriptionListScreen';
import { UnifiedSubscription } from '@cairn/shared';

const feedsFilter = (s: UnifiedSubscription) => s.type !== 'email';

const getFeedsSubtitle = (s: UnifiedSubscription): string | undefined =>
  s.rss_data?.feed_url ?? s.description;

export const FeedsScreen: React.FC = () => (
  <SubscriptionListScreen
    title="Feeds"
    filter={feedsFilter}
    getSubtitle={getFeedsSubtitle}
    emptyMessage="No feeds yet"
  />
);
