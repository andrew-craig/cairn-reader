import React from 'react';
import { View, StyleSheet, useColorScheme, Platform } from 'react-native';
import { BlurView } from 'expo-blur';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { QuickAccessButton, QuickAccessButtonIcon } from './QuickAccessButton';
import { Colors, Spacing } from '../../constants';

interface BottomActionMenuAction {
  icon: QuickAccessButtonIcon;
  onPress: () => void;
  active?: boolean;
}

interface BottomActionMenuProps {
  actions: BottomActionMenuAction[];
}

export const BottomActionMenu: React.FC<BottomActionMenuProps> = ({ actions }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  return (
    <View
      style={[
        styles.container,
        {
          paddingBottom: insets.bottom + Spacing.sm,
        },
      ]}
    >
      {Platform.OS === 'ios' ? (
        <BlurView intensity={20} tint={colorScheme === 'dark' ? 'dark' : 'light'} style={styles.blurContainer}>
          <View style={[styles.menuContainer, { backgroundColor: colors.card }]}>
            {actions.map((action, index) => (
              <QuickAccessButton
                key={index}
                icon={action.icon}
                onPress={action.onPress}
                active={action.active}
              />
            ))}
          </View>
        </BlurView>
      ) : (
        <View style={[styles.menuContainer, { backgroundColor: colors.card }]}>
          {actions.map((action, index) => (
            <QuickAccessButton
              key={index}
              icon={action.icon}
              onPress={action.onPress}
              active={action.active}
            />
          ))}
        </View>
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: Spacing.sm,
  },
  blurContainer: {
    borderRadius: 32,
    overflow: 'hidden',
  },
  menuContainer: {
    flexDirection: 'row',
    gap: 16,
    height: 54,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: Spacing.sm,
    paddingVertical: 6,
    borderRadius: 32,
    shadowColor: 'rgba(30, 41, 59, 0.2)',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 1,
    shadowRadius: 4,
    elevation: 4,
  },
});
