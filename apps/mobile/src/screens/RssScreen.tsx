import React from 'react';
import { SubscriptionListScreen } from '../components/SubscriptionListScreen';
import { UnifiedSubscription } from '@cairn/shared';

const rssFilter = (s: UnifiedSubscription) => s.type !== 'email';

const getRssSubtitle = (s: UnifiedSubscription): string | undefined =>
  s.rss_data?.feed_url ?? s.description;

export const RssScreen: React.FC = () => (
  <SubscriptionListScreen
    title="RSS"
    filter={rssFilter}
    getSubtitle={getRssSubtitle}
    emptyMessage="No RSS feeds yet"
  />
);
