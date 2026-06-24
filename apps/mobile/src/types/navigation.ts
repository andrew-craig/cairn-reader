import { Article } from '@cairn/shared';

export type RootStackParamList = {
  MainTabs: undefined;
  ArticleDetail: { article: Article; articles?: Article[]; currentIndex?: number };
  ExploreArticleDetail: { article: Article; articles?: Article[]; currentIndex?: number };
  AddArticle: undefined;
  Bookmarks: undefined;
  Votes: undefined;
  Account: undefined;
  About: undefined;
  Feeds: undefined;
  Newsletters: undefined;
};

export type MainTabParamList = {
  Explore: undefined;
  Read: undefined;
  You: undefined;
};
