import { Article } from './article';

export type RootStackParamList = {
  MainTabs: undefined;
  ArticleDetail: { article: Article; articles?: Article[]; currentIndex?: number };
  ExploreArticleDetail: { article: Article; articles?: Article[]; currentIndex?: number };
  AddArticle: undefined;
  Bookmarks: undefined;
  Votes: undefined;
  Account: undefined;
  Feeds: undefined;
  Newsletters: undefined;
};

export type MainTabParamList = {
  Explore: undefined;
  Read: undefined;
  You: undefined;
};
