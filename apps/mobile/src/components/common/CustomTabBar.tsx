import React from 'react';
import { View, TouchableOpacity, StyleSheet, useColorScheme, Platform } from 'react-native';
import { BottomTabBarProps } from '@react-navigation/bottom-tabs';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { BlurView } from 'expo-blur';
import { Colors, Spacing } from '../../constants';
import { ExploreIcon } from '../icons/ExploreIcon';
import { ReadIcon } from '../icons/ReadIcon';
import { ProfileIcon } from '../icons/ProfileIcon';

const getTabIcon = (routeName: string) => {
  switch (routeName) {
    case 'Explore':
      return ExploreIcon;
    case 'Read':
      return ReadIcon;
    case 'You':
      return ProfileIcon;
    default:
      return ExploreIcon;
  }
};

export const CustomTabBar: React.FC<BottomTabBarProps> = ({
  state,
  descriptors,
  navigation,
}) => {
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
            styles.tabBar,
            {
              backgroundColor: colorScheme === 'dark'
                ? 'rgba(28, 28, 30, 0.85)'
                : 'rgba(251, 250, 249, 0.85)',
            },
          ]}
        >
          {state.routes.map((route, index) => {
            const { options } = descriptors[route.key];
            const isFocused = state.index === index;

            const onPress = () => {
              const event = navigation.emit({
                type: 'tabPress',
                target: route.key,
                canPreventDefault: true,
              });

              if (!isFocused && !event.defaultPrevented) {
                navigation.navigate(route.name);
              }
            };

            const onLongPress = () => {
              navigation.emit({
                type: 'tabLongPress',
                target: route.key,
              });
            };

            const Icon = getTabIcon(route.name);

            return (
              <TouchableOpacity
                key={route.key}
                accessibilityRole="button"
                accessibilityState={isFocused ? { selected: true } : {}}
                accessibilityLabel={options.tabBarAccessibilityLabel}
                testID={options.tabBarTestID}
                onPress={onPress}
                onLongPress={onLongPress}
                style={[
                  styles.tab,
                  isFocused && {
                    backgroundColor: colors.hover,
                  },
                ]}
              >
                <Icon
                  size={32}
                  filled={isFocused}
                  color={colors.text}
                />
              </TouchableOpacity>
            );
          })}
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
    paddingTop: Spacing.sm,
    alignItems: 'center',
    justifyContent: 'center',
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
  tabBar: {
    flexDirection: 'row',
    gap: 16,
    height: 54,
    paddingVertical: 6,
    paddingHorizontal: 12,
    borderRadius: 32,
    overflow: 'hidden',
  },
  tab: {
    width: 86,
    height: 42,
    borderRadius: 21,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
