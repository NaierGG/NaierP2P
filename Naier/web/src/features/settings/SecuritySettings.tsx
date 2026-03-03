import { useEffect, useMemo, useState } from "react";
import { Copy, Key, Shield, Smartphone, Trash2 } from "lucide-react";

import { clearAuth, setKeyPair } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import {
  approveDevice,
  createPendingDevice,
  fetchDevices,
  revokeDevice,
} from "@/features/settings/settingsApi";
import { generateKeyPair, generateSigningKeyPair } from "@/shared/lib/crypto";
import { useEncryption } from "@/shared/hooks/useEncryption";
import { keyStore } from "@/shared/lib/keystore";
import type { Device } from "@/shared/types";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

type DeviceView = Device & { current?: boolean };

interface PairingPayload {
  device_signing_key: string;
  device_exchange_key: string;
  device_name: string;
  platform: "web";
}

function buildDefaultPairingPayload(index: number): PairingPayload {
  return {
    device_signing_key: `pending-device-signing-${index}`,
    device_exchange_key: `pending-device-exchange-${index}`,
    device_name: `Pending Web Device ${index}`,
    platform: "web",
  };
}

export default function SecuritySettings() {
  const dispatch = useAppDispatch();
  const accessToken = useAppSelector((state) => state.auth.accessToken);
  const keyPair = useAppSelector((state) => state.auth.keyPair);
  const [devices, setDevices] = useState<DeviceView[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [importValue, setImportValue] = useState("");
  const [pairingValue, setPairingValue] = useState("");
  const [pendingDeviceId, setPendingDeviceId] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { clearStoredKeys } = useEncryption();

  const publicKeyPreview = useMemo(
    () => keyPair?.identity.signing.publicKey ?? "신원 서명 키가 로드되지 않았습니다",
    [keyPair?.identity.signing.publicKey]
  );

  async function loadDevices() {
    if (!accessToken) return;

    setLoadingDevices(true);
    try {
      const response = await fetchDevices(accessToken);
      setDevices(response.devices);
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "디바이스 로드에 실패했습니다.");
    } finally {
      setLoadingDevices(false);
    }
  }

  useEffect(() => {
    void loadDevices();
  }, [accessToken]);

  async function exportKeyBundle() {
    if (!keyPair) return;
    await navigator.clipboard.writeText(JSON.stringify(keyPair, null, 2));
    setNotice("키 번들이 복사되었습니다.");
  }

  async function exportPublicKey() {
    if (!keyPair) return;
    await navigator.clipboard.writeText(keyPair.identity.signing.publicKey);
    setNotice("신원 서명 키가 복사되었습니다.");
  }

  async function importKeyBundle() {
    const trimmed = importValue.trim();
    if (!trimmed) {
      setNotice("내보낸 키 번들을 먼저 붙여넣기하세요.");
      return;
    }

    try {
      const parsed = JSON.parse(trimmed) as NonNullable<typeof keyPair>;
      await keyStore.saveKeyBundle(parsed);
      dispatch(setKeyPair(parsed));
      setImportValue("");
      setNotice("키 번들을 가져왔습니다.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "키 번들 가져오기에 실패했습니다.");
    }
  }

  async function resetIdentity() {
    await clearStoredKeys();
    dispatch(clearAuth());
  }

  async function handleRevokeDevice(deviceId: string) {
    if (!accessToken) return;

    try {
      await revokeDevice(accessToken, deviceId);
      await loadDevices();
      setNotice("디바이스가 해제되었습니다.");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "디바이스 해제에 실패했습니다.");
    }
  }

  async function generatePairingPayload() {
    const [signing, exchange] = await Promise.all([
      generateSigningKeyPair(),
      generateKeyPair(),
    ]);

    const payload: PairingPayload = {
      device_signing_key: signing.publicKey,
      device_exchange_key: exchange.publicKey,
      device_name: "New web browser",
      platform: "web",
    };

    setPairingValue(JSON.stringify(payload, null, 2));
    setNotice("페어링 페이로드가 생성되었습니다.");
    setError(null);
  }

  function parsePairingPayload(): PairingPayload {
    const trimmed = pairingValue.trim();
    if (!trimmed) throw new Error("먼저 디바이스 페어링 페이로드를 붙여넣거나 생성하세요.");

    const parsed = JSON.parse(trimmed) as Partial<PairingPayload>;
    if (!parsed.device_signing_key || !parsed.device_exchange_key || !parsed.device_name || !parsed.platform) {
      throw new Error("페어링 페이로드에 필수 필드가 누락되었습니다.");
    }

    return {
      device_signing_key: parsed.device_signing_key,
      device_exchange_key: parsed.device_exchange_key,
      device_name: parsed.device_name,
      platform: parsed.platform,
    };
  }

  async function handleRegisterPendingDevice() {
    if (!accessToken) return;

    try {
      const payload = parsePairingPayload();
      const response = await createPendingDevice(accessToken, payload);
      setPendingDeviceId(response.device.id);
      await loadDevices();
      setNotice(`대기 디바이스 등록: ${response.device.id}`);
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "대기 디바이스 등록에 실패했습니다.");
    }
  }

  async function handleApprovePendingDevice() {
    if (!accessToken || !pendingDeviceId) {
      setNotice("먼저 대기 디바이스를 등록하세요.");
      return;
    }

    try {
      await approveDevice(accessToken, pendingDeviceId);
      await loadDevices();
      setNotice(`디바이스 승인: ${pendingDeviceId}`);
      setPendingDeviceId(null);
      setPairingValue("");
      setError(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "디바이스 승인에 실패했습니다.");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 키 관리 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-4 w-4 text-primary" />
            신원 및 디바이스 키
          </CardTitle>
          <CardDescription>
            브라우저에 장기 신원 키와 현재 신뢰 디바이스 키가 저장되어 있습니다.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">신원 서명 키</span>
            <div className="rounded-xl bg-muted p-3 font-mono text-xs text-muted-foreground break-all">
              {publicKeyPreview}
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => void exportPublicKey()}>
              <Copy className="mr-2 h-3.5 w-3.5" />
              신원 키 복사
            </Button>
            <Button size="sm" variant="secondary" onClick={() => void exportKeyBundle()}>
              키 번들 복사
            </Button>
          </div>

          <Separator />

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">키 번들 가져오기</span>
            <Textarea
              onChange={(e) => setImportValue(e.target.value)}
              placeholder="내보낸 키 번들 JSON을 붙여넣기"
              rows={6}
              value={importValue}
              className="font-mono text-xs"
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" onClick={() => void importKeyBundle()}>
              가져오기
            </Button>
            <Button size="sm" variant="destructive" onClick={() => void resetIdentity()}>
              <Trash2 className="mr-2 h-3.5 w-3.5" />
              로컬 신원 초기화
            </Button>
          </div>

          {notice && <p className="text-sm text-muted-foreground">{notice}</p>}
          {error && <p className="text-sm text-destructive">{error}</p>}
        </CardContent>
      </Card>

      {/* 디바이스 페어링 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-primary" />
            다른 디바이스 승인
          </CardTitle>
          <CardDescription>
            새 디바이스가 공개키가 포함된 페이로드를 생성하면, 이미 신뢰된 디바이스에서 등록 및 승인합니다.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <Button size="sm" variant="secondary" onClick={() => void generatePairingPayload()}>
            샘플 페이로드 생성
          </Button>

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">디바이스 페어링 페이로드</span>
            <Textarea
              onChange={(e) => setPairingValue(e.target.value)}
              placeholder={JSON.stringify(buildDefaultPairingPayload(1), null, 2)}
              rows={6}
              value={pairingValue}
              className="font-mono text-xs"
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" onClick={() => void handleRegisterPendingDevice()}>
              대기 디바이스 등록
            </Button>
            <Button size="sm" variant="secondary" onClick={() => void handleApprovePendingDevice()}>
              대기 디바이스 승인
            </Button>
            {pendingDeviceId && (
              <Badge variant="secondary">대기 ID: {pendingDeviceId}</Badge>
            )}
          </div>
        </CardContent>
      </Card>

      {/* 디바이스 목록 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Smartphone className="h-4 w-4 text-primary" />
            신뢰 디바이스
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loadingDevices && <p className="text-sm text-muted-foreground">로드 중...</p>}
          <div className="flex flex-col gap-2">
            {devices.map((device) => (
              <div
                key={device.id}
                className="flex items-center justify-between gap-4 rounded-xl bg-muted p-3"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium">{device.device_name || "이름 없는 디바이스"}</p>
                  <p className="text-xs text-muted-foreground">
                    {device.platform}
                    {device.last_seen && ` · ${new Date(device.last_seen).toLocaleString("ko-KR")}`}
                    {device.current && " · 현재"}
                    {device.trusted === false && " · 승인 대기"}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant={device.current ? "secondary" : "outline"}
                  disabled={device.current}
                  onClick={() => void handleRevokeDevice(device.id)}
                >
                  {device.current ? "현재" : "해제"}
                </Button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
