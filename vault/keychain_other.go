//go:build !darwin

package vault

import "fmt"

// KeychainBackend is only implemented on macOS. On other platforms it returns
// an error so code depending on it fails clearly. Use FileBackend or implement
// a platform-specific Backend (e.g. libsecret on Linux, DPAPI on Windows, or a
// PKCS#11 HSM).
type KeychainBackend struct{}

var errKeychainUnsupported = fmt.Errorf("vault: OS keychain backend is only available on macOS")

// NewKeychainBackend returns a backend that errors on every operation off macOS.
func NewKeychainBackend(_ string) *KeychainBackend { return &KeychainBackend{} }

func (*KeychainBackend) Write(string, []byte) error  { return errKeychainUnsupported }
func (*KeychainBackend) Read(string) ([]byte, error) { return nil, errKeychainUnsupported }
func (*KeychainBackend) List() ([]string, error)     { return nil, errKeychainUnsupported }
func (*KeychainBackend) Delete(string) error         { return errKeychainUnsupported }
