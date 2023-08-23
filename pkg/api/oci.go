package api

// dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"

	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"

	"github.com/opencontainers/umoci"
)

type OCIDirRepo struct {
	url    *url.URL
	config *OCIAPIConfig
}

func NewOCIDirRepo(url *url.URL, config *OCIAPIConfig) (*OCIDirRepo, error) {
	return &OCIDirRepo{url: url, config: config}, nil
}

func (odr *OCIDirRepo) Type() OCIRepoType {
	return OCIDirRepoType
}

func (odr *OCIDirRepo) OCIDir() string {
	// oci://home/ubuntu/build/oci:img:v2.31
	//       |  ||                  ^    ^ /
	//      /   |`-----.    .-------|----|'
	//     (host)      (path)       |    |
	//                           (name)(tag)
	// ociDir = host + path - (name:tag)
	toks := strings.Split(odr.url.Path, ":")
	return filepath.Join(odr.url.Host, toks[0])
}

func (odr *OCIDirRepo) RepoPath() string {
	toks := strings.Split(odr.url.Path, ":")
	return toks[0]
}

func (odr *OCIDirRepo) ImageName() string {
	toks := strings.Split(odr.url.Path, ":")
	if len(toks) == 2 || len(toks) == 3 {
		// /tmp/oci:image
		// /tmp/oci:image:tag
		//   0       1     2
		return toks[1]
	}
	return "" // this URL does not have the expected number of fields
}

func (odr *OCIDirRepo) SourceURL() string {
	return odr.url.String()
}

func (odr *OCIDirRepo) RepoTag() string {
	toks := strings.Split(odr.url.Path, ":")
	if len(toks) == 3 {
		return toks[2]
	}
	return "" // this URL does not have the expected number of fields
}

func (odr *OCIDirRepo) GetRepoTagList() (*dspec.TagList, error) {
	tagList := dspec.TagList{Tags: []string{}}

	// if URI is pointing to an image, no RepoTags are represent
	image := odr.ImageName()
	if len(image) > 0 {
		return &tagList, nil
	}

	tagList.Name = filepath.Base(odr.OCIDir())
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

func (odr *OCIDirRepo) GetManifest() (*ispec.Manifest, []byte, error) {
	ociDir := odr.OCIDir()
	image := odr.ImageName()
	tag := odr.RepoTag()
	imgRef := image
	if len(tag) > 0 {
		imgRef = fmt.Sprintf("%s:%s", image, tag)
	}

	log.WithFields(log.Fields{
		"ociDir": ociDir,
		"image":  image,
		"tag":    tag,
		"imgRef": imgRef,
	}).Debug("OCIDIr.GetManifest opening OCI layout")

	oci, err := umoci.OpenLayout(ociDir)
	if err != nil {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Failed to open OCI Layout at directory %q: %s", ociDir, err)
	}

	log.WithFields(log.Fields{
		"ociDir": ociDir,
		"imgRef": imgRef,
	}).Debug("OCIDir.GetManifest resolving image references")
	descriptorPaths, err := oci.ResolveReference(context.Background(), imgRef)
	if err != nil {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Failed to ResolveReference for OCI image '%s' in OCI Layout at directory %q: %s", imgRef, ociDir, err)
	}

	if len(descriptorPaths) != 1 {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Bad descriptor for OCI image '%s' in OCI Layout at directory %q: %s", imgRef, ociDir, err)
	}

	blob, err := oci.FromDescriptor(context.Background(), descriptorPaths[0].Descriptor())
	if err != nil {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Failed to parse referenced blob for descriptor '%s' for OCI tag '%s' in OCI Layout at directory %q: %s", descriptorPaths[0].Descriptor(), tag, ociDir, err)
	}

	defer blob.Close()

	if blob.Descriptor.MediaType != ispec.MediaTypeImageManifest {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Descriptor does not point to a manifest: '%s' for OCI tag '%s'", blob.Descriptor.MediaType, tag)
	}

	reader, err := oci.GetBlob(context.Background(), blob.Descriptor.Digest)
	if err != nil {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Failed to read OCI Manifest blob '%s' for OCI tag '%s' from OCI Layout at directory %q: %s", blob.Descriptor.Digest, tag, ociDir, err)

	}
	defer reader.Close()

	manifestBytes, err := io.ReadAll(reader)
	if err != nil {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Failed to read OCI Manifest blob '%s' for OCI tag '%s' from OCI Layout at directory %q: %s", blob.Descriptor.Digest, tag, ociDir, err)
	}

	var manifest ispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return &ispec.Manifest{}, []byte{}, fmt.Errorf("Failed to unmarshal OCI Manifest blob '%s' for OCI tag '%s' from OCI Layout at directory %q: %s", blob.Descriptor.Digest, tag, ociDir, err)
	}
	return &manifest, manifestBytes, nil
}

