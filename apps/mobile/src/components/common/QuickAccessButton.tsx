import React from 'react';
import { TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { BookmarkIcon, ArchiveIcon, ReadIcon, ReturnIcon, ArrowDownIcon } from '../icons';
import { Colors } from '../../constants';

export type QuickAccessButtonIcon = 'bookmark' | 'archive' | 'save-to-reads' | 'return' | 'next-article';

interface QuickAccessButtonProps {
  icon: QuickAccessButtonIcon;
  onPress: () => void;
  accessibilityLabel: string;
  active?: boolean;
  disabled?: boolean;
}

export const QuickAccessButton: React.FC<QuickAccessButtonProps> = ({
  icon,
  onPress,
  accessibilityLabel,
  active = false,
  disabled = false,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const renderIcon = () => {
    const iconColor = disabled ? colors.textSecondary : active ? colors.primary : colors.text;
    const size = 24;

    switch (icon) {
      case 'bookmark':
        return <BookmarkIcon size={size} color={iconColor} filled={active} />;
      case 'archive':
        return <ArchiveIcon size={size} color={iconColor} />;
      case 'save-to-reads':
        return <ReadIcon size={size} color={iconColor} />;
      case 'return':
        return <ReturnIcon size={size} color={iconColor} />;
      case 'next-article':
        return <ArrowDownIcon size={size} color={iconColor} />;
    }
  };

  return (
    <TouchableOpacity
      style={styles.button}
      onPress={disabled ? undefined : onPress}
      activeOpacity={disabled ? 1 : 0.7}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      accessibilityState={{ disabled, selected: active }}
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
