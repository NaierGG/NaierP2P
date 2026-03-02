# 웹 클라이언트 상세 문서

## 1. 개요

웹 클라이언트는 Vite + React 18 + TypeScript 기반이다. 목적은 다음과 같다.

- challenge 기반 로그인
- 브라우저 로컬 키 저장
- 실시간 채팅 UI
- WebSocket 재연결
- E2E 암호화 전처리

## 2. 주요 기술 스택

- React 18
- TypeScript
- React Router v6
- Redux Toolkit
- RTK Query
- Axios
- Framer Motion
- tweetnacl
- WebCrypto API
- idb

## 3. 디렉토리 구조

```text
web/src/
├── app/
│   ├── store/
│   ├── router.tsx
│   └── App.tsx
├── features/
│   ├── auth/
│   ├── channels/
│   ├── messages/
│   ├── presence/
│   └── settings/
└── shared/
    ├── hooks/
    ├── lib/
    └── types/
```

## 4. 상태 관리

### authSlice

보관 정보:

- 현재 사용자
- access token
- 인증 여부
- 메모리 상의 key pair

주의:

- private key는 서버로 보내지 않는다.
- refresh token은 API refresh 흐름과 함께 사용한다.

### channelSlice

- 채널 목록
- 활성 채널 id
- unread count
- 마지막 메시지 요약

### messageSlice

- `channelId -> messages[]`
- pagination cursor
- `hasMore`
- optimistic pending message

### presenceSlice

- `userId -> status`
- typing 상태

## 5. API 레이어

`shared/lib/api.ts`는 Axios 인스턴스를 감싼다.

역할:

- baseURL 설정
- access token 자동 첨부
- 401 발생 시 refresh 시도
- refresh 성공 후 원요청 재시도

## 6. 암호화 레이어

### 키 생성

`generateKeyPair()`는 로컬 키페어를 만든다.

### 챌린지 서명

`signChallenge()`는 로그인용 challenge를 서명한다.

### 메시지 암복호화

- `encryptMessage()`
- `decryptMessage()`

구현 포인트:

- AES-GCM 사용
- `ciphertext`와 `iv`를 별도로 보관

### 채널 키 저장

IndexedDB를 사용한다.

스토어:

- `keypairs`
- `channelkeys`

보안상 의미:

- 브라우저 저장소는 사용자의 로컬 신뢰영역으로 취급한다.
- 하지만 브라우저 자체가 완전한 HSM은 아니므로 export/import UX가 중요하다.

## 7. WebSocket 클라이언트

### 기능

- 자동 연결
- 자동 재연결
- exponential backoff
- 연결 전 큐잉
- ping 루프

### 상태

- `connecting`
- `connected`
- `disconnected`
- `reconnecting`

### 이벤트 반영

수신 이벤트는 Redux 액션으로 변환된다.

예시:

- `MESSAGE_NEW -> addMessage`
- `TYPING -> setTyping`
- `PRESENCE -> setPresence`

## 8. 인증 UI

### LoginPage

흐름:

1. username 입력
2. challenge 요청
3. keystore에서 키 로드
4. challenge 서명
5. 로그인 호출
6. 토큰 저장 및 앱 영역 진입

### KeygenFlow

3단계 온보딩:

1. 키 개념 소개
2. 키 생성
3. 백업 강제 강조

### RegisterPage

- username
- display name
- public key 제출

## 9. 채팅 UI

### Channel List

- 검색
- DM/Group/Public 필터
- 마지막 메시지 미리보기
- unread count
- 활성 채널 강조

### Message List

- 가상 스크롤
- 상단 스크롤 시 이전 페이지 로드
- 날짜 구분선
- 연속 메시지 그룹핑

### Message Bubble

- 내 메시지 / 남 메시지 스타일 분기
- text/image/file 렌더링
- 반응 표시
- hover action
- reply preview
- optimistic sending 상태

### Message Input

- 텍스트 입력
- 이모지
- 첨부
- reply/edit 상태
- typing 이벤트 전송
- 전송 직전 암호화

## 10. 현재 구현 상태와 한계

들어간 것:

- 라우팅
- 기본 레이아웃
- 인증 플로우
- 키 저장
- WebSocket 클라이언트
- 메인 채팅 UI 골격

아직 부족한 것:

- 실제 디자인 polish
- drag reorder 실연결
- 파일 업로드와 미리보기 전체 완성
- 실환경 E2E 키 모델 정리
- 브라우저 수준 테스트

## 11. 빌드

예시:

```bash
npm install
npm run build
```

현재 `build`는 성공한 상태로 정리돼 있다.
