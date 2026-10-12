const MS_PER_DAY = 24 * 60 * 60 * 1000;

export const pluralize = (count: number, singular: string, plural?: string): string => {
  return count === 1 ? singular : (plural ?? `${singular}s`);
};

// Relative label ("Today", "3 days ago", "2 months ago") for a past timestamp.
// Timestamps slightly in the future (clock skew, timezone-less publish dates)
// are treated as "Today".
export const formatDate = (timestamp: number): string => {
  const diffInDays = Math.max(0, Math.floor((Date.now() - timestamp) / MS_PER_DAY));

  if (diffInDays === 0) {
    return 'Today';
  } else if (diffInDays === 1) {
    return 'Yesterday';
  } else if (diffInDays < 7) {
    return `${diffInDays} days ago`;
  } else if (diffInDays < 30) {
    const weeks = Math.floor(diffInDays / 7);
    return `${weeks} ${pluralize(weeks, 'week')} ago`;
  } else if (diffInDays < 365) {
    const months = Math.floor(diffInDays / 30);
    return `${months} ${pluralize(months, 'month')} ago`;
  } else {
    const years = Math.floor(diffInDays / 365);
    return `${years} ${pluralize(years, 'year')} ago`;
  }
};

// Relative label for an article's ISO `publishedDate`, or null when it is
// missing or unparseable.
export const formatPublishedDate = (value?: string): string | null => {
  if (!value) return null;
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? null : formatDate(timestamp);
};
