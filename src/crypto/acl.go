package crypto

import (
	"encoding/hex"
	"encoding/json"
	"os"
)

const ACLFile = "acl.json"

type PeerInfo struct {
	PubKeyEd string `json:"pubkey_ed"`
	PubKeyX  string `json:"pubkey_x"`
}

type ACL struct {
	Peers map[string]PeerInfo `json:"peers"`
}

func LoadACL() (*ACL, error) {
	acl := &ACL{Peers: make(map[string]PeerInfo)}
	data, err := os.ReadFile(ACLFile)
	if err != nil {
		if os.IsNotExist(err) {
			return acl, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, acl); err != nil {
		return nil, err
	}
	return acl, nil
}

func (a *ACL) IsAllowed(did string) bool {
	_, exists := a.Peers[did]
	return exists
}

func (a *ACL) GetPeerKeys(did string) (pubEd []byte, pubX []byte, err error) {
	peer, exists := a.Peers[did]
	if !exists {
		return nil, nil, os.ErrNotExist
	}

	pubEd, err = hex.DecodeString(peer.PubKeyEd)
	if err != nil {
		return nil, nil, err
	}

	pubX, err = hex.DecodeString(peer.PubKeyX)
	if err != nil {
		return nil, nil, err
	}

	return pubEd, pubX, nil
}

func (a *ACL) Save() error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ACLFile, data, 0644)
}

// AddPeer agrega un peer a la ACL
func AddPeer(did, pubKeyEd, pubKeyX string) error {
	acl, err := LoadACL()
	if err != nil {
		return err
	}
	acl.Peers[did] = PeerInfo{
		PubKeyEd: pubKeyEd,
		PubKeyX:  pubKeyX,
	}
	return acl.Save()
}

// RemovePeer elimina un peer de la ACL
func RemovePeer(did string) error {
	acl, err := LoadACL()
	if err != nil {
		return err
	}
	if _, exists := acl.Peers[did]; !exists {
		return nil
	}
	delete(acl.Peers, did)
	return acl.Save()
}

// ClearACL limpia toda la ACL
func ClearACL() error {
	acl := &ACL{Peers: make(map[string]PeerInfo)}
	return acl.Save()
}
