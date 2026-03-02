class AuthSession {
  const AuthSession({
    this.userId,
    this.username,
    this.accessToken,
    this.refreshToken,
    this.isHydrated = false,
  });

  final String? userId;
  final String? username;
  final String? accessToken;
  final String? refreshToken;
  final bool isHydrated;

  bool get isAuthenticated => accessToken != null && userId != null;

  AuthSession copyWith({
    String? userId,
    String? username,
    String? accessToken,
    String? refreshToken,
    bool? isHydrated,
    bool clearTokens = false,
  }) {
    return AuthSession(
      userId: userId ?? this.userId,
      username: username ?? this.username,
      accessToken: clearTokens ? null : accessToken ?? this.accessToken,
      refreshToken: clearTokens ? null : refreshToken ?? this.refreshToken,
      isHydrated: isHydrated ?? this.isHydrated,
    );
  }
}
