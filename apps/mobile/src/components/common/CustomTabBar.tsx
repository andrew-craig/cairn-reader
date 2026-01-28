import React from 'react';
import { View, TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { BottomTabBarProps } from '@react-navigation/bottom-tabs';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Svg, Path } from 'react-native-svg';
import { Colors, Spacing } from '../../constants';

type TabIconProps = {
  name: 'explore' | 'read' | 'profile';
  focused: boolean;
  color: string;
};

const TabIcon: React.FC<TabIconProps> = ({ name, focused, color }) => {
  if (name === 'explore') {
    return (
      <Svg width={32} height={32} viewBox="0 0 26 26" fill={focused ? color : 'none'} stroke={focused ? 'none' : color} strokeWidth={focused ? 0 : 2}>
        <Path
          d="M24.8548 14.8611L23.3415 13.3478C23.1137 13.1196 22.9241 12.8563 22.7801 12.5678L21.3401 9.68781C21.3009 9.60964 21.2437 9.54191 21.1732 9.49018C21.1027 9.43845 21.021 9.40421 20.9346 9.39028C20.8483 9.37635 20.7599 9.38313 20.6767 9.41005C20.5935 9.43697 20.5179 9.48328 20.4561 9.54514C20.317 9.68411 20.1433 9.78334 19.9529 9.83257C19.7626 9.88181 19.5625 9.87924 19.3735 9.82514L17.6761 9.34114C17.4046 9.2648 17.1147 9.28769 16.8585 9.40572C16.6022 9.52375 16.3965 9.7292 16.278 9.98522C16.1595 10.2412 16.1362 10.5311 16.2121 10.8028C16.2879 11.0745 16.4581 11.3102 16.6921 11.4678L17.4748 11.9878C18.2615 12.5145 18.3735 13.6278 17.7041 14.2971L17.4375 14.5638C17.1548 14.8465 16.9975 15.2278 16.9975 15.6251V16.1718C16.9975 16.7171 16.8508 17.2505 16.5708 17.7158L14.8175 20.6371C14.5675 21.054 14.2138 21.399 13.7909 21.6385C13.368 21.878 12.8902 22.0039 12.4041 22.0038C12.0311 22.0038 11.6733 21.8556 11.4095 21.5918C11.1457 21.328 10.9975 20.9702 10.9975 20.5971V19.0345C10.9975 17.8078 10.2508 16.7051 9.11213 16.2491L8.2388 15.9011C7.60338 15.6468 7.07405 15.1828 6.73863 14.5862C6.40322 13.9896 6.2819 13.2962 6.3948 12.6211L6.40413 12.5651C6.46611 12.1942 6.59728 11.8383 6.7908 11.5158L6.9108 11.3158C7.22922 10.7856 7.70376 10.3667 8.26944 10.1165C8.83511 9.86641 9.46428 9.79723 10.0708 9.91848L11.6415 10.2331C12.0111 10.3069 12.3949 10.2387 12.7166 10.0423C13.0382 9.84592 13.2742 9.53561 13.3775 9.17314L13.6548 8.19981C13.7509 7.86355 13.7264 7.50429 13.5857 7.18414C13.445 6.86398 13.1968 6.60306 12.8841 6.44648L11.9975 6.00381L11.8761 6.12514C11.5976 6.40372 11.2668 6.6247 10.9029 6.77546C10.5389 6.92622 10.1488 7.00381 9.7548 7.00381H9.51479C9.18279 7.00381 8.86546 7.13714 8.63213 7.36914C8.41828 7.58493 8.13274 7.7148 7.82956 7.73416C7.52638 7.75353 7.22663 7.66105 6.98707 7.47423C6.7475 7.28741 6.58477 7.01923 6.52967 6.72047C6.47458 6.42171 6.53096 6.11312 6.68813 5.85314L8.56946 2.71581C8.75698 2.40396 8.88619 2.06059 8.9508 1.70248M24.8548 14.8611C25.1778 12.804 24.9598 10.6982 24.2225 8.75078C23.4851 6.80335 22.2537 5.08132 20.6492 3.75396C19.0448 2.42659 17.1225 1.53959 15.0714 1.18014C13.0203 0.820686 10.911 1.00116 8.9508 1.70381C6.95656 2.41866 5.18711 3.64871 3.82226 5.26895C2.45742 6.8892 1.54583 8.84189 1.18014 10.9286C0.814448 13.0153 1.00769 15.1616 1.74022 17.1494C2.47276 19.1372 3.71847 20.8956 5.35078 22.246C6.98308 23.5964 8.94378 24.4907 11.0336 24.8378C13.1235 25.1849 15.268 24.9726 17.2492 24.2225C19.2304 23.4723 20.9778 22.211 22.3136 20.5668C23.6495 18.9226 24.5262 16.954 24.8548 14.8611Z"
        />
      </Svg>
    );
  }

  if (name === 'read') {
    return (
      <Svg width={32} height={32} viewBox="0 0 24 24" fill={focused ? color : 'none'} stroke={focused ? 'none' : color}>
        {focused ? (
          <Path d="M11.25 4.533A9.707 9.707 0 0 0 6 3a9.735 9.735 0 0 0-3.25.555.75.75 0 0 0-.5.707v14.25a.75.75 0 0 0 1 .707A8.237 8.237 0 0 1 6 18.75c1.995 0 3.823.707 5.25 1.886V4.533ZM12.75 20.636A8.214 8.214 0 0 1 18 18.75c.966 0 1.89.166 2.75.47a.75.75 0 0 0 1-.708V4.262a.75.75 0 0 0-.5-.707A9.735 9.735 0 0 0 18 3a9.707 9.707 0 0 0-5.25 1.533v16.103Z" />
        ) : (
          <Path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M12 6.042A8.967 8.967 0 0 0 6 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 0 1 6 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 0 1 6-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0 0 18 18a8.967 8.967 0 0 0-6 2.292m0-14.25v14.25"
          />
        )}
      </Svg>
    );
  }

  // profile icon
  return (
    <Svg width={32} height={32} viewBox="0 0 24 24" fill={focused ? color : 'none'} stroke={focused ? 'none' : color}>
      {focused ? (
        <Path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M18.685 19.097A9.723 9.723 0 0 0 21.75 12c0-5.385-4.365-9.75-9.75-9.75S2.25 6.615 2.25 12a9.723 9.723 0 0 0 3.065 7.097A9.716 9.716 0 0 0 12 21.75a9.716 9.716 0 0 0 6.685-2.653Zm-12.54-1.285A7.486 7.486 0 0 1 12 15a7.486 7.486 0 0 1 5.855 2.812A8.224 8.224 0 0 1 12 20.25a8.224 8.224 0 0 1-5.855-2.438ZM15.75 9a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0Z"
        />
      ) : (
        <Path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M17.982 18.725A7.488 7.488 0 0 0 12 15.75a7.488 7.488 0 0 0-5.982 2.975m11.963 0a9 9 0 1 0-11.963 0m11.963 0A8.966 8.966 0 0 1 12 21a8.966 8.966 0 0 1-5.982-2.275M15 9.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
        />
      )}
    </Svg>
  );
};

const getIconName = (routeName: string): 'explore' | 'read' | 'profile' => {
  switch (routeName) {
    case 'Explore':
      return 'explore';
    case 'Read':
      return 'read';
    case 'You':
      return 'profile';
    default:
      return 'explore';
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
      <View
        style={[
          styles.tabBar,
          {
            backgroundColor: colors.card,
            shadowColor: '#0F172A',
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

          const iconName = getIconName(route.name);

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
              <TabIcon
                name={iconName}
                focused={isFocused}
                color={colors.text}
              />
            </TouchableOpacity>
          );
        })}
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
  tabBar: {
    flexDirection: 'row',
    gap: 16,
    height: 54,
    paddingVertical: 6,
    paddingHorizontal: 12,
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
  tab: {
    width: 86,
    height: 42,
    borderRadius: 21,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
