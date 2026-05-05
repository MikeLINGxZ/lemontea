package plugins

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type credentialStore struct {
	keyPath  string
	dataPath string
}

type encryptedCredentialPayload struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func newCredentialStore(stateDir string) *credentialStore {
	return &credentialStore{
		keyPath:  filepath.Join(stateDir, "credentials.key"),
		dataPath: filepath.Join(stateDir, "credentials.json.enc"),
	}
}

func (s *credentialStore) Get(key string) (string, bool, error) {
	items, err := s.load()
	if err != nil {
		return "", false, err
	}
	value, ok := items[key]
	return value, ok, nil
}

func (s *credentialStore) Set(key, value string) error {
	items, err := s.load()
	if err != nil {
		return err
	}
	items[key] = value
	return s.save(items)
}

func (s *credentialStore) Delete(key string) error {
	items, err := s.load()
	if err != nil {
		return err
	}
	delete(items, key)
	return s.save(items)
}

func (s *credentialStore) load() (map[string]string, error) {
	if _, err := os.Stat(s.dataPath); err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	raw, err := os.ReadFile(s.dataPath)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]string{}, nil
	}

	var payload encryptedCredentialPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	key, err := s.ensureKey()
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	if len(plaintext) == 0 {
		return map[string]string{}, nil
	}
	var items map[string]string
	if err := json.Unmarshal(plaintext, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = map[string]string{}
	}
	return items, nil
}

func (s *credentialStore) save(items map[string]string) error {
	if items == nil {
		items = map[string]string{}
	}
	key, err := s.ensureKey()
	if err != nil {
		return err
	}
	plaintext, err := json.Marshal(items)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	payload := encryptedCredentialPayload{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(s.dataPath, raw, 0600)
}

func (s *credentialStore) ensureKey() ([]byte, error) {
	if raw, err := os.ReadFile(s.keyPath); err == nil {
		decoded, decodeErr := base64.StdEncoding.DecodeString(string(raw))
		if decodeErr != nil {
			return nil, decodeErr
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("invalid credential store key length")
		}
		return decoded, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0600); err != nil {
		return nil, err
	}
	return key, nil
}
