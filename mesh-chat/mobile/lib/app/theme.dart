import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';

ThemeData buildLightTheme() {
  const seed = Color(0xFFD56D3A);
  return ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(
      seedColor: seed,
      brightness: Brightness.light,
      surface: const Color(0xFFF6F1E9),
    ),
    scaffoldBackgroundColor: const Color(0xFFF4EFE8),
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.transparent,
      elevation: 0,
      centerTitle: false,
    ),
    cupertinoOverrideTheme: const CupertinoThemeData(
      primaryColor: seed,
      scaffoldBackgroundColor: Color(0xFFF4EFE8),
    ),
  );
}

ThemeData buildDarkTheme() {
  const seed = Color(0xFFFFB36D);
  return ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(
      seedColor: seed,
      brightness: Brightness.dark,
      surface: const Color(0xFF12151B),
    ),
    scaffoldBackgroundColor: const Color(0xFF0E1116),
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.transparent,
      elevation: 0,
      centerTitle: false,
    ),
    cupertinoOverrideTheme: const CupertinoThemeData(
      brightness: Brightness.dark,
      primaryColor: seed,
      scaffoldBackgroundColor: Color(0xFF0E1116),
    ),
  );
}
