package api

// dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/opencontainers/umoci"
)

type OCIDirRepo struct {
	url *url.URL
}

func NewOCIDirRepo(url *url.URL) (*OCIDirRepo, error) {
	return &OCIDirRepo{url: url}, nil
}

func (odr *OCIDirRepo) Type() OCIRepoType {
	return OCIDirRepoType
}

func (odr *OCIDirRepo) OCIDir() string {
	// oci://home/ubuntu/build/oci
	//       |  ||               |
	//      /   |`-----.    .----'
	//     (host)      (path)
	return filepath.Join(odr.url.Host, odr.url.Path)
}

func (odr *OCIDirRepo) RepoPath() string {
	return filepath.Base(odr.OCIDir())
}

func (odr *OCIDirRepo) GetRepoTagList() (*dspec.TagList, error) {
	tagList := dspec.TagList{
		Name: filepath.Base(odr.OCIDir()),
	}

	tags, err := odr.GetRepoTags()
	if err != nil {
		return nil, fmt.Errorf("Failed to get repo tags: %s", err)
	}

	tagList.Tags = tags
	return &tagList, nil
}

// oci:///path/to/oci/dir
func (odr *OCIDirRepo) GetRepoTags() ([]string, error) {
	ociDir := odr.OCIDir()
	oci, err := umoci.OpenLayout(ociDir)
	if err != nil {
		return []string{}, fmt.Errorf("Failed to open OCI Layout at directory %q: %s", ociDir, err)
	}

	refs, err := oci.ListReferences(context.Background())
	if err != nil {
		return []string{}, fmt.Errorf("Failed to get OCI References from layout at directory %q: %s", ociDir, err)
	}

	return refs, nil
}

func (odr *OCIDirRepo) GetOCIManifest(tag string) (*ispec.Manifest, error) {
	ociDir := odr.OCIDir()
	oci, err := umoci.OpenLayout(ociDir)
	if err != nil {
		return &ispec.Manifest{}, fmt.Errorf("Failed to open OCI Layout at directory %q: %s", ociDir, err)
	}

	descriptorPaths, err := oci.ResolveReference(context.Background(), tag)
	if err != nil {
		return &ispec.Manifest{}, fmt.Errorf("Failed to ResolveReference for OCI tag '%s' in OCI Layout at directory %q: %s", tag, ociDir, err)
	}

	if len(descriptorPaths) != 1 {
		return &ispec.Manifest{}, fmt.Errorf("Bad descriptor for OCI tag '%s' in OCI Layout at directory %q: %s", tag, ociDir, err)
	}

	blob, err := oci.FromDescriptor(context.Background(), descriptorPaths[0].Descriptor())
	if err != nil {
		return &ispec.Manifest{}, fmt.Errorf("Failed to parse referenced blob for descriptor '%s' for OCI tag '%s' in OCI Layout at directory %q: %s", descriptorPaths[0].Descriptor(), tag, ociDir, err)
	}

	defer blob.Close()

	if blob.Descriptor.MediaType != ispec.MediaTypeImageManifest {
		return &ispec.Manifest{}, fmt.Errorf("Descriptor does not point to a manifest: '%s' for OCI tag '%s'", blob.Descriptor.MediaType, tag)
	}

	reader, err := oci.GetBlob(context.Background(), blob.Descriptor.Digest)
	if err != nil {
		return &ispec.Manifest{}, fmt.Errorf("Failed to read OCI Manifest blob '%s' for OCI tag '%s' from OCI Layout at directory %q: %s", blob.Descriptor.Digest, tag, ociDir, err)

	}
	defer reader.Close()

	manifestBytes, err := io.ReadAll(reader)
	if err != nil {
		return &ispec.Manifest{}, fmt.Errorf("Failed to read OCI Manifest blob '%s' for OCI tag '%s' from OCI Layout at directory %q: %s", blob.Descriptor.Digest, tag, ociDir, err)
	}

	var manifest ispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return &ispec.Manifest{}, fmt.Errorf("Failed to unmarshal OCI Manifest blob '%s' for OCI tag '%s' from OCI Layout at directory %q: %s", blob.Descriptor.Digest, tag, ociDir, err)
	}
	return &manifest, nil
}

func (odr *OCIDirRepo) GetRepositories() ([]string, error) {
	return []string{odr.OCIDir()}, nil
}
