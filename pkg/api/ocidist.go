package api

// dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/apex/log"
	"github.com/bloodorangeio/reggie"
	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
)

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
	case "ocidist":
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, odr.url.Host)
}

func (odr *OCIDistRepo) RepoPath() string {
	toks := strings.Split(odr.url.Path, ":")
	if len(toks) > 0 {
		return toks[0]
	}
	return odr.url.Path
}

func (odr *OCIDistRepo) RepoTag() string {
	toks := strings.Split(odr.url.Path, ":")
	if len(toks) > 0 {
		return toks[len(toks)-1]
	}
	return ""
}

func (odr *OCIDistRepo) GetRepoTagList() (*dspec.TagList, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
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

func (odr *OCIDistRepo) GetImage(config *ispec.Descriptor) (*ispec.Image, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithUserAgent(UserAgent),
		reggie.WithInsecureSkipTLSVerify(!odr.config.TLSVerify), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.GET, "/v2/<name>/blobs/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(string(config.Digest)))

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

func (odr *OCIDistRepo) GetRepositories() ([]string, error) {
	url := odr.BasePath()

	client, err := reggie.NewClient(url,
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
