import React from 'react';
import { useColorScheme, TouchableOpacity } from 'react-native';
import { createStackNavigator } from '@react-navigation/stack';
import { Ionicons } from '@expo/vector-icons';
import { RootStackParamList } from '../types';
import { AddArticleScreen, ArticleDetailScreen } from '../screens';
import { TabNavigator } from './TabNavigator';
import { Colors } from '../constants';

const Stack = createStackNavigator<RootStackParamList>();

export default function RootNavigator() {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  return (
    <Stack.Navigator
      screenOptions={{
        headerStyle: {
          backgroundColor: colors.card,
        },
        headerTintColor: colors.text,
        headerBackTitleVisible: false,
      }}
    >
      <Stack.Screen
        name="MainTabs"
        component={TabNavigator}
        options={({ navigation }) => ({
          headerShown: false,
          headerRight: () => (
            <TouchableOpacity
              onPress={() => navigation.navigate('AddArticle')}
              style={{ marginRight: 16 }}
            >
              <Ionicons name="add-circle-outline" size={28} color={colors.primary} />
            </TouchableOpacity>
          ),
        })}
      />
      <Stack.Screen
        name="ArticleDetail"
        component={ArticleDetailScreen}
        options={{
          title: 'Article',
          presentation: 'card',
        }}
      />
      <Stack.Screen
        name="AddArticle"
        component={AddArticleScreen}
        options={{
          title: 'Add Article',
          presentation: 'modal',
        }}
      />
    </Stack.Navigator>
  );
}
