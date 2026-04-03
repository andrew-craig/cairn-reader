import { Article } from './article';

export type RootStackParamList = {
  MainTabs: undefined;
  ArticleDetail: { article: Article };
  ExploreArticleDetail: { article: Article };
  AddArticle: undefined;
  Bookmarks: undefined;
  Votes: undefined;
  Account: undefined;
  Feeds: undefined;
};

export type MainTabParamList = {
  Explore: undefined;
  Read: undefined;
  You: undefined;
};
