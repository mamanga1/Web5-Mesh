package crypto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const aliasesFile = ".xion/aliases.json"

type AliasStore struct {
	Aliases map[string]string `json:"aliases"` // alias → DID
	mu      sync.RWMutex
}

var aliasCache *AliasStore
var aliasOnce sync.Once

// selfDID guarda el DID del nodo local para mostrarlo como "yo"
var selfDID string
var selfDIDMu sync.RWMutex

// SetSelfDID registra el DID propio del nodo (llamar al iniciar shell)
func SetSelfDID(did string) {
	selfDIDMu.Lock()
	selfDID = did
	selfDIDMu.Unlock()
}

// GetSelfDID retorna el DID propio
func GetSelfDID() string {
	selfDIDMu.RLock()
	defer selfDIDMu.RUnlock()
	return selfDID
}

func getAliasesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, aliasesFile)
}

func LoadAliases() (*AliasStore, error) {
	aliasOnce.Do(func() {
		aliasCache = &AliasStore{
			Aliases: make(map[string]string),
		}
	})

	path := getAliasesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return aliasCache, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, aliasCache); err != nil {
		return nil, err
	}
	return aliasCache, nil
}

func SaveAliases(store *AliasStore) error {
	path := getAliasesPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func AddAlias(alias, did string) error {
	store, err := LoadAliases()
	if err != nil {
		return err
	}
	store.mu.Lock()
	store.Aliases[strings.ToLower(alias)] = did
	store.mu.Unlock()
	return SaveAliases(store)
}

func RemoveAlias(alias string) error {
	store, err := LoadAliases()
	if err != nil {
		return err
	}
	store.mu.Lock()
	delete(store.Aliases, strings.ToLower(alias))
	store.mu.Unlock()
	return SaveAliases(store)
}

// ResolveNode convierte alias → DID. Si ya es DID, lo devuelve tal cual.
func ResolveNode(nameOrDID string) (string, bool) {
	if strings.HasPrefix(nameOrDID, "did:") {
		return nameOrDID, true
	}

	store, err := LoadAliases()
	if err != nil {
		return nameOrDID, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()

	if did, exists := store.Aliases[strings.ToLower(nameOrDID)]; exists {
		return did, true
	}
	return nameOrDID, false
}

// ResolveDID convierte DID → alias (para mostrar en pantalla).
// Si no encuentra alias, devuelve el DID truncado.
func ResolveDID(did string) string {
	// 1. Si es el propio nodo
	selfDIDMu.RLock()
	currentSelf := selfDID
	selfDIDMu.RUnlock()
	if currentSelf != "" && currentSelf == did {
		return "yo"
	}

	// 2. Buscar en aliases locales
	store, err := LoadAliases()
	if err == nil {
		store.mu.RLock()
		for alias, d := range store.Aliases {
			if d == did {
				store.mu.RUnlock()
				return alias
			}
		}
		store.mu.RUnlock()
	}

	// 3. Si no encuentra nada, devolver DID truncado
	if len(did) > 20 {
		return did[:10] + "..." + did[len(did)-6:]
	}
	return did
}
