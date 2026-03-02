# Mesh Chat

탈중앙화 메신저 프로토타입 저장소다. 현재 저장소는 아래 3개 클라이언트와 1개 백엔드, 그리고 운영 인프라 정의를 포함한다.

- `backend/`: Go 기반 API, 인증, WebSocket, 채널, 메시지, 미디어, federation
- `web/`: React + Vite + TypeScript 웹 클라이언트
- `mobile/`: Flutter 모바일 클라이언트
- `infra/`: 운영용 Docker Compose, Nginx, Fly.io, CI/CD

## 1. 현재 상태

현재 구현된 범위는 아래와 같다.

- Phase 1: Go 백엔드 코어, 마이그레이션, 인증, WebSocket, 채널/메시지, presence, media
- Phase 2: React 웹 초기화, 암호화 레이어, WebSocket 클라이언트, 인증 플로우, 메인 채팅 UI
- Phase 3: Flutter 초기화, 모바일 채팅 UI 스캐폴드, 알림 서비스 스캐폴드
- Phase 4: federation 모델, 리졸버, 송수신 서비스, 외부 federation 엔드포인트
- Phase 5: 운영용 인프라 구성, Nginx, Fly.io 배포 설정, GitHub Actions CI/CD

아직 남아 있는 큰 작업도 있다.

- federation 수신 이벤트를 로컬 DB에 완전 영속화하는 처리
- 모바일 앱의 실제 API/WS/FCM 실연결
- 전체 통합 테스트와 런타임 검증
- 운영용 인증서, 도메인, secret 실제값 반영

## 2. 빠른 구조

```text
mesh-chat/
├── backend/
├── web/
├── mobile/
├── infra/
└── docs/
```

## 3. 문서 목록

- [시스템 아키텍처](./docs/ARCHITECTURE.md)
- [백엔드 상세 문서](./docs/BACKEND.md)
- [웹 클라이언트 상세 문서](./docs/WEB.md)
- [모바일 클라이언트 상세 문서](./docs/MOBILE.md)
- [Federation 상세 문서](./docs/FEDERATION.md)
- [배포 및 운영 문서](./docs/DEPLOYMENT.md)

## 4. 개발 환경 요약

### 백엔드

- Go `1.22`
- PostgreSQL
- Redis
- MinIO

### 웹

- Node.js `20+` 권장
- npm

### 모바일

- Flutter `3.22+`
- Dart `3.4+`

## 5. 로컬 실행 순서

### 백엔드 단독

1. `backend/docker-compose.yml`로 PostgreSQL, Redis, backend를 먼저 띄운다.
2. 필요하면 `cmd/migrate`로 마이그레이션을 적용한다.
3. `GET /health`로 상태를 확인한다.

### 웹

1. `web/`에서 `npm install`
2. `npm run build` 또는 `npm run dev`

### 모바일

1. `mobile/`에서 `flutter pub get`
2. `flutter analyze`
3. 시뮬레이터 또는 디바이스에서 실행

## 6. 중요한 주의사항

### 인증 키 모델

현재 구현은 초기 스펙 문구와 완전히 같지 않다. 원문에는 `X25519 공개키`와 로그인 서명 검증이 같이 적혀 있었는데, 실제 서명 검증은 `Ed25519` 계열 공개키가 더 자연스럽다. 현재 코드와 웹 클라이언트는 이 차이를 감안한 상태다. 이후 정식화할 때는 아래 중 하나로 정리하는 편이 좋다.

- 인증용 서명 키와 E2E 키교환 키를 분리
- 또는 하나의 명확한 identity key 체계를 문서와 코드에서 동시에 고정

### 검증 상태

이 저장소는 현재 코드 생성과 구조화는 많이 진행됐지만, 이 환경에서는 아래 검증이 완전히 끝나지 않았다.

- Go 빌드 및 실행 검증
- Docker 전체 스택 실구동
- Flutter analyze 및 런타임 검증
- federation 다중 서버 통합 테스트

## 7. 추천 읽기 순서

1. [시스템 아키텍처](./docs/ARCHITECTURE.md)
2. [백엔드 상세 문서](./docs/BACKEND.md)
3. [웹 클라이언트 상세 문서](./docs/WEB.md)
4. [모바일 클라이언트 상세 문서](./docs/MOBILE.md)
5. [Federation 상세 문서](./docs/FEDERATION.md)
6. [배포 및 운영 문서](./docs/DEPLOYMENT.md)
