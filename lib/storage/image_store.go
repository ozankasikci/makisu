//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/uber/makisu/lib/docker/image"
)

// ImageStore contains a manifeststore, a layertarstore, and a sandbox dir.
type ImageStore struct {
	RootDir    string
	SandboxDir string
	Manifests  *ManifestStore
	Layers     *LayerTarStore
}

// NewImageStore creates a new ImageStore.
func NewImageStore(rootDir string) (*ImageStore, error) {
	sandboxParent := filepath.Join(rootDir, "sandbox")
	if err := os.MkdirAll(sandboxParent, 0755); err != nil {
		return nil, fmt.Errorf("init sandbox parent dir: %s", err)
	}
	sandboxDir, err := ioutil.TempDir(sandboxParent, "sandbox")
	if err != nil {
		return nil, fmt.Errorf("init sandbox dir: %s", err)
	}

	m, err := NewManifestStore(rootDir)
	if err != nil {
		return nil, fmt.Errorf("init manifest store: %s", err)
	}
	l, err := NewLayerTarStore(rootDir)
	if err != nil {
		return nil, fmt.Errorf("init layer store: %s", err)
	}

	return &ImageStore{
		RootDir:    rootDir,
		SandboxDir: sandboxDir,
		Manifests:  m,
		Layers:     l,
	}, nil
}

// CleanupSandbox removes sandbox dir. This should be done after every build.
func CleanupSandbox(rootDir string) error {
	sandboxParent := filepath.Join(rootDir, "sandbox")
	if err := os.RemoveAll(sandboxParent); err != nil {
		return fmt.Errorf("remove sandbox parent %s: %s", sandboxParent, err)
	}
	return nil
}

func (store *ImageStore) SaveManifest(
	distManifest image.DistributionManifest, imageName image.Name) error {

	distManifestJSON, err := json.Marshal(distManifest)
	if err != nil {
		return fmt.Errorf("marshal manifest to JSON: %s", err)
	}

	distManifestFile, err := ioutil.TempFile(store.SandboxDir, "")
	if err != nil {
		return fmt.Errorf("create tmp manifest file: %s", err)
	}
	if _, err := distManifestFile.Write(distManifestJSON); err != nil {
		return fmt.Errorf("write manifest file: %s", err)
	}
	if err := distManifestFile.Close(); err != nil {
		return fmt.Errorf("close manifest file: %s", err)
	}

	distManifestPath := distManifestFile.Name()
	if err := store.Manifests.LinkStoreFileFrom(
		imageName.GetRepository(), imageName.GetTag(), distManifestPath); err != nil && !os.IsExist(err) {

		return fmt.Errorf("commit replica manifest to store: %s", err)
	}

	return nil
}
