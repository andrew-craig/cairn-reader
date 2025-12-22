import { Article } from './article';

export type RootStackParamList = {
  MainTabs: undefined;
  ArticleDetail: { article: Article };
  AddArticle: undefined;
};

export type MainTabParamList = {
  Explore: undefined;
  Read: undefined;
  Settings: undefined;
};