func (odr *OCIDirRepo) GetImage(config *ispec.Descriptor) (*ispec.Image, error) {
	ociDir := odr.OCIDir()
	oci, err := umoci.OpenLayout(ociDir)
	if err != nil {
		return &ispec.Image{}, fmt.Errorf("Failed to open OCI Layout at directory %q: %s", ociDir, err)
	}

	configBlob, err := oci.FromDescriptor(context.Background(), *config)
	if err != nil {
		return &ispec.Image{}, err
	}

	if configBlob.Descriptor.MediaType != ispec.MediaTypeImageConfig {
		return &ispec.Image{}, fmt.Errorf("bad image config type: %s", configBlob.Descriptor.MediaType)
	}

	img := configBlob.Data.(ispec.Image)
	return &img, nil
}

func (odr *OCIDirRepo) GetReferrers(image *ispec.Descriptor) (*ispec.Index, error) {
	ociDir := odr.OCIDir()
	oci, err := umoci.OpenLayout(ociDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to open OCI Layout at directory %q: %s", ociDir, err)
	}

	ociIndex, err := oci.GetIndex(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Failed to get index from OCI Layout at directory %q: %s", ociDir, err)
	}

	refs := ispec.Index{
		MediaType: ispec.MediaTypeImageIndex,
	}
	for _, indexManifest := range ociIndex.Manifests {
		if indexManifest.MediaType == ispec.MediaTypeImageManifest && indexManifest.Digest != image.Digest {

			// get the blob @ manifest.Digest
			// we can't use oci since it doesn't yet support "subject" descriptors
			blob, err := odr.GetBlob(&indexManifest)
			if err != nil {
				return nil, fmt.Errorf("Failed to read index manifest blob: %s", err)
			}

			var refManifest ispec.Manifest
			if err := json.Unmarshal(blob, &refManifest); err != nil {
				return nil, fmt.Errorf("Failed to unmarshal index manifest blob into manifest: %s", err)
			}

			if refManifest.Subject.Digest == image.Digest {
				match := ispec.Descriptor{
					ArtifactType: refManifest.ArtifactType,
					MediaType:    indexManifest.MediaType,
					Digest:       indexManifest.Digest,
					Size:         indexManifest.Size,
				}
				refs.Manifests = append(refs.Manifests, match)
			}
		}
	}

	return &refs, nil
}

func (odr *OCIDirRepo) GetBlob(layer *ispec.Descriptor) ([]byte, error) {
	ociDir := odr.OCIDir()

	// fmt.Printf("GetBlob(%s)\n", layer.Digest.String())
	algo, digest, ok := strings.Cut(layer.Digest.String(), ":")
	if !ok {
		return []byte{}, fmt.Errorf("Failed to split layer digest '%s' into algo and hash", layer.Digest.Encoded())
	}

	blobPath := filepath.Join(ociDir, "blobs", algo, digest)

	blobBytes, err := ioutil.ReadFile(blobPath)
	if err != nil {
		return []byte{}, fmt.Errorf("Failed to read OCI layer blob @ %q: %s", blobPath, err)
	}

	return blobBytes, nil
}

func (odr *OCIDirRepo) BlobHead(layer *ispec.Descriptor) error {
	return fmt.Errorf("Not implemented yet")
}

func (odr *OCIDirRepo) GetRepositories() ([]string, error) {
	return []string{odr.OCIDir()}, nil
}

func (odr *OCIDirRepo) PutBlob(layer *ispec.Descriptor, blob []byte) error {
	return fmt.Errorf("Not implemented yet")
}

func (odr *OCIDirRepo) PutManifest(manifest *ispec.Manifest) error {
	return fmt.Errorf("Not implemented yet")
}

func (odr *OCIDirRepo) PutArtifact(aritfactName, artifactType string, artifactBlob []byte) error {
	return fmt.Errorf("Not implemented yet")
}
