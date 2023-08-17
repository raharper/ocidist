package types

import (
	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type OCIRepoType string

const (
	OCIDirRepoType  OCIRepoType = "oci"
	OCIDistRepoType OCIRepoType = "ocidist"
)

type OCIAPI interface {
	Type() OCIRepoType

	GetRepoTags() ([]string, error)
	GetRepositories() ([]string, error)

	GetRepoTagList() (*dspec.TagList, error)
	GetManifest() (*ispec.Manifest, []byte, error)
	GetImage(*ispec.Descriptor) (*ispec.Image, error)

	ImageName() string
	SourceURL() string
	RepoPath() string
	RepoTag() string
}

type OCIAPIConfig struct {
	TLSVerify bool
}
