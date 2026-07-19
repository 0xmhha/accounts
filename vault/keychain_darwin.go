//go:build darwin

package vault

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

// KeychainBackend stores keystores in the macOS login keychain as generic
// passwords under a service name. The stored value is the (already
// password-encrypted) keystore document, base64-encoded. A companion index
// entry tracks the set of ids so List works.
//
// Note: this shells out to the `security` CLI; the value is visible to `ps`
// while the command runs, but it is already password-encrypted. Not exercised
// in CI (it touches the real login keychain).
type KeychainBackend struct {
	service string
}

const indexAccount = "__index__"

// NewKeychainBackend returns a keychain backend under the given service name
// (e.g. "com.0xmhha.accounts").
func NewKeychainBackend(service string) *KeychainBackend {
	return &KeychainBackend{service: service}
}

func (b *KeychainBackend) security(args ...string) ([]byte, error) {
	cmd := exec.Command("security", args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("security %s: %w: %s", args[0], err, strings.TrimSpace(errBuf.String()))
	}
	return out.Bytes(), nil
}

func (b *KeychainBackend) setItem(account string, data []byte) error {
	enc := base64.StdEncoding.EncodeToString(data)
	_, err := b.security("add-generic-password", "-U", "-s", b.service, "-a", account, "-w", enc)
	return err
}

func (b *KeychainBackend) getItem(account string) ([]byte, error) {
	out, err := b.security("find-generic-password", "-s", b.service, "-a", account, "-w")
	if err != nil {
		return nil, ErrNotFound
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(string(out)))
}

func (b *KeychainBackend) delItem(account string) error {
	_, err := b.security("delete-generic-password", "-s", b.service, "-a", account)
	return err
}

func (b *KeychainBackend) readIndex() map[string]bool {
	set := map[string]bool{}
	data, err := b.getItem(indexAccount)
	if err != nil {
		return set
	}
	for _, id := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if id != "" {
			set[id] = true
		}
	}
	return set
}

func (b *KeychainBackend) writeIndex(set map[string]bool) error {
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return b.setItem(indexAccount, []byte(strings.Join(ids, "\n")))
}

func (b *KeychainBackend) Write(id string, data []byte) error {
	if err := b.setItem(id, data); err != nil {
		return err
	}
	set := b.readIndex()
	set[id] = true
	return b.writeIndex(set)
}

func (b *KeychainBackend) Read(id string) ([]byte, error) {
	return b.getItem(id)
}

func (b *KeychainBackend) Delete(id string) error {
	if err := b.delItem(id); err != nil {
		return ErrNotFound
	}
	set := b.readIndex()
	delete(set, id)
	return b.writeIndex(set)
}

func (b *KeychainBackend) List() ([]string, error) {
	set := b.readIndex()
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids, nil
}
