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
      <View style={styles.shadowWrapper}>
        <BlurView
          intensity={Platform.OS === 'ios' ? 10 : 20}
          tint={colorScheme === 'dark' ? 'dark' : 'light'}
          style={[
            styles.menuContainer,
            {
              backgroundColor: colorScheme === 'dark'
                ? 'rgba(28, 28, 30, 0.85)'
                : 'rgba(251, 250, 249, 0.85)',
            },
          ]}
        >
          {actions.map((action, index) => (
            <QuickAccessButton
              key={index}
              icon={action.icon}
              onPress={action.onPress}
              active={action.active}
            />
          ))}
        </BlurView>
      </View>
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
  shadowWrapper: {
    borderRadius: 32,
    // Shadow matching Figma design: 0px 2px 4px 0px rgba(15, 23, 42, 0.2)
    shadowColor: '#0F172A',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.2,
    shadowRadius: 4,
    elevation: 4,
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
    overflow: 'hidden',
  },
});
