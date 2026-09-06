import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { ReadScreen } from './ReadScreen';
import { ReadService } from '../services/read';

// Regression proof for the PR #382 review defect: ReadScreen's focus effect
// only calls `load(true)` (the network refresh) inside
// `ArticleStore.listRecent(...).then(...)`, with no `.catch`. If the SQLite
// read rejected, `load(true)` would never run and the screen would be stuck
// on the initial spinner forever.
//
// This file deliberately does NOT mock '../services/articleStore' (unlike
// ReadScreen.test.tsx) — it exercises the *real* ArticleStore against a DB
// that fails to open, to prove the screen recovers via articleStore.ts's
// "reads never reject" contract rather than a screen-level `.catch` (the
// task explicitly rejects adding one per call site).

jest.mock('../services/read', () => ({
  ReadService: {
    listUserContents: jest.fn(),
  },
}));

jest.mock('expo-sqlite', () => ({
  openDatabaseAsync: jest.fn().mockRejectedValue(new Error('simulated SQLite open failure')),
}));

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn() }),
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;

describe('ReadScreen recovers when the local store read fails', () => {
  it('still fires the network load and clears the spinner', async () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    mockedReadService.listUserContents.mockResolvedValue({
      contents: [],
      total_count: 0,
      limit: 20,
      cursor: '',
      has_more: false,
    });

    render(<ReadScreen />);

    await waitFor(() => expect(mockedReadService.listUserContents).toHaveBeenCalled());
    expect(await screen.findByText('No saved articles yet')).toBeTruthy();
  });
});
