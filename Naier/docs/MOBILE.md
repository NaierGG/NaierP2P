# 모바일 클라이언트 상세 문서

## 1. 개요

모바일 클라이언트는 Flutter 기반이다. 목표는 웹과 같은 실시간 채팅 경험을 iOS/Android에서 제공하는 것이다.

현재는 구조와 UI 스캐폴드가 먼저 들어가 있고, 실제 API/WS/푸시 연동은 다음 단계에서 이어 붙이는 상태다.

## 2. 기술 스택

- Flutter 3.x
- Riverpod
- GoRouter
- Dio
- web_socket_channel
- flutter_secure_storage
- Hive
- firebase_messaging
- flutter_local_notifications

## 3. 디렉토리 구조

```text
mobile/lib/
├── app/
│   ├── router.dart
│   └── theme.dart
├── core/
│   ├── crypto/
│   ├── network/
│   └── storage/
├── features/
│   ├── auth/
│   ├── channels/
│   ├── messages/
│   └── notifications/
└── shared/
    └── models/
```

## 4. 앱 부트스트랩

### `main.dart`

- Flutter binding 초기화
- storage bootstrap
- Riverpod `ProviderScope`
- `MaterialApp.router`

### `router.dart`

라우팅:

- `/auth/login`
- `/auth/keygen`
- `/app`
- `/app/channel/:id`

인증 상태에 따라 redirect한다.

## 5. 저장 전략

### Secure Storage

민감정보 저장:

- identity keypair
- auth session

### Hive

캐시 저장:

- channel keys
- message cache

## 6. 네트워크 계층

### API Client

- Dio 기반
- access token 자동 첨부
- 401 시 refresh 시도
- refresh 실패 시 세션 clear

### WebSocket Client

- 재연결
- 이벤트 스트림 broadcast
- 앱 foreground/background 상태에 따라 연결 관리

## 7. 채팅 UI

### Message 모델

`ChatMessage`는 현재 UI용 모델로 분리돼 있다.

포함 정보:

- sender
- content
- type
- createdAt
- reply preview
- attachment label
- delivery status
- reactions

### Message List

지원 요소:

- reverse scroll
- 날짜 구분선
- 연속 메시지 그룹핑
- 이전 메시지 로드 훅

### Message Bubble

지원 요소:

- 텍스트 / 이미지 / 파일 렌더링
- 길게 누르기 컨텍스트 액션
- reaction chip
- delivery status 아이콘
- reply preview

### Message Input

지원 요소:

- reply/edit 상태 배너
- 첨부 버튼
- 이모지 버튼 자리
- 타이핑 debounce
- optimistic send payload 생성

### Channel Detail

현재 화면은 실서비스 연결 전 임시 상태 모델을 들고 있다.

지원 요소:

- optimistic message append
- sending -> sent/failed 상태 변경
- reply/edit/delete
- reaction toggle
- typing 표시

## 8. 알림

`NotificationService`는 다음 역할의 스캐폴드다.

- FCM 권한 요청
- foreground 수신 시 로컬 알림 표시
- notification tap 시 channel route intent 생성

아직 미연결인 부분:

- 실제 Firebase 프로젝트 설정
- token을 backend devices 테이블로 등록
- 탭 시 GoRouter와 실제 딥링크 연결

## 9. 현재 구현 상태

들어간 것:

- 앱 라우팅
- theme
- secure storage / Hive 골격
- 채팅 UI 기본 동작
- notification service 골격

아직 필요한 것:

- 실제 REST API 연결
- WebSocket 이벤트 모델 연결
- 암호화 전송과 복호화 연결
- 이미지/파일 picker 실연결
- 푸시 토큰 등록
- Flutter analyze 통과 확인

## 10. 실행 체크리스트

```bash
flutter pub get
flutter analyze
flutter run
```

이 저장소가 작성된 환경에서는 `flutter`가 설치되어 있지 않아 위 과정을 실제로 돌리진 못했다.
