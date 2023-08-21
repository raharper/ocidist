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
	"encoding/json"
	"fmt"

	"github.com/raharper/ocidist/pkg/api"
	"github.com/spf13/cobra"
)

var sociCmd = &cobra.Command{
	Use:   "soci <cmd>",
	Short: "manage Signed OCI (soci) images",
}

var sociInspectCmd = &cobra.Command{
	Use:     "inspect <URI>",
	Short:   "inspect a Signed OCI (soci) image",
	RunE:    runSociInspect,
	PreRunE: doBeforeRunCmd,
}

var sociGetCmd = &cobra.Command{
	Use:   "get <URI>",
	Short: "get a Signed OCI (soci) image",
	Long: `
$ ocidist soci get ocidist://localhost:5000/product/services/svc:v1.2
{
 "install": {},
 "pubkeycrt": {},
 "signature": {},
}
`,
	RunE:    runSociGet,
	PreRunE: doBeforeRunCmd,
}

func runSociInspect(cmd *cobra.Command, args []string) error {
	rawURL := args[0]
	if rawURL == "" {
		return fmt.Errorf("Missing URL argument")
	}
	cmd.SilenceUsage = true

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	config := &api.OCIAPIConfig{TLSVerify: tlsVerify}
	ociApi, err := api.NewOCIAPI(rawURL, config)
	if err != nil {
		return err
	}

	sociRef, err := api.NewSOCIRef(ociApi)
	if err != nil {
		return err
	}

	info := api.SOCIInfo{
		Ref:          ociApi.ImageName(), // sociRef.ImageName()? or sociRef.API.ImageName()?
		Digest:       sociRef.Digest.String(),
		InstallLayer: fmt.Sprintf("%s: %s", sociRef.Install.Digest.String(), sociRef.Manifest.ArtifactType),
		Referrers: []string{
			fmt.Sprintf("%s: %s", sociRef.Signature.Digest.String(), sociRef.Signature.ArtifactType),
			fmt.Sprintf("%s: %s", sociRef.PubKeyCrt.Digest.String(), sociRef.PubKeyCrt.ArtifactType),
		},
	}

	_, verifyInfo, _ := sociRef.Verify(cmd.Flag("ca-file").Value.String())
	info.Verification = verifyInfo

	content, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", content)
	return nil
}

func runSociGet(cmd *cobra.Command, args []string) error {
	rawURL := args[0]
	if rawURL == "" {
		return fmt.Errorf("Missing URL argument")
	}
	cmd.SilenceUsage = true

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	config := &api.OCIAPIConfig{TLSVerify: tlsVerify}
	ociApi, err := api.NewOCIAPI(rawURL, config)
	if err != nil {
		return err
	}

	sociRef, err := api.NewSOCIRef(ociApi)
	if err != nil {
		return err
	}

	sociArtifacts, err := sociRef.GetArtifacts()
	if err != nil {
		return err
	}
	content, err := json.MarshalIndent(sociArtifacts, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", content)

	return nil
}

func init() {
	sociInspectCmd.PersistentFlags().StringP("ca-file", "c", "", "verify soci cert is issued from specified CA and still valid")

	sociCmd.PersistentFlags().BoolP("tls-verify", "T", true, "toggle tls verification")
	sociCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug output")

	sociCmd.AddCommand(sociInspectCmd)
	sociCmd.AddCommand(sociGetCmd)
	rootCmd.AddCommand(sociCmd)
}
