import React from 'react';
import { TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { BookmarkIcon, ArchiveIcon, ReturnIcon, ThumbsUpIcon, ThumbsDownIcon } from '../icons';
import { Colors } from '../../constants';

export type QuickAccessButtonIcon = 'bookmark' | 'archive' | 'return' | 'thumbs-up' | 'thumbs-down';

interface QuickAccessButtonProps {
  icon: QuickAccessButtonIcon;
  onPress: () => void;
  active?: boolean;
}

export const QuickAccessButton: React.FC<QuickAccessButtonProps> = ({
  icon,
  onPress,
  active = false,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const renderIcon = () => {
    const iconColor = active ? colors.primary : colors.text;
    const size = 24;

    switch (icon) {
      case 'bookmark':
        return <BookmarkIcon size={size} color={iconColor} />;
      case 'archive':
        return <ArchiveIcon size={size} color={iconColor} />;
      case 'return':
        return <ReturnIcon size={size} color={iconColor} />;
      case 'thumbs-up':
        return <ThumbsUpIcon size={size} color={iconColor} />;
      case 'thumbs-down':
        return <ThumbsDownIcon size={size} color={iconColor} />;
    }
  };

  return (
    <TouchableOpacity
      style={[
        styles.button,
        { backgroundColor: active ? colors.hover : 'transparent' }
      ]}
      onPress={onPress}
      activeOpacity={0.7}
    >
      {renderIcon()}
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  button: {
    width: 42,
    height: 42,
    borderRadius: 21,
    justifyContent: 'center',
    alignItems: 'center',
  },
});
