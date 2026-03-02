# 백엔드 상세 문서

## 1. 개요

백엔드는 Go `1.22` 기반이며, Gin을 HTTP 프레임워크로 사용한다. 핵심 역할은 다음과 같다.

- 인증과 JWT 발급
- 채널 및 메시지 CRUD
- WebSocket 실시간 허브
- Presence 관리
- 미디어 업로드
- Federation 송수신

## 2. 주요 디렉토리

```text
backend/
├── cmd/
│   ├── server/
│   └── migrate/
├── internal/
│   ├── auth/
│   ├── channel/
│   ├── message/
│   ├── websocket/
│   ├── presence/
│   ├── media/
│   ├── federation/
│   └── config/
├── migrations/
└── pkg/
```

## 3. 부트스트랩

### `cmd/server`

서버 시작 시 아래를 순서대로 수행한다.

1. 설정 로드
2. PostgreSQL 연결
3. Redis 연결
4. Gin 라우터 생성
5. 공통 미들웨어 등록
6. 서비스와 핸들러 생성
7. WebSocket 허브 시작
8. HTTP 서버 시작
9. SIGINT, SIGTERM graceful shutdown 처리

### `cmd/migrate`

마이그레이션 CLI는 `up`, `down`, `status`를 지원한다.

예시:

```bash
./migrate up
./migrate down 1
./migrate status
```

## 4. 환경 변수

환경변수 prefix는 `MESH_`다.

### 서버

- `MESH_SERVER_HOST`
- `MESH_SERVER_PORT`
- `MESH_SERVER_MODE`

### 데이터베이스

- `MESH_DATABASE_POSTGRES_DSN`
- `MESH_DATABASE_REDIS_ADDR`
- `MESH_DATABASE_REDIS_PASSWORD`

### 인증

- `MESH_AUTH_JWT_SECRET`
- `MESH_AUTH_JWT_EXPIRY`
- `MESH_AUTH_REFRESH_EXPIRY`

### 미디어

- `MESH_MEDIA_MINIO_ENDPOINT`
- `MESH_MEDIA_MINIO_BUCKET`
- `MESH_MEDIA_MINIO_ACCESS_KEY`
- `MESH_MEDIA_MINIO_SECRET_KEY`

### Federation

- `MESH_FEDERATION_SERVER_DOMAIN`
- `MESH_FEDERATION_SERVER_PUBLIC_KEY`
- `MESH_FEDERATION_SERVER_PRIVATE_KEY`

## 5. REST API

기본 prefix는 `/api/v1`다.

### Health

- `GET /health`

응답 예시:

```json
{
  "status": "ok",
  "mode": "release"
}
```

### 인증

- `POST /api/v1/auth/challenge`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`

### 채널

- `POST /api/v1/channels`
- `GET /api/v1/channels`
- `GET /api/v1/channels/:id`
- `PUT /api/v1/channels/:id`
- `DELETE /api/v1/channels/:id`
- `POST /api/v1/channels/join`
- `POST /api/v1/channels/:id/invite`
- `GET /api/v1/channels/:id/members`
- `DELETE /api/v1/channels/:id/members/:userId`
- `POST /api/v1/dm/:userId`

### 메시지

- `GET /api/v1/channels/:id/messages`
- `POST /api/v1/channels/:id/messages`
- `PUT /api/v1/messages/:id`
- `DELETE /api/v1/messages/:id`
- `POST /api/v1/messages/:id/reactions`
- `DELETE /api/v1/messages/:id/reactions/:emoji`

### 미디어

- `POST /api/v1/media/upload`
- `GET /api/v1/media/*objectPath`

## 6. 인증 흐름

### Challenge 기반 인증

로그인은 password가 아니라 challenge 서명 방식이다.

1. 클라이언트가 username으로 challenge를 요청한다.
2. 서버는 Redis에 5분 TTL로 저장한다.
3. 클라이언트는 개인키로 challenge를 서명한다.
4. 서버는 저장된 challenge와 서명을 검증한다.
5. JWT access/refresh token을 발급한다.

### JWT Claims

Claims에는 대략 아래 정보가 들어간다.

- `user_id`
- `device_id`
- `server_id`
- 표준 만료 정보

### 로그아웃

로그아웃은 refresh token을 Redis blacklist에 올리는 방식이다.

## 7. WebSocket

### 엔드포인트

- `GET /api/v1/ws?token={jwt}`

### 허브 구조

허브는 다음 정보를 관리한다.

- `clients`: client id 기준 연결 맵
- `channels`: channel id별 구독 클라이언트 집합
- `userConns`: user id별 다중 디바이스 연결
- `broadcast`: 채널 이벤트 브로드캐스트 큐
- Redis pub/sub 연결

### 클라이언트 이벤트

- `MESSAGE_SEND`
- `MESSAGE_EDIT`
- `MESSAGE_DELETE`
- `TYPING_START`
- `TYPING_STOP`
- `REACTION_ADD`
- `REACTION_REMOVE`
- `CHANNEL_JOIN`
- `CHANNEL_LEAVE`
- `PRESENCE_UPDATE`
- `READ_ACK`

### 서버 이벤트

- `MESSAGE_NEW`
- `MESSAGE_UPDATED`
- `MESSAGE_DELETED`
- `TYPING`
- `REACTION`
- `PRESENCE`
- `MEMBER_JOINED`
- `MEMBER_LEFT`
- `ERROR`

## 8. 데이터베이스 테이블

주요 테이블:

- `users`
- `channels`
- `channel_members`
- `messages`
- `reactions`
- `devices`
- `federated_servers`

### 메시지 저장 정책

- soft delete 지원
- edit 시 `is_edited` 갱신
- content는 암호화된 payload를 저장하는 전제를 둔다
- reaction은 별도 테이블로 저장

## 9. Presence

Redis를 사용한다.

- 유저 상태: `online`, `away`, `dnd`
- typing 상태는 channel + user 조합으로 단기 TTL 저장
- 멀티 디바이스 환경을 고려해 user 기준 online 추적

## 10. 미디어

MinIO 래퍼를 사용한다.

허용 MIME 타입:

- `image/jpeg`
- `image/png`
- `image/gif`
- `image/webp`
- `video/mp4`
- `application/pdf`

제한:

- 이미지 최대 10MB
- 일반 파일 최대 50MB

경로 규칙:

```text
{userID}/{year}/{month}/{uuid}.{ext}
```

## 11. Federation

외부 서버 노출 엔드포인트:

- `POST /_federation/v1/events`
- `GET /_federation/v1/users/:username`
- `GET /_federation/v1/server-key`
- `GET /_federation/v1/.well-known`

현재는 federation 수신 이벤트의 서명 검증과 기본 payload 검증은 되지만, 완전한 cross-server persistence까지는 아직 아니다.

## 12. 알려진 기술 부채

- 인증 키 모델 문서와 구현 간 차이
- federation replay protection 미구현
- device별 세밀한 키 관리 미완성
- 테스트 코드 부족
- OpenAPI 스펙 부재

## 13. 추천 다음 작업

1. `go test ./...`와 `go build ./...`를 통과시키는 CI 고도화
2. federation 수신 이벤트를 내부 message/channel 서비스와 연결
3. device/session 관리 API 확장
4. 관리자용 관측성과 운영 메트릭 추가
