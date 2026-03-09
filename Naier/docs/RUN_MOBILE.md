# Run Mobile

This project does not bundle Flutter in the repository. Run these steps on a machine with Flutter and the platform toolchain installed.

## Prerequisites

- Flutter SDK
- Android Studio or Xcode
- A running backend on `http://localhost:8080`

Check the toolchain:

```powershell
flutter doctor
```

Install dependencies:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\mobile"
flutter pub get
flutter analyze
```

## Default network behavior

The mobile app now chooses its default API base URL by platform:

- Android emulator: `http://10.0.2.2:8080/api/v1`
- iOS simulator: `http://localhost:8080/api/v1`

The WebSocket URL is derived automatically from the API base URL.

You can override both at launch time:

```powershell
flutter run --dart-define=MESH_API_BASE_URL=http://192.168.0.10:8080/api/v1
```

```powershell
flutter run --dart-define=MESH_API_BASE_URL=http://192.168.0.10:8080/api/v1 --dart-define=MESH_WS_BASE_URL=ws://192.168.0.10:8080/ws
```

## Run

List devices:

```powershell
flutter devices
```

Run on the selected device:

```powershell
flutter run -d <device_id>
```

## Runtime smoke checklist

1. Register a new account
   On invite-only beta servers, enter a valid invite code during registration.
2. Log in with the challenge flow
3. Open a channel and send a message
4. Background and resume the app
5. Confirm reconnect sync catches up
6. Open settings and verify the device list
7. Test backup export/import once implemented in the mobile UI
