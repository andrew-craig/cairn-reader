import React from 'react';
import { StyleSheet } from 'react-native';
import { render, screen } from '@testing-library/react-native';
import { ArticleRow } from './ArticleRow';
import { Colors, FontFamily } from '../../constants';
import { Article } from '../../types';

const baseArticle: Article = {
  id: 'a1',
  url: 'https://example.com/article',
  title: 'Test article title',
  author: 'Author',
  tags: [],
  isRead: false,
  isFavorite: false,
  list: 'reads' as const,
  addedAt: Date.now(),
};

describe('ArticleRow', () => {
  it('renders unread titles with the bold font family', () => {
    render(<ArticleRow article={baseArticle} onPress={() => {}} />);
    const flat = StyleSheet.flatten(screen.getByText(baseArticle.title).props.style);
    expect(flat.fontFamily).toBe(FontFamily.defaultBold);
  });

  it('renders read titles with the default (non-bold) font family', () => {
    render(<ArticleRow article={{ ...baseArticle, isRead: true }} onPress={() => {}} />);
    const flat = StyleSheet.flatten(screen.getByText(baseArticle.title).props.style);
    expect(flat.fontFamily).toBe(FontFamily.default);
  });

  it('renders partially read titles in the secondary colour at the unread weight', () => {
    render(<ArticleRow article={{ ...baseArticle, scrollFraction: 0.4 }} onPress={() => {}} />);
    const flat = StyleSheet.flatten(screen.getByText(baseArticle.title).props.style);
    expect(flat.fontFamily).toBe(FontFamily.defaultBold);
    expect(flat.color).toBe(Colors.light.textSecondary);
  });

  it('shows the waiting-to-save note only for a pending row', () => {
    const note = 'Waiting to save — will sync when online';
    const { rerender } = render(<ArticleRow article={baseArticle} onPress={() => {}} />);
    expect(screen.queryByText(note)).toBeNull();

    rerender(<ArticleRow article={baseArticle} onPress={() => {}} pending />);
    expect(screen.getByText(note)).toBeTruthy();
  });
});
