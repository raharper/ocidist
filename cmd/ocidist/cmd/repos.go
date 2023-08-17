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

	"github.com/raharper/ocidist/pkg/api"
	"github.com/raharper/ocidist/pkg/types"

	"github.com/spf13/cobra"
)

// reposCmd represents the repos command
var reposCmd = &cobra.Command{
	Use:   "repos [URL]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Print a list of repositories available at URL",
	Long: `
$ ocidist repos ocidist://localhost:5000
`,
	RunE: doRepos,
}

func doRepos(cmd *cobra.Command, args []string) error {
	rawURL := args[0]

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	config := &types.OCIAPIConfig{TLSVerify: tlsVerify}
	ociApi, err := api.NewOCIAPI(rawURL, config)
	if err != nil {
		return err
	}

	repos, err := ociApi.GetRepositories()
	if err != nil {
		return err
	}

	for _, repo := range repos {
		fmt.Printf(" %s\n", repo)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(reposCmd)
	reposCmd.PersistentFlags().BoolP("tls-verify", "T", true, "toggle tls verification")
}
