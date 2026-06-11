// Small presentation helpers ported from apps/mobile/src/utils/helpers.ts.
export const pluralize = (count: number, singular: string, plural?: string): string => {
  return count === 1 ? singular : (plural ?? `${singular}s`);
};
