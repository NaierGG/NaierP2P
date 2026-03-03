# 배포 및 운영 문서

## 1. 개요

이 문서는 `infra/`와 `.github/workflows/`에 들어간 운영 배포 구성을 설명한다.

포함 범위:

- 운영용 Docker Compose
- Nginx reverse proxy
- MinIO
- PostgreSQL backup sidecar
- Fly.io 배포
- GitHub Actions CI/CD

## 2. 운영용 Docker Compose

파일:

- `infra/docker-compose.yml`

구성 서비스:

- `nginx`
- `backend-1`
- `backend-2`
- `backend-3`
- `postgres`
- `postgres-backup`
- `redis`
- `minio`
- `certbot`

### 백엔드 3개 인스턴스를 둔 이유

- WebSocket과 API 부하 분산
- 단일 인스턴스 장애 완화
- Redis pub/sub 기반 멀티 노드 전파 실험

### 운영 전 주의할 값

반드시 실제값으로 바꿔야 하는 항목:

- `POSTGRES_PASSWORD`
- `MESH_AUTH_JWT_SECRET`
- `MINIO_ROOT_USER`
- `MINIO_ROOT_PASSWORD`
- `MESH_FEDERATION_SERVER_DOMAIN`
- `MESH_FEDERATION_SERVER_PUBLIC_KEY`
- `MESH_FEDERATION_SERVER_PRIVATE_KEY`

## 3. Nginx

파일:

- `infra/nginx/nginx.conf`

### 역할

- HTTP -> HTTPS 리다이렉트
- `/api/` 프록시
- `/ws` WebSocket 프록시
- `/_federation/` 프록시
- gzip 압축
- API rate limit
- WebSocket connection limit
- HSTS, CSP, X-Frame-Options 등 보안 헤더

### 인증서 경로

현재 설정은 아래 경로를 사용한다.

```text
/etc/letsencrypt/live/naier.example.com/fullchain.pem
/etc/letsencrypt/live/naier.example.com/privkey.pem
```

실제 도메인으로 반드시 수정해야 한다.

## 4. Fly.io

파일:

- `infra/fly.toml`

핵심 설정:

- 앱 이름: `naier-api`
- primary region: `nrt`
- `release_command = "/app/migrate up"`
- health check: `/health`

### 배포 방식

1. 컨테이너가 빌드된다.
2. 새 릴리스 전에 `/app/migrate up`가 실행된다.
3. 정상일 경우 새 버전이 올라간다.

## 5. GitHub Actions

파일:

- `.github/workflows/deploy.yml`

### PR 시

- backend: `go mod download`, `go test ./...`
- web: `npm ci`, `npm run build`
- mobile: `flutter pub get`, `flutter analyze`

### main push 시

- Docker build
- Fly secrets 동기화
- Fly deploy
- migration status 확인

## 6. 필요한 GitHub Secrets

- `FLY_API_TOKEN`
- `MESH_AUTH_JWT_SECRET`
- `MESH_DATABASE_POSTGRES_DSN`
- `MESH_DATABASE_REDIS_ADDR`
- `MESH_DATABASE_REDIS_PASSWORD`
- `MESH_MEDIA_MINIO_ENDPOINT`
- `MESH_MEDIA_MINIO_BUCKET`
- `MESH_MEDIA_MINIO_ACCESS_KEY`
- `MESH_MEDIA_MINIO_SECRET_KEY`
- `MESH_FEDERATION_SERVER_DOMAIN`
- `MESH_FEDERATION_SERVER_PUBLIC_KEY`
- `MESH_FEDERATION_SERVER_PRIVATE_KEY`

## 7. 운영 체크리스트

### 배포 전

- DNS 설정
- TLS 인증서 경로 확인
- DB 백업 경로 확인
- MinIO 버킷 생성 확인
- federation 공개키 등록 확인

### 배포 후

- `/health` 확인
- WebSocket 연결 확인
- 파일 업로드 확인
- Redis pub/sub 브로드캐스트 확인
- migration status 확인

## 8. 현재 한계

- 실제 운영 비밀값 주입 전 상태
- Nginx 실구동 검증 미완료
- Compose 전체 실검증 미완료
- Fly 실제 릴리스 검증 미완료

즉, 인프라 정의는 들어가 있지만 운영 검증은 따로 해야 한다.

## 9. 추천 운영 보강

1. Prometheus와 Grafana 추가
2. PostgreSQL managed service 사용 검토
3. Redis persistence 및 auth 강화
4. MinIO 버킷 lifecycle 정책 추가
5. Sentry 또는 OpenTelemetry 추가
