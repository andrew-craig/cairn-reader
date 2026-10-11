import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { MainTabParamList } from '../types';
import {
  FeedScreen,
  ReadsScreen,
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
      <Tab.Screen name="Feed" component={FeedScreen} />
      <Tab.Screen name="Reads" component={ReadsScreen} />
      <Tab.Screen name="You" component={YouScreen} />
    </Tab.Navigator>
  );
};
