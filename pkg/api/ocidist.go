package api

// dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bloodorangeio/reggie"
	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

var ManifestV2 = specs.Versioned{
	SchemaVersion: 2,
}

const (
	UserAgent = "ocidist/0.0.1 (https://github.com/project-machine/ocidist)"
)

type OCIDistRepo struct {
	url    *url.URL
	config *OCIAPIConfig
	// TODO add client
}

func (odr *OCIDistRepo) Type() OCIRepoType {
	return OCIDistRepoType
}

func NewOCIDistRepo(url *url.URL, config *OCIAPIConfig) (*OCIDistRepo, error) {
	return &OCIDistRepo{url: url, config: config}, nil
}

func (odr *OCIDistRepo) BasePath() string {
	scheme := odr.url.Scheme
	switch odr.url.Scheme {
	case "ocidist", "docker":
		scheme = "http"
	}
	if odr.config.TLSVerify {
		scheme += "s"
	}
	return fmt.Sprintf("%s://%s", scheme, odr.url.Host)
}

func (odr *OCIDistRepo) RepoPath() string {
	path := ""
	toks := strings.Split(odr.url.Path, ":")
	if len(toks) > 0 {
		path = toks[0]
	} else {
		path = odr.url.Path
	}
	return strings.TrimLeft(path, "/")
}

func (odr *OCIDistRepo) RepoTag() string {
	toks := strings.Split(odr.url.Path, ":")
	if len(toks) > 0 {
		return toks[len(toks)-1]
	}
	return ""
}

func (odr *OCIDistRepo) SourceURL() string {
	return odr.url.String()
}

func (odr *OCIDistRepo) ImageName() string {
	return filepath.Join(odr.url.Host, odr.RepoPath())
}

func (odr *OCIDistRepo) GetRepoTagList() (*dspec.TagList, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.GET, "/v2/<name>/tags/list",
		reggie.WithName(repoPath))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var tagList dspec.TagList
	if err := json.Unmarshal([]byte(resp.Body()), &tagList); err != nil {
		return nil, err
	}
	return &tagList, nil
}

func (odr *OCIDistRepo) GetRepoTags() ([]string, error) {
	tagList, err := odr.GetRepoTagList()
	if err != nil {
		return []string{}, err
	}

	return tagList.Tags, nil
}

func (odr *OCIDistRepo) GetManifest() (*ispec.Manifest, []byte, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()
	tag := odr.RepoTag()

	log.WithFields(log.Fields{
		"url":      url,
		"repoPath": repoPath,
		"tag":      tag,
	}).Debug("OCIDist.GetManifest() creating new Client")
	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	log.WithFields(log.Fields{
		"url":      url,
		"repoPath": repoPath,
		"tag":      tag,
	}).Debug("OCIDist.GetManifest() issueing request")
	req := client.NewRequest(
		reggie.GET, "/v2/<name>/manifests/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(tag))

	log.WithFields(log.Fields{
		"req.URL": req.URL,
	}).Debug("OCIDIst.GetManifest() request URL")

	resp, err := client.Do(req)
	if err != nil {
		return nil, []byte{}, fmt.Errorf("Failed to get a response from server: %s", err)
	}
	var manifest ispec.Manifest
	manifestBytes := resp.Body()
	log.WithFields(log.Fields{
		"resp.Body": string(manifestBytes),
	}).Debug("OCIDist.GetManifest() request response body")
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, []byte{}, fmt.Errorf("Failed to unmarshal response Body: %s", err)
	}
	return &manifest, manifestBytes, nil
}

func (odr *OCIDistRepo) ManifestHead() error {
	url := odr.BasePath()
	repoPath := odr.RepoPath()
	tag := odr.RepoTag()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
		reggie.WithDefaultName(repoPath),
	)

	req := client.NewRequest(
		reggie.HEAD, "/v2/<name>/manifests/<digest>",
		reggie.WithDigest(tag))

	log.WithFields(log.Fields{
		"url":        url,
		"repoPath":   repoPath,
		"tag":        tag,
		"req.Method": req.Method,
		"req.URL":    req.URL,
	}).Debug("OCIDist.ManifestHead() creating new Request")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"resp":   resp,
		"Status": resp.Status(),
	}).Debug("OCIDist.ManifestHead() got HEAD response")

	if resp.StatusCode() != 200 {
		return fmt.Errorf("Failed to find manifest, StatusCode: %d", resp.StatusCode())
	}

	return nil
}

