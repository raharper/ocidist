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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

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

var sociPutCmd = &cobra.Command{
	Use:   "put <SOCI bundle>",
	Short: "publish SOCI bundle to a registry",
	Long: `
$ jq < mySvc.soci
{
 "install": {<install JSON>},
 "pubkeycert": {<public cert text>},
 "signature: {
    "encoding": "base64",
	"data": "<base64(signature of install JSON)",
 }
}
$ ocidist soci put mySvc.soci ocidist://localhost:5000/product/services/svc:v1.2
`,
	RunE:    runSociPut,
	PreRunE: doBeforeRunCmd,
}

var sociBundleCmd = &cobra.Command{
	Use:   "bundle <soci name> --install <f> --sign-key <f> --pub-key <f>",
	Short: "build a SOCI bundle",
	Long: `
$ ocidist soci bundle mysvc --install svc.json --sign-key privkey.pem --pub-key cert.pem
wrote mysvc.soci
$ jq < mysvc.soci
{
 "install": {<install JSON>},
 "pubkeycert": {<public cert text>},
 "signature: {
    "encoding": "base64",
	"data": "<base64(signature of install JSON)",
 }
}
`,
	RunE:    runSociBundle,
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

func runSociPut(cmd *cobra.Command, args []string) error {
	sociBundle := args[0]
	if sociBundle == "" {
		return fmt.Errorf("Missing soci bundle argument")
	}
	rawURL := args[1]
	if rawURL == "" {
		return fmt.Errorf("Missing URL argument")
	}

	tlsVerify, err := cmd.Flags().GetBool("tls-verify")
	if err != nil {
		return err
	}

	sociBytes, err := ioutil.ReadFile(sociBundle)
	if err != nil {
		return fmt.Errorf("Failed to read SOCI bundle file %q: %s", sociBundle, err)
	}

	var sociArtifacts api.SOCIArtifacts
	if err := json.Unmarshal(sociBytes, &sociArtifacts); err != nil {
		return fmt.Errorf("Failed to unmarshal SOCI bundle %q: %s", sociBundle, err)
	}

	config := &api.OCIAPIConfig{TLSVerify: tlsVerify}
	ociApi, err := api.NewOCIAPI(rawURL, config)
	if err != nil {
		return err
	}

	// push install artifact
	aType, err := api.SOCIArtifactType("atomix", api.SOCIArtifactInstall)
	if err != nil {
		return err
	}
	if err := ociApi.PutArtifact("install.json", aType, []byte(sociArtifacts.Install)); err != nil {
		return fmt.Errorf("failed to PUT artifact '%s' to url '%s': %s", api.SOCIArtifactInstall, rawURL, err)
	}
	// push pubkey artifact
	aType, err = api.SOCIArtifactType("atomix", api.SOCIArtifactPubKeyCrt)
	if err != nil {
		return err
	}
	if err := ociApi.PutArtifact("pubkeycrt.pem", aType, []byte(sociArtifacts.PubKeyCrt)); err != nil {
		return fmt.Errorf("failed to PUT artifact '%s' to url '%s': %s", api.SOCIArtifactPubKeyCrt, rawURL, err)
	}

	sigBlob, err := sociArtifacts.SignatureBlob()
	if err != nil {
		return fmt.Errorf("failed to get signature blob: %s", err)
	}

	aType, err = api.SOCIArtifactType("atomix", api.SOCIArtifactSignature)
	if err != nil {
		return err
	}
	if err := ociApi.PutArtifact("install.json.signature", aType, sigBlob); err != nil {
		return fmt.Errorf("failed to PUT artifact '%s' to url '%s': %s", api.SOCIArtifactSignature, rawURL, err)
	}

	/*
		sociRef, err := api.NewSOCIRef(ociApi)
		if err != nil {
			return err
		}

		if err := sociRef.Verify(); err != nil {
			return fmt.Errorf("failed to verify SOCI: %s", err)
		}
	*/

	return nil
}

func runSociBundle(cmd *cobra.Command, args []string) error {
	sociName := args[0]
	if len(sociName) == 0 {
		return fmt.Errorf("Missing soci name")
	}

	installFile := cmd.Flag("install-file").Value.String()
	pubKeyFile := cmd.Flag("pub-key").Value.String()
	signKeyFile := cmd.Flag("sign-key").Value.String()

	fmt.Printf("Generating SOCI bundle '%s.soci' with\n Install: %s\n PubKeyCrt: %s\n SignKey: %s\n", sociName, installFile, pubKeyFile, signKeyFile)

	/*
		1. use openssl dgst to sign install json capture signature
		2. base64 encode signature
		3. populate SOCIRef structure with install.json, pubkeycert, and signature
		4. write out JSON Unmarshall of SOCIRef to <output.bundle> file
	*/

	installBytes, err := ioutil.ReadFile(installFile)
	if err != nil {
		return fmt.Errorf("Failed to read install file %q: %s", installFile, err)
	}

	pubKeyBytes, err := ioutil.ReadFile(pubKeyFile)
	if err != nil {
		return fmt.Errorf("Failed to read pub-key file %q: %s", pubKeyFile, err)
	}

	// generate the signature
	sigFile, err := ioutil.TempFile("", "install-signature-XXX")
	if err != nil {
		return fmt.Errorf("Failed to create temp file for signature: %s", err)
	}
	defer os.Remove(sigFile.Name())

	fmt.Printf("Generating signature of install %q with privkey %q\n", installFile, signKeyFile)
	signArgs := []string{"openssl", "dgst", "-sha256", "-sign", signKeyFile, "-out", sigFile.Name(), installFile}
	signCmd := exec.Command(signArgs[0], signArgs[1:]...)
	_, err = signCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to sign install file %q with command '%s': %s", installFile, strings.Join(signArgs, ", "), err)
	}

	sigBytes, err := ioutil.ReadFile(sigFile.Name())
	if err != nil {
		return fmt.Errorf("Failed to read generated signature of install file %q: %s", sigFile.Name(), err)
	}

	sociArtifact, err := api.NewSOCIArtifacts(installBytes, pubKeyBytes, sigBytes)
	if err != nil {
		return fmt.Errorf("Failed to create a new SOCI Artifacts object: %s", err)
	}
	sociBytes, err := json.MarshalIndent(sociArtifact, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal SOCI Aritifacts: %s", err)
	}

	artifactsFile := fmt.Sprintf("%s.soci", sociName)
	if err := ioutil.WriteFile(artifactsFile, sociBytes, 0644); err != nil {
		return fmt.Errorf("Failed to write SOCI Artifacts to file %q: %s", artifactsFile, err)
	}

	fmt.Printf("Wrote SOCI Bundle: %s\n", artifactsFile)
	return nil
}

func init() {
	sociInspectCmd.PersistentFlags().StringP("ca-file", "c", "", "verify soci cert is issued from specified CA and still valid")

	sociBundleCmd.PersistentFlags().StringP("install-file", "i", "", "specify path to artifact vnd.machine.install file")
	sociBundleCmd.MarkPersistentFlagRequired("install-file")
	sociBundleCmd.PersistentFlags().StringP("pub-key", "p", "", "specify path to artifact vnd.machine.pubkeycrt file")
	sociBundleCmd.MarkPersistentFlagRequired("pub-key")
	sociBundleCmd.PersistentFlags().StringP("sign-key", "s", "", "specify path to private signing key file")
	sociBundleCmd.MarkPersistentFlagRequired("sign-key")

	sociCmd.PersistentFlags().BoolP("tls-verify", "T", true, "toggle tls verification")
	sociCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug output")

	sociCmd.AddCommand(sociInspectCmd)
	sociCmd.AddCommand(sociGetCmd)
	sociCmd.AddCommand(sociPutCmd)
	sociCmd.AddCommand(sociBundleCmd)
	rootCmd.AddCommand(sociCmd)
}
