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
	"github.com/raharper/ocidist/pkg/api"
	"github.com/raharper/ocidist/pkg/image"

	"github.com/spf13/cobra"
)

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:   "copy <source URL> <dest URL>",
	Args:  cobra.MinimumNArgs(2),
	Short: "copy an OCI from one URL to another",
	Long: `
$ ocidist copy ocidist://localhost:5000/myrepo/myimage:v2.1 oci:///ocidir/myrepo/myimage:v2.1
...
OK
`,
	RunE: doCopy,
}

func doCopy(cmd *cobra.Command, args []string) error {
	rawSrc := args[0]
	rawDest := args[1]

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	copyOpts := image.ImageCopyOpts{
		SrcSkipTLS:  !tlsVerify,
		DestSkipTLS: !tlsVerify,
	}

	if err := api.ImageCopy(rawSrc, rawDest, copyOpts); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(copyCmd)
	copyCmd.PersistentFlags().BoolP("tls-verify", "T", true, "toggle tls verification")

}
