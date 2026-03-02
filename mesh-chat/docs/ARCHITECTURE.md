# 시스템 아키텍처

## 1. 목표

Mesh Chat의 목표는 다음 요구사항을 동시에 만족하는 메신저 시스템이다.

- 실시간 메시지 송수신
- 다중 디바이스 지원
- 채널, DM, 그룹 채팅 지원
- 미디어 업로드
- presence와 typing indicator
- 추후 federation 확장 가능 구조

## 2. 전체 구성

```text
Clients
├── React Web App
└── Flutter Mobile App
        │
        ▼
Gateway
└── Nginx
        │
        ▼
Backend
├── Auth
├── Channel
├── Message
├── WebSocket Hub
├── Presence
├── Media
└── Federation
        │
        ▼
Data Layer
├── PostgreSQL
├── Redis
└── MinIO
```

## 3. 컴포넌트별 역할

### React Web App

- 브라우저 인증 플로우 담당
- 메시지 목록, 채널 목록, 입력 UI 제공
- IndexedDB에 키 저장
- WebSocket으로 실시간 이벤트 수신

### Flutter Mobile App

- 모바일 인증 및 키 저장
- 채팅 UI, 알림, 첨부 UI
- `flutter_secure_storage`와 Hive 사용
- 추후 FCM/APNS 연동

### Nginx

- TLS 종료
- API 및 WebSocket reverse proxy
- federation 엔드포인트 노출
- rate limit
- 보안 헤더 추가

### Go Backend

- REST API 제공
- JWT 기반 인증
- Redis 기반 challenge, refresh revoke, presence
- WebSocket 허브로 실시간 전파
- PostgreSQL 영속화
- MinIO 미디어 저장

### PostgreSQL

- 사용자, 채널, 메시지, 멤버십, 디바이스, federation 서버 테이블 저장

### Redis

- 인증 challenge TTL 저장
- refresh token blacklist
- presence 상태
- typing 상태
- WebSocket 멀티 인스턴스 pub/sub

### MinIO

- 이미지와 파일 저장
- presigned URL 발급

## 4. 핵심 요청 흐름

### 회원가입

1. 클라이언트가 챌린지를 요청한다.
2. 서버가 Redis에 챌린지를 저장한다.
3. 클라이언트가 개인키로 챌린지를 서명한다.
4. 서버가 공개키와 서명을 검증한다.
5. 사용자와 디바이스를 생성하고 JWT를 발급한다.

### 로그인

1. 클라이언트가 username으로 챌린지를 요청한다.
2. 클라이언트가 keystore에서 키를 읽는다.
3. 챌린지를 서명해 로그인 요청을 보낸다.
4. 서버가 서명을 검증한다.
5. access token과 refresh token을 발급한다.

### 메시지 송신

1. 클라이언트는 메시지를 암호화한다.
2. WebSocket 또는 HTTP fallback으로 메시지를 보낸다.
3. 서버는 DB에 저장한다.
4. WebSocket 허브가 같은 채널의 클라이언트에 이벤트를 브로드캐스트한다.
5. 멀티 인스턴스인 경우 Redis pub/sub로 다른 backend 인스턴스에도 전파한다.

### 미디어 업로드

1. 클라이언트가 multipart/form-data로 업로드한다.
2. 서버가 MIME 타입과 크기를 검증한다.
3. MinIO에 저장한다.
4. 클라이언트는 응답으로 받은 URL을 메시지에 첨부한다.

## 5. 데이터 경계

### PostgreSQL에 저장되는 것

- 사용자 메타데이터
- 채널 및 채널 멤버십
- 메시지 본문과 메타데이터
- 반응
- 디바이스 정보
- federation 서버 레지스트리

### Redis에 저장되는 것

- 단기 TTL challenge
- 로그아웃된 refresh token 상태
- online/away/dnd 상태
- typing 상태
- 멀티 노드 메시지 전파

### 클라이언트 로컬에 저장되는 것

- 웹: IndexedDB에 keypair, channel key
- 모바일: secure storage에 identity key, Hive에 channel key와 message cache

## 6. 현재 설계상 결정

### WebSocket 우선, HTTP 보조

실시간 전송은 WebSocket이 주 경로다. 하지만 메시지 생성은 HTTP fallback도 존재한다. 이 구조는 네트워크 불안정 환경과 모바일 foreground/background 전환을 고려한 것이다.

### 서비스 분리 수준

현재 저장소는 논리적으로 `auth`, `channel`, `message`, `presence`, `media`, `federation`으로 나뉘지만, 실제 배포는 하나의 Go 프로세스로 구동된다. 향후 분산이 필요하면 패키지 단위로 분리하기 쉽도록 내부 경계를 먼저 만들어 둔 상태다.

### federation은 최소 구현

서버 간 이벤트 모델, DNS TXT 기반 discovery, 서버 키 검증, 송수신 엔드포인트는 들어가 있다. 다만 원격 메시지를 로컬 이벤트 모델에 완전히 병합하는 로직은 아직 확장 단계다.

## 7. 확장 포인트

- OpenTelemetry 기반 tracing
- background job 큐
- message delivery ack
- device별 암호키 fan-out
- federation replay protection
- message search 인덱스
- moderation 및 abuse detection