func (odr *OCIDistRepo) PutManifest(manifest *ispec.Manifest) error {
	log.WithFields(log.Fields{
		"manifest": manifest,
	}).Debug("OCIDist.PutManifest() called")

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("Failed to marshal manifest: %s", err)
	}

	url := odr.BasePath()
	repoPath := odr.RepoPath()
	tag := odr.RepoTag()
	ref := tag

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithDefaultName(repoPath),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	// if manifest has a subject, then PUT via sha256
	if manifest.Subject != nil {
		dgst := digest.FromBytes(manifestJSON)
		ref = dgst.String()
		log.WithFields(log.Fields{
			"digest": ref,
		}).Debug("OCIDist.PutManifest() has subject, using PUT via digest")
	}

	log.WithFields(log.Fields{
		"url":      url,
		"repoPath": repoPath,
		"ref":      ref,
	}).Debug("OCIDist.PutManifest() created new client")

	req := client.NewRequest(
		reggie.PUT, "/v2/<name>/manifests/<reference>",
		reggie.WithReference(ref)).
		SetHeader("Content-Type", ispec.MediaTypeImageManifest).
		SetBody(manifestJSON)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to PUT manifest: %s", err)
	}

	log.WithFields(log.Fields{
		"response":   resp,
		"Status":     resp.Status(),
		"StatusCode": resp.StatusCode(),
	}).Debug("OCIDist.PutManifest() got response")

	if resp.StatusCode() != 201 {
		return fmt.Errorf("Failed to PUT manifest, StatusCode: %d", resp.StatusCode())
	}

	/*
		// update referres if defined on this manifest
		if manifest.Subject != nil {
			mSub := manifest.Subject
			if mSub.MediaType != "" && mSub.Size > 0 && mSub.Digest.String() != "" {
				if err := odr.PutReferrers(manifest); err != nil {
					return fmt.Errorf("Failed PUT new referrers association with manifest: %s", err)
				}
			}
		}
	*/
	return nil
}

func (odr *OCIDistRepo) GetManifestWithDigest() (*ispec.Manifest, []byte, digest.Digest, error) {
	manifest, mBytes, err := odr.GetManifest()
	if err != nil {
		return manifest, mBytes, digest.FromString(""), fmt.Errorf("Failed to get manifest: %s", err)
	}

	digest := digest.NewDigestFromBytes("sha256", mBytes)

	return manifest, mBytes, digest, nil
}

func (odr *OCIDistRepo) GetImage(image *ispec.Descriptor) (*ispec.Image, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.GET, "/v2/<name>/blobs/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(string(image.Digest)))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var img ispec.Image
	if err := json.Unmarshal([]byte(resp.Body()), &img); err != nil {
		return nil, err
	}

	return &img, nil
}

func (odr *OCIDistRepo) GetReferrers(image *ispec.Descriptor) (*ispec.Index, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.GET, "/v2/<name>/referrers/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(string(image.Digest)))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var index ispec.Index
	if err := json.Unmarshal([]byte(resp.Body()), &index); err != nil {
		return nil, err
	}

	return &index, nil
}

/*
func (odr *OCIDistRepo) PutReferrers(manifest *ispec.Manifest) error {

}
*/

func (odr *OCIDistRepo) GetBlob(layer *ispec.Descriptor) ([]byte, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.GET, "/v2/<name>/blobs/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(string(layer.Digest)))

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	return resp.Body(), nil
}

func (odr *OCIDistRepo) BlobHead(layer *ispec.Descriptor) error {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.HEAD, "/v2/<name>/blobs/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(string(layer.Digest)))

	log.WithFields(log.Fields{
		"url":      url,
		"repoPath": repoPath,
		"layer":    layer,
		"req":      req,
	}).Debug("OCIDist.BlobHead() creating new Request")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"resp":   resp,
		"Status": resp.Status(),
	}).Debug("OCIDist.BlobHead() got HEAD response")

	if resp.StatusCode() != 200 {
		return fmt.Errorf("Failed to find blob, StatusCode: %d", resp.StatusCode())
	}

	return nil
}

func (odr *OCIDistRepo) GetRepositories() ([]string, error) {
	url := odr.BasePath()

	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(reggie.GET, "/v2/_catalog")

	resp, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}

	var repoList dspec.RepositoryList
	if err := json.Unmarshal([]byte(resp.Body()), &repoList); err != nil {
		return []string{}, err
	}
	return repoList.Repositories, nil
}

