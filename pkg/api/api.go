package api

import (
	"fmt"
	"net/url"

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

	SourceURL() string
	RepoPath() string
	RepoTag() string
}

type OCIAPIConfig struct {
	TLSVerify bool
}

func NewOCIAPI(rawURL string, config *OCIAPIConfig) (OCIAPI, error) {

	url, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url '%s': %s", rawURL, err)
	}

	switch url.Scheme {
	case "docker", "https", "http", "ocidist":
		return NewOCIDistRepo(url, config)
	case "oci":
		return NewOCIDirRepo(url, config)
	}

	return nil, fmt.Errorf("Unknown URL scheme '%s' in url '%s'", url.Scheme, rawURL)
}
