/*
Copyright © 2023 Ryan Harper <rharper@woxford.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raharper/ocidist/pkg/api"

	"github.com/containers/image/v5/types"
	digest "github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect <URL>",
	Args:  cobra.MinimumNArgs(1),
	Short: "print information about an OCI Image at URL",
	Long: `
$ ocidist inspect ocidist://localhost:5000/myrepo/myimage:v2.1
{
  "Name": "localhost:5000/myrepo/myimage",
  "Digest": "sha256:xxx",
  "RepoTags": [
    "v2.1"
  ],
  ...
}`,
	RunE:    doInspect,
	PreRunE: doBeforeRunCmd,
}

// github.com/containers/skopeo/cmd/skopeo/inspect/output.go:Output
type InspectOutput struct {
	Name          string `json:",omitempty"`
	Tag           string `json:",omitempty"`
	Digest        digest.Digest
	RepoTags      []string
	Created       *time.Time
	DockerVersion string
	Labels        map[string]string
	Architecture  string
	Os            string
	Layers        []string
	LayersData    []types.ImageInspectLayer `json:",omitempty"`
	Env           []string
}

func doInspect(cmd *cobra.Command, args []string) error {
	rawURL := args[0]

	/*
		outputConfig, err := cmd.Flags().GetBool("config")
		if err != nil {
			return err
		}

		rawOutput, err := cmd.Flags().GetBool("raw")
		if err != nil {
			return err
		}
	*/

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	apiConfig := &api.OCIAPIConfig{TLSVerify: tlsVerify}
	ociApi, err := api.NewOCIAPI(rawURL, apiConfig)
	if err != nil {
		return err
	}

	manifest, manifestBytes, err := ociApi.GetManifest()
	if err != nil {
		return err
	}

	// compute manifest digest
	hash := sha256.New()
	hash.Write(manifestBytes)
	manifestDigest := digest.Digest(fmt.Sprintf("sha256:%s", hex.EncodeToString(hash.Sum(nil))))

	img, err := ociApi.GetImage(&manifest.Config)
	if err != nil {
		return err
	}

	tagList, err := ociApi.GetRepoTagList()
	if err != nil {
		return err
	}

	var layers []string
	for _, desc := range manifest.Layers {
		layers = append(layers, string(desc.Digest))
	}

	output := InspectOutput{
		Digest:       manifestDigest,
		RepoTags:     tagList.Tags,
		Created:      img.Created,
		Labels:       img.Config.Labels,
		Architecture: img.Architecture,
		Os:           img.OS,
		Layers:       layers,
		Env:          img.Config.Env,
	}

	if ociApi.Type() == api.OCIDistRepoType {
		output.Name = ociApi.ImageName()
	}

	outputBytes, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", outputBytes)

	return nil
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug output")
	inspectCmd.PersistentFlags().BoolP("config", "c", false, "output configuration")
	inspectCmd.PersistentFlags().BoolP("tls-verify", "T", true, "toggle tls verification")
	inspectCmd.PersistentFlags().BoolP("raw", "r", false, "output raw manifest or configuration")
}
