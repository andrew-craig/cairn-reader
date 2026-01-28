import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { MainTabParamList } from '../types';
import {
  ExploreScreen,
  ReadScreen,
  YouScreen,
} from '../screens';
import { CustomTabBar } from '../components/common';

const Tab = createBottomTabNavigator<MainTabParamList>();

export const TabNavigator: React.FC = () => {
  return (
    <Tab.Navigator
      tabBar={(props) => <CustomTabBar {...props} />}
      screenOptions={{
        headerShown: false,
      }}
    >
      <Tab.Screen name="Explore" component={ExploreScreen} />
      <Tab.Screen name="Read" component={ReadScreen} />
      <Tab.Screen name="You" component={YouScreen} />
    </Tab.Navigator>
  );
};
