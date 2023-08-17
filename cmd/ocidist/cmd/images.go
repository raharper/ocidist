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
	"fmt"
	"strings"

	"github.com/raharper/ocidist/pkg/api"
	"github.com/raharper/ocidist/pkg/types"

	"github.com/spf13/cobra"
)

// imagesCmd represents the images command
var imagesCmd = &cobra.Command{
	Use:   "images [URL]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Print image versions at URL",
	Long: `
$ ocidist images ocidist://localhost:5000/myrepo/myimage
myimage:v1
myimage:v2
`,
	RunE: doImages,
}

func doImages(cmd *cobra.Command, args []string) error {
	rawURL := args[0]

	tagsOnly, err := cmd.Flags().GetBool("tags-only")
	if err != nil {
		return err
	}

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	config := &types.OCIAPIConfig{TLSVerify: tlsVerify}
	ociApi, err := api.NewOCIAPI(rawURL, config)
	if err != nil {
		return err
	}

	var imageName string
	if !tagsOnly {
		imageName = ociApi.RepoPath()
	}

	tags, err := ociApi.GetRepoTags()
	if err != nil {
		return err
	}

	// fmt.Printf("URL=%s repo:\n", ociApi.RepoPath())
	for _, tag := range tags {
		if tagsOnly {
			fmt.Printf("%s\n", tag)
		} else {
			fmt.Printf("%s\n", strings.Join([]string{imageName, tag}, "/"))
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.PersistentFlags().BoolP("tags-only", "t", false, "print image tags only")
	imagesCmd.PersistentFlags().BoolP("tls-verify", "T", true, "toggle tls verification")
}
