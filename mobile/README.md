# mobile — gomobile bindings

`package mobile` is a [gomobile](https://pkg.go.dev/golang.org/x/mobile)-friendly
facade over the SDK. Its API uses only binding-safe types (string / []byte /
int64 / error / bound pointers), so it compiles to native Android and iOS
libraries.

The Go logic here is unit-tested (`go test ./mobile/`). Generating and running
the native artifacts requires the mobile toolchains (Android SDK/NDK, Xcode) and
`gomobile`, which are environment-specific and not run in CI here.

## One-time setup

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
gomobile init
```

## Android (AAR)

```bash
gomobile bind -target=android -androidapi 21 \
  -o accounts.aar github.com/0xmhha/accounts/mobile
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
