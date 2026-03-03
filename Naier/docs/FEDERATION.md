# Federation 상세 문서

## 1. 목적

Federation은 서로 다른 서버 인스턴스가 메시지와 유저 정보를 교환하도록 만드는 계층이다. 목표는 Matrix와 유사한 방향성이지만, 현재 구현은 훨씬 단순화된 형태다.

## 2. 기본 개념

### 서버 식별자

각 서버는 도메인을 가진다.

예시:

```text
naier.example.com
chat.second.example
```

### 사용자 주소

확장 시 아래와 같은 포맷을 염두에 둔다.

```text
@username:server.domain
```

### 서버 키

각 서버는 아래를 가진다.

- federation public key
- federation private key

이 키는 서버 간 이벤트 서명과 검증에 사용된다.

## 3. Discovery

### DNS TXT

다른 서버의 공개키와 endpoint는 DNS TXT를 통해 발견한다.

조회 레코드:

```text
_naier._tcp.{domain}
```

예시 값:

```text
v=mc1;key={base64publickey};endpoint=https://naier.example.com
```

### Resolver 동작

Resolver는 아래 순서로 조회한다.

1. 메모리 캐시 확인
2. `federated_servers` DB 확인
3. DNS TXT 조회
4. 결과를 DB와 캐시에 반영

## 4. 이벤트 모델

현재 핵심 구조는 `FederatedEvent`다.

필드:

- `event_id`
- `type`
- `server_id`
- `timestamp`
- `payload`
- `signature`

### 현재 정의된 타입

- `MESSAGE_FORWARD`
- `USER_SYNC`

추후 추가 가능:

- `CHANNEL_INVITE`
- `CHANNEL_JOIN`
- `READ_ACK`
- `PRESENCE_UPDATE`

## 5. 송신 흐름

1. 대상 도메인을 resolve한다.
2. 이벤트에 `event_id`, `server_id`, `timestamp`를 채운다.
3. 이벤트 본문을 서버 private key로 서명한다.
4. `POST /_federation/v1/events`로 전송한다.

## 6. 수신 흐름

1. 이벤트 envelope 수신
2. 필수 필드 검증
3. timestamp skew 검증
4. 발신 서버 public key 조회
5. 서명 검증
6. 서버 상태 갱신
7. payload별 처리

## 7. 외부 노출 엔드포인트

### `POST /_federation/v1/events`

다른 서버의 이벤트를 수신한다.

현재 처리 수준:

- 유효성 검사
- 서명 검증
- payload 기본 검사

### `GET /_federation/v1/users/:username`

원격 서버가 해당 username의 공개 사용자 정보를 가져갈 수 있다.

### `GET /_federation/v1/server-key`

이 서버의 domain과 public key를 반환한다.

### `GET /_federation/v1/.well-known`

이 서버의 federation 메타데이터를 반환한다.

## 8. 현재 구현 범위

완료:

- resolver
- TXT record parsing
- DB cache
- event signing
- event verification
- 원격 유저 조회
- 서버 메타데이터 엔드포인트

미완료:

- 원격 메시지 영속화
- 중복 이벤트 방지
- event replay protection
- remote membership state sync
- remote media proxy 정책

## 9. 보안 고려사항

### 반드시 필요한 보강

- event id 기준 dedupe store
- nonce 또는 monotonic sequence
- 서버 키 회전 전략
- blocked federation server 정책
- request timeout, retry, circuit breaker

### 시간 기반 검증

현재 구현은 timestamp skew 허용 범위를 둔다. 하지만 이것만으로 replay 공격이 막히는 것은 아니다.

## 10. 다음 추천 작업

1. `federated_events` 같은 dedupe 테이블 추가
2. `MESSAGE_FORWARD` 수신 시 로컬 채널과 사용자 resolve
3. remote user cache와 local shadow user 모델 설계
4. federation 통합 테스트 환경 두 세트 구성
