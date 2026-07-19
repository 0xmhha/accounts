# mobile — gomobile bindings

`package mobile` is a [gomobile](https://pkg.go.dev/golang.org/x/mobile)-friendly
facade over the SDK. Its API uses only binding-safe types (string / []byte /
int64 / error / bound pointers), so it compiles to native Android and iOS
libraries.

The Go logic here is unit-tested (`go test ./mobile/`). Generating the native
artifacts requires the mobile toolchains (Android SDK/NDK, Xcode) and
`gomobile`, which are environment-specific and not run in CI here.

## Verification status

- **Android AAR — built and verified.** `gomobile bind -target=android` produces
  a 14 MB `.aar` with native `libgojni.so` for all four ABIs (armeabi-v7a,
  arm64-v8a, x86, x86_64) plus Kotlin/Java bindings exposing the API below
  (confirmed via `javap`: `Mobile.generateAccount`, `Account.addressHex`, etc.).
- **iOS XCFramework — build path is the same** but was blocked on the build
  machine by a broken Xcode simulator plugin (`IDESimulatorFoundation` failed to
  load). This is an Xcode installation issue, not an SDK/gomobile one; fix with
  `xcodebuild -runFirstLaunch` / reinstalling Xcode components, then re-run.

## One-time setup

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
gomobile init
# gomobile bind needs the bind package resolvable at build time. It is NOT kept
# in the SDK go.mod (to avoid burdening library consumers), so add it before a
# bind and revert afterward:
go get golang.org/x/mobile/bind
```

## Android (AAR)

```bash
export ANDROID_HOME="$HOME/Library/Android/sdk"
export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/<version>"
gomobile bind -target=android -androidapi 21 \
  -o accounts.aar github.com/0xmhha/accounts/mobile
# then: git checkout go.mod go.sum   # drop the temporary x/mobile requirement
```

Import `accounts.aar` in Android Studio; call e.g. `Mobile.generateAccount()`.

## iOS (XCFramework)

```bash
gomobile bind -target=ios \
  -o Accounts.xcframework github.com/0xmhha/accounts/mobile
```

Add `Accounts.xcframework` to Xcode; call e.g. `MobileGenerateAccount()`.

## API (selected)

| Function / method | Purpose |
|-------------------|---------|
| `GenerateAccount()`, `DeriveAccount(mnemonic, passphrase, index)` | create / HD-derive an account |
| `AccountFromPrivateKeyHex`, `AccountFromKeystore` | import |
| `Account.AddressHex()`, `Account.PrivateKeyHex()` | read |
| `Account.SignHashHex`, `Account.SignPersonal` | sign |
| `Account.ToKeystore(password)` | encrypt at rest |
| `SignDynamicFeeTransfer(...)` | build+sign a 0x02 transfer → raw hex to submit |
| `NewMnemonic(bits)`, `Keccak256Hex(data)` | utilities |
