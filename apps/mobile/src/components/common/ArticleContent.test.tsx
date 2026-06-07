import { isScrollTargetReachable } from './ArticleContent';

describe('isScrollTargetReachable', () => {
  it('is false while the content is still shorter than the saved offset', () => {
    // Email mid-render: only part of the body has laid out so far.
    expect(isScrollTargetReachable(800, 600, 2000)).toBe(false);
  });

  it('is true once the content has grown tall enough to reach the offset', () => {
    // Fully rendered: target offset leaves at least one viewport below it.
    expect(isScrollTargetReachable(2600, 600, 2000)).toBe(true);
  });

  it('is false before the viewport has been measured', () => {
    // onContentSizeChange can fire before onLayout, leaving layoutHeight at 0.
    expect(isScrollTargetReachable(5000, 0, 2000)).toBe(false);
  });

  it('treats an offset exactly at the max scroll as reachable', () => {
    expect(isScrollTargetReachable(2600, 600, 2000)).toBe(true);
    expect(isScrollTargetReachable(2599, 600, 2000)).toBe(false);
  });
});
