import { isScrollTargetReachable } from './ArticleContent';

describe('isScrollTargetReachable', () => {
  it('is false while the content is still shorter than the saved fraction target', () => {
    // fraction 0.8 on contentHeight 800 → target 640, max scroll 800-600=200 < 640
    expect(isScrollTargetReachable(800, 600, 0.8)).toBe(false);
  });

  it('is true once the content has grown tall enough to reach the fraction', () => {
    // fraction 0.5 on contentHeight 2600 → target 1300, max scroll 2600-600=2000 >= 1300
    expect(isScrollTargetReachable(2600, 600, 0.5)).toBe(true);
  });

  it('is false before the viewport has been measured', () => {
    expect(isScrollTargetReachable(5000, 0, 0.5)).toBe(false);
  });

  it('treats a fraction at the exact max scroll boundary as reachable', () => {
    // fraction 0.5 on contentHeight 1200 → target 600, max scroll 1200-600=600 >= 600
    expect(isScrollTargetReachable(1200, 600, 0.5)).toBe(true);
    // fraction 0.51 → target 612, max scroll 600 < 612
    expect(isScrollTargetReachable(1200, 600, 0.51)).toBe(false);
  });

  it('handles zero fraction', () => {
    expect(isScrollTargetReachable(2000, 600, 0)).toBe(true);
  });
});
