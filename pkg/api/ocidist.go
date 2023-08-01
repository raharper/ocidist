package api

// dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/bloodorangeio/reggie"
	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type OCIDistRepo struct {
	url *url.URL
	// TODO add client
}

func (odr *OCIDistRepo) Type() OCIRepoType {
	return OCIDistRepoType
}

func NewOCIDistRepo(url *url.URL) (*OCIDistRepo, error) {
	return &OCIDistRepo{url: url}, nil
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
	return odr.url.Path
}

func (odr *OCIDistRepo) GetRepoTagList() (*dspec.TagList, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithInsecureSkipTLSVerify(true), // skip TLS verification
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

func (odr *OCIDistRepo) GetOCIManifest(tag string) (*ispec.Manifest, error) {
	url := odr.BasePath()
	repoPath := odr.RepoPath()

	client, err := reggie.NewClient(url,
		reggie.WithInsecureSkipTLSVerify(true), // skip TLS verification
	)

	req := client.NewRequest(
		reggie.GET, "/v2/<name>/manifest/<digest>",
		reggie.WithName(repoPath),
		reggie.WithDigest(tag))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var manifest ispec.Manifest
	if err := json.Unmarshal([]byte(resp.Body()), &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (odr *OCIDistRepo) GetRepositories() ([]string, error) {
	url := odr.BasePath()

	client, err := reggie.NewClient(url,
		reggie.WithInsecureSkipTLSVerify(true), // skip TLS verification
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
