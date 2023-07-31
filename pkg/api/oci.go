package api

// dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
import (
	"encoding/json"

	"github.com/bloodorangeio/reggie"
	dspec "github.com/opencontainers/distribution-spec/specs-go/v1"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

func GetRepoTagList(client *reggie.Client, repoPath string) (*dspec.TagList, error) {
	log.Info("GetRepoTagList:", "repoPath", repoPath)
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

func GetRepoTags(client *reggie.Client, repoPath string) ([]string, error) {
	log.Info("GetRepoTags:", "repoPath", repoPath)
	tagList, err := GetRepoTagList(client, repoPath)
	if err != nil {
		return []string{}, err
	}

	return tagList.Tags, nil
}

func GetOCIManifest(client *reggie.Client, repoPath, tag string) (*ispec.Manifest, error) {
	log.Info("GetOCIManifest:", "repoPath", repoPath, "tag", tag)
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

func GetRepositories(url string) ([]string, error) {
	log.Info("GetRepositories:", "url", url)
	log.Infof("creating new client at url: %s", url)
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
