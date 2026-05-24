import React from 'react';
import { StyleSheet } from 'react-native';
import { render, screen } from '@testing-library/react-native';
import { ArticleRow } from './ArticleRow';
import { FontFamily } from '../../constants';
import { Article } from '../../types';

const baseArticle: Article = {
  id: 'a1',
  url: 'https://example.com/article',
  title: 'Test article title',
  author: 'Author',
  tags: [],
  isRead: false,
  isFavorite: false,
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
});