func (odr *OCIDistRepo) PutBlob(layer *ispec.Descriptor, blob []byte) error {
	log.WithFields(log.Fields{
		"layer":    layer,
		"blobSize": len(blob),
	}).Debug("OCIDist.PutBlob() called")

	// if blob already exists, skip put
	if err := odr.BlobHead(layer); err == nil {
		log.WithFields(log.Fields{
			"layer": layer,
		}).Debug("OCIDist.PutBlob() blob already exists")
		return nil
	}

	url := odr.BasePath()
	repoPath := odr.RepoPath()
	client, err := reggie.NewClient(url,
		reggie.WithDebug(odr.config.Debug),
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
		reggie.WithDefaultName(repoPath),
	)

	log.WithFields(log.Fields{
		"url":      url,
		"repoPath": repoPath,
	}).Debug("OCIDist.PutBlob() created new client")

	// get upload url
	req := client.NewRequest(reggie.POST, "/v2/<name>/blobs/uploads/")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to get upload URL: %s", err)
	}

	log.WithFields(log.Fields{
		"resp":     resp,
		"location": resp.GetRelativeLocation(),
	}).Debug("OCIDist.PutBlob() got POST response")

	// FIXME: attempt anonymous blob mount?

	// upload in one chunk
	req = client.NewRequest(reggie.PUT, resp.GetRelativeLocation()).
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("Content-Length", fmt.Sprintf("%d", layer.Size)).
		SetQueryParam("digest", layer.Digest.String()).
		SetBody(blob)

	log.WithFields(log.Fields{
		"uploadURL": resp.GetRelativeLocation(),
	}).Debug("OCIDist.PutBlob() create new PUT request")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to PUT blob: %s", err)
	}

	log.WithFields(log.Fields{
		"resp":   resp,
		"Status": resp.Status(),
	}).Debug("OCIDist.PutBlob() got PUT response")

	if resp.StatusCode() != 201 {
		return fmt.Errorf("Failed to PUT blob: %s", err)
	}
	return nil
}

func (odr *OCIDistRepo) PutArtifact(artifactName, artifactType string, artifactBlob []byte) error {
	emptyConfig := ispec.Descriptor{
		MediaType: "application/vnd.oci.empty.v1+json",
		Size:      2,
		Digest:    digest.FromBytes([]byte("{}")),
	}

	log.WithFields(log.Fields{
		"artifactName": artifactName,
		"artifactType": artifactType,
	}).Debug("OCIDist.PutArtifact() called")

	// create and upload artifact blob first
	blobs := []ispec.Descriptor{
		{
			MediaType: "application/octet-stream",
			Size:      int64(len(artifactBlob)),
			Digest:    digest.FromBytes(artifactBlob),
			Annotations: map[string]string{
				ispec.AnnotationTitle: artifactName,
			},
		},
	}

	log.WithFields(log.Fields{
		"blob.Digest": blobs[0].Digest.String(),
	}).Debug("OCIDist.PutArtifact() created blob, uploading...")

	// upload empty config
	if err := odr.PutBlob(&emptyConfig, []byte("{}")); err != nil {
		return fmt.Errorf("Failed to put empty config blob: %s", err)
	}

	// upload blob
	if err := odr.PutBlob(&blobs[0], artifactBlob); err != nil {
		return fmt.Errorf("Failed to put artifact blob: %s", err)
	}

	log.WithFields(log.Fields{
		"blob.Digest": blobs[0].Digest.String(),
	}).Debug("OCIDist.PutArtifact() upload OK")

	// create a manifest referencing the uploaded blob in its layers
	manifest := ispec.Manifest{
		MediaType:    ispec.MediaTypeImageManifest,
		ArtifactType: artifactType,
		Config:       emptyConfig,
		Layers:       blobs,
		Versioned:    ManifestV2,
	}

	// check if there is an existing manifest, if so, fetch and reference this
	// manifest on subsequent artifacts
	if err := odr.ManifestHead(); err == nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Debug("OCIDist.PutArtifact() manifest HEAD OK, getting manifest with digest")

		refManifest, refMBytes, err := odr.GetManifest()
		if err != nil {
			return fmt.Errorf("Failed to get subject manifest: %s", err)
		}
		subject := ispec.Descriptor{
			MediaType: refManifest.MediaType,
			Digest:    digest.FromBytes(refMBytes),
			Size:      int64(len(refMBytes)),
		}
		manifest.Subject = &subject
		log.WithFields(log.Fields{
			"Manifest.Subject": subject,
		}).Debug("OCIDist.PutArtifact() existing Manifest, adding Manifest.Subject reference")
	} else {
		log.Debugf("OCIDist.PutArtifact() no manifest created yet, skipping subject referrers")
	}

	log.WithFields(log.Fields{
		"manifest.Config.Digest": manifest.Config.Digest.String(),
	}).Debug("OCIDist.PutArtifact() created manifest, calling Put Manifest")

	// put the manifest pointing to artifact
	if err := odr.PutManifest(&manifest); err != nil {
		return fmt.Errorf("Failed to put artifact manifest: %s", err)
	}

	return nil
}
