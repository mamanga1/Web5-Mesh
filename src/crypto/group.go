// Copyright (C) 2026 Fernando Martin Lopez. All Rights Reserved.
// SPDX-License-Identifier: AGPL-3.0-only WITH Commons-Clause-1.0
//
// This file is part of Web5-Mesh — sovereign network kernel prototype (Fase 1).
// Use of this source code is governed by the AGPLv3 + Commons Clause
// license that can be found in the LICENSE file at the root of this repo.
//
// Commercial use, SaaS deployment, or resale without a commercial license
// agreement is strictly prohibited. Contact the author for licensing.

package crypto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// FIX: respetar $XION_HOME igual que acl.go y alias.go.
// Antes usaba ".xion/groups.json" hardcodeado relativo al directorio
// de ejecución, ignorando XION_HOME — rompía en Android/mobile.
func getGroupsFile() string {
	if home := os.Getenv("XION_HOME"); home != "" {
		return filepath.Join(home, "groups.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".xion", "groups.json")
	}
	return filepath.Join(home, ".xion", "groups.json")
}

type Group struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
	Admin   string   `json:"admin"`
	Created string   `json:"created"`
}

type GroupStore struct {
	Groups map[string]*Group `json:"groups"`
	mu     sync.RWMutex
}

var groupCache *GroupStore
var groupOnce sync.Once

func LoadGroups() (*GroupStore, error) {
	groupOnce.Do(func() {
		groupCache = &GroupStore{
			Groups: make(map[string]*Group),
		}
	})

	path := getGroupsFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return groupCache, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, groupCache); err != nil {
		return nil, err
	}
	if groupCache.Groups == nil {
		groupCache.Groups = make(map[string]*Group)
	}
	return groupCache, nil
}

func SaveGroups(store *GroupStore) error {
	path := getGroupsFile()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func CreateGroup(alias, name, adminDID string) error {
	store, err := LoadGroups()
	if err != nil {
		return err
	}
	store.mu.Lock()
	store.Groups[alias] = &Group{
		Name:    name,
		Members: []string{adminDID},
		Admin:   adminDID,
		Created: "2026-07-01",
	}
	store.mu.Unlock()
	return SaveGroups(store)
}

func AddMember(alias, did string) error {
	store, err := LoadGroups()
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	group, exists := store.Groups[alias]
	if !exists {
		return nil
	}

	for _, m := range group.Members {
		if m == did {
			return nil
		}
	}

	group.Members = append(group.Members, did)
	return SaveGroups(store)
}

func RemoveMember(alias, did string) error {
	store, err := LoadGroups()
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	group, exists := store.Groups[alias]
	if !exists {
		return nil
	}

	newMembers := []string{}
	for _, m := range group.Members {
		if m != did {
			newMembers = append(newMembers, m)
		}
	}
	group.Members = newMembers
	return SaveGroups(store)
}

func GetGroup(alias string) (*Group, bool) {
	store, err := LoadGroups()
	if err != nil {
		return nil, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()

	group, exists := store.Groups[alias]
	return group, exists
}

func SaveGroupDirect(alias string, group *Group) error {
	store, err := LoadGroups()
	if err != nil {
		return err
	}
	store.mu.Lock()
	store.Groups[alias] = group
	store.mu.Unlock()
	return SaveGroups(store)
}

func DeleteGroup(alias string) error {
	store, err := LoadGroups()
	if err != nil {
		return err
	}
	store.mu.Lock()
	delete(store.Groups, alias)
	store.mu.Unlock()
	return SaveGroups(store)
}
