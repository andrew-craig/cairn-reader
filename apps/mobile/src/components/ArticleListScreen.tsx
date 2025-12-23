import React, { ReactNode } from 'react';
import {
  View,
  Text,
  useColorScheme,
  FlatList,
  ActivityIndicator,
} from 'react-native';
import { Colors, GlobalStyles } from '../constants';
import { ArticleRow } from './common/ArticleRow';
import { Article } from '../types';

interface ArticleListScreenProps {
  title: string;
  articles: Article[];
  loading: boolean;
  headerActions?: ReactNode;
  onArticlePress: (article: Article) => void;
  onRefresh?: () => void;
  refreshing?: boolean;
  emptyMessage?: string;
}

export const ArticleListScreen: React.FC<ArticleListScreenProps> = ({
  title,
  articles,
  loading,
  headerActions,
  onArticlePress,
  onRefresh,
  refreshing = false,
  emptyMessage = 'No articles found',
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const renderHeader = () => (
    <View style={[GlobalStyles.header, { backgroundColor: colors.background }]}>
      <View style={GlobalStyles.headerLeft}>
        <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>{title}</Text>
      </View>
      {headerActions && (
        <View style={GlobalStyles.headerRight}>{headerActions}</View>
      )}
    </View>
  );

  const renderArticle = ({ item }: { item: Article }) => (
    <ArticleRow article={item} onPress={() => onArticlePress(item)} />
  );

  const renderEmpty = () => (
    <View style={GlobalStyles.emptyContainer}>
      <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
        {emptyMessage}
      </Text>
    </View>
  );

  if (loading) {
    return (
      <View style={[GlobalStyles.container, GlobalStyles.centerContent, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  return (
    <View style={[GlobalStyles.container, { backgroundColor: colors.background }]}>
      <FlatList
        data={articles}
        renderItem={renderArticle}
        keyExtractor={(item) => item.id}
        ListHeaderComponent={renderHeader}
        ListEmptyComponent={renderEmpty}
        showsVerticalScrollIndicator={false}
        onRefresh={onRefresh}
        refreshing={refreshing}
      />
    </View>
  );
};
