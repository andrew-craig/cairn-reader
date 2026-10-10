import { Article } from '@cairn/shared';

export type RootStackParamList = {
  MainTabs: undefined;
  ArticleDetail: { article: Article; articles?: Article[]; currentIndex?: number; onArchived?: (articleId: string) => void };
  AddArticle: undefined;
  Bookmarks: undefined;
  Account: undefined;
  About: undefined;
  Feeds: undefined;
  Newsletters: undefined;
};

export type MainTabParamList = {
  Feed: undefined;
  Reads: undefined;
  You: undefined;
};
