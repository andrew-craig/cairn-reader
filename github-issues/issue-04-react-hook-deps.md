# [MEDIUM] Fix React Hook dependency warnings in ExploreScreen

Labels: bug, priority:medium, mobile, react

## Problem

The `ExploreScreen` component has a `useEffect` hook with a missing dependency that could cause stale closures or missed re-renders.

## Impact

- **Stale Closures**: The effect may capture an old version of `loadExploreArticles`
- **Incorrect Behavior**: Changes to dependencies won't trigger the effect to re-run
- **React Warnings**: Development mode shows console warnings

## Affected File

**File:** `src/screens/ExploreScreen.tsx:35`

### Current Code (Line 35)
```typescript
useEffect(() => {
  loadExploreArticles();
}, []); // Missing dependency: 'loadExploreArticles'
```

### Issue
ESLint warning: `React Hook useEffect has a missing dependency: 'loadExploreArticles'. Either include it or remove the dependency array (react-hooks/exhaustive-deps)`

## Recommended Fixes

### Option 1: Wrap function in useCallback (Recommended)
```typescript
const loadExploreArticles = useCallback(async (minArticles?: number) => {
  if (loadingRef.current) return;

  setLoading(true);
  loadingRef.current = true;
  setError(null);

  try {
    // ... existing logic
  } catch (err) {
    // ... existing error handling
  } finally {
    setLoading(false);
    loadingRef.current = false;
  }
}, [/* add dependencies that loadExploreArticles uses */]);

useEffect(() => {
  loadExploreArticles();
}, [loadExploreArticles]);
```

### Option 2: Move function inside useEffect (If only used there)
```typescript
useEffect(() => {
  const loadExploreArticles = async (minArticles?: number) => {
    // ... function body
  };

  loadExploreArticles();
}, [/* dependencies used inside loadExploreArticles */]);
```

### Option 3: Explicitly disable rule (Only if intentional)
```typescript
useEffect(() => {
  loadExploreArticles();
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

**Note:** Only use Option 3 if you're certain the function should only run once on mount.

## Implementation Steps

1. Review `loadExploreArticles` function to identify all dependencies it uses
2. Choose appropriate fix based on usage pattern
3. Wrap function in `useCallback` with proper dependencies (recommended)
4. Update `useEffect` to include the function in dependency array
5. Test to ensure articles still load correctly on mount

## Testing

After fixes, verify:
```bash
cd apps/mobile
npm run lint        # Should show no React hooks warnings
npm start           # Test in development mode
```

Manual testing:
- [ ] Articles load correctly when app starts
- [ ] No console warnings about missing dependencies
- [ ] Refresh functionality still works
- [ ] Loading states work correctly

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Section: "React Hook Dependency Warning")
- React Docs: [useEffect dependencies](https://react.dev/reference/react/useEffect#specifying-reactive-dependencies)
- React Docs: [useCallback](https://react.dev/reference/react/useCallback)
- Estimated Effort: **30 minutes**

## Acceptance Criteria

- [ ] `useEffect` dependency array includes all required dependencies
- [ ] No ESLint warnings for react-hooks/exhaustive-deps
- [ ] Articles load correctly on component mount
- [ ] All existing functionality works as expected
- [ ] App runs without console warnings
