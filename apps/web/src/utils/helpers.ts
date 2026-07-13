// Small presentation helpers ported from apps/mobile/src/utils/helpers.ts.
export const pluralize = (count: number, singular: string, plural?: string): string => {
  return count === 1 ? singular : (plural ?? `${singular}s`);
};

// Matches the format used by the article reader (ReadArticle/ExploreArticle).
export const formatPublishedDate = (value?: string): string | null => {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
};
