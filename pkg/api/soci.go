package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	SOCIArtifactInstall   = "install"
	SOCIArtifactPubKeyCrt = "pubkeycrt"
	SOCIArtifactSignature = "signature"
)

func SOCIArtifactType(product, artifactType string) (string, error) {
	switch artifactType {
	case SOCIArtifactInstall, SOCIArtifactPubKeyCrt, SOCIArtifactSignature:
		return fmt.Sprintf("application/vnd.%s.%s", product, artifactType), nil
	}
	return "", fmt.Errorf("Unknown SOCI Artifact Type: '%s'", artifactType)
}

type SOCIInfo struct {
	Ref          string   `json:"ref"`
	Digest       string   `json:"digest"`
	InstallLayer string   `json:"install-layer"`
	Referrers    []string `json:"referrers"`
	Verification string   `json:"verification"`
}

type SOCIRef struct {
	API       OCIAPI
	Manifest  *ispec.Manifest  `json:"manifest"`
	Digest    digest.Digest    `json:"digest"`
	Install   ispec.Descriptor `json:"install"`
	PubKeyCrt ispec.Descriptor `json:"pubkeycrt"`
	Signature ispec.Descriptor `json:"signature"`
}

type Signature struct {
	Encoding string `json:"encoding"`
	Data     string `json:"data"`
}

type SOCIArtifacts struct {
	Install   string    `json:"install"`
	PubKeyCrt string    `json:"pubkeycrt"`
	Signature Signature `json:"signature"`
}

func (sa SOCIArtifacts) SignatureBlob() ([]byte, error) {
	if sa.Signature.Encoding != "base64" {
		return []byte{}, fmt.Errorf("Unsupported signature encoding '%s'", sa.Signature.Encoding)
	}
	sigBlob, err := base64.StdEncoding.DecodeString(sa.Signature.Data)
	if err != nil {
		return []byte{}, fmt.Errorf("Error decoding base64 signature data: %s", err)
	}

	return sigBlob, nil
}

func NewSOCIRef(ociApi OCIAPI) (SOCIRef, error) {
	rawURL := ociApi.SourceURL()

	manifest, mBytes, err := ociApi.GetManifest()
	if err != nil {
		return SOCIRef{}, fmt.Errorf("Failed getting manifest for soci %s: %s", rawURL, err)
	}

	// compute manifest digest
	hash := sha256.New()
	hash.Write(mBytes)
	manifestDigest := digest.Digest(fmt.Sprintf("sha256:%s", hex.EncodeToString(hash.Sum(nil))))

	// create a manifest descriptor
	sociManifestLayer := ispec.Descriptor{
		MediaType: ispec.MediaTypeImageManifest,
		Digest:    manifestDigest,
	}

	// confirm artifactType is atomix.install
	atxSociInst, err := SOCIArtifactType("atomix", SOCIArtifactInstall)
	if err != nil {
		return SOCIRef{}, fmt.Errorf("Failed composing SOCIArtifactType: %s", err)
	}
	if manifest.ArtifactType != atxSociInst {
		return SOCIRef{}, fmt.Errorf("%s does not point to a valid SOCI image, found artifactType '%s' expected '%s'", rawURL, manifest.ArtifactType, atxSociInst)
	}

	// confirm soci has install layer
	if len(manifest.Layers) != 1 {
		return SOCIRef{}, fmt.Errorf("SOCI Image '%s' has %d layers, expected 1", rawURL, len(manifest.Layers))
	}

	// track the blobs we want to fetch
	sociRef := SOCIRef{
		API:      ociApi,
		Manifest: manifest,
		Digest:   manifestDigest,
		Install:  manifest.Layers[0],
	}

	// ask for referrers
	referrers, err := ociApi.GetReferrers(&sociManifestLayer)
	if err != nil {
		return SOCIRef{}, fmt.Errorf("Failed getting Referrers for soci: %s", err)
	}

	// collect soci artifact refs
	atxSociCert, err := SOCIArtifactType("atomix", SOCIArtifactPubKeyCrt)
	if err != nil {
		return SOCIRef{}, fmt.Errorf("Failed composing SOCIArtifactType: %s", err)
	}
	atxSociSig, err := SOCIArtifactType("atomix", SOCIArtifactSignature)
	if err != nil {
		return SOCIRef{}, fmt.Errorf("Failed coposing SOCIArtifactType: %s", err)
	}
	for _, ref := range referrers.Manifests {
		switch ref.ArtifactType {
		case atxSociCert:
			sociRef.PubKeyCrt = ref
		case atxSociSig:
			sociRef.Signature = ref
		}
	}

	return sociRef, nil
}

func (sref SOCIRef) GetInstallBlob() ([]byte, error) {
	return sref.API.GetBlob(&sref.Install)
}

func (sref SOCIRef) GetSignatureBlob() ([]byte, error) {
	sigManifestBytes, err := sref.API.GetBlob(&sref.Signature)
	if err != nil {
		return []byte{}, fmt.Errorf("Failed to get soci signature ref blob: %s", err)
	}
	var sigManifest ispec.Manifest
	if err := json.Unmarshal(sigManifestBytes, &sigManifest); err != nil {
		return []byte{}, err
	}

	return sref.API.GetBlob(&sigManifest.Layers[0])
}

func (sref SOCIRef) GetPubKeyCrtBlob() ([]byte, error) {
	certManifestBytes, err := sref.API.GetBlob(&sref.PubKeyCrt)
	if err != nil {
		return []byte{}, err
	}
	var certManifest ispec.Manifest
	if err := json.Unmarshal(certManifestBytes, &certManifest); err != nil {
		return []byte{}, err
	}

	return sref.API.GetBlob(&certManifest.Layers[0])
}

func (sref SOCIRef) GetArtifacts() (SOCIArtifacts, error) {
	install, err := sref.GetInstallBlob()
	if err != nil {
		return SOCIArtifacts{}, err
	}

	pubkeycrt, err := sref.GetPubKeyCrtBlob()
	if err != nil {
		return SOCIArtifacts{}, err
	}

	signature, err := sref.GetSignatureBlob()
	if err != nil {
		return SOCIArtifacts{}, err
	}

	return NewSOCIArtifacts(install, pubkeycrt, signature)
}

func NewSOCIArtifacts(install, pubkeycrt, sig []byte) (SOCIArtifacts, error) {
	return SOCIArtifacts{
		Install:   string(install),
		PubKeyCrt: string(pubkeycrt),
		Signature: Signature{
			Encoding: "base64",
			Data:     base64.StdEncoding.EncodeToString(sig),
		},
	}, nil
}

func (sref SOCIRef) Verify(caFile string) (bool, string, error) {
	installBytes, err := sref.GetInstallBlob()
	if err != nil {
		return false, "", err
	}

	sigBytes, err := sref.GetSignatureBlob()
	if err != nil {
		return false, "", fmt.Errorf("Failed to get soci signature layer blob: %s", err)
	}

	certBytes, err := sref.GetPubKeyCrtBlob()
	if err != nil {
		return false, "", fmt.Errorf("failed to get soci pubkeycrt layer blob: %s", err)
	}

	workdir, err := os.MkdirTemp("", "ocidistv")
	if err != nil {
		return false, "", fmt.Errorf("failed to make temp directory: %s", err)
	}
	defer os.RemoveAll(workdir)

	iFile := filepath.Join(workdir, "install.json")
	if err := os.WriteFile(iFile, installBytes, 0o0644); err != nil {
		return false, "", fmt.Errorf("Failed to write soci install file %q: %s", iFile, err)
	}
	sFile := filepath.Join(workdir, "install.json.signed")
	if err := os.WriteFile(sFile, sigBytes, 0o0644); err != nil {
		return false, "", fmt.Errorf("Failed to write signature file %q: %s", sFile, err)
	}

	kFile := filepath.Join(workdir, "install.pubkey")
	cFile := filepath.Join(workdir, "install.pem")
	if err := os.WriteFile(cFile, certBytes, 0o0644); err != nil {
		return false, "", fmt.Errorf("Failed to write certificate file %q: %s", cFile, err)
	}

	pubKeyArgs := []string{"openssl", "x509", "-pubkey", "-in", cFile, "-out", kFile}
	sslCmd := exec.Command(pubKeyArgs[0], pubKeyArgs[1:]...)
	// fmt.Printf("Running Command: %s\n", strings.Join(pubKeyArgs, " "))
	_, err = sslCmd.CombinedOutput()
	if err != nil {
		return false, "", fmt.Errorf("Failed to extract pubkey from cert %q: %s", cFile, err)
	}
	// fmt.Printf("openssl extracted pubkey from %q to %q\n", cFile, kFile)

	_, err = os.Stat(kFile)
	if err != nil && os.IsNotExist(err) {
		return false, "", fmt.Errorf("Failed to extract pubkey from cert %q: %s", cFile, err)
	}

	if caFile != "" {
		// verify that the cert is still valid
		verifyCertArgs := []string{"openssl", "verify", "-CAfile", caFile, cFile}
		verifyCertCmd := exec.Command(verifyCertArgs[0], verifyCertArgs[1:]...)
		// fmt.Printf("Running Command: %s\n", strings.Join(verifyCertArgs, " "))
		_, err := verifyCertCmd.CombinedOutput()
		if err != nil {
			return false, fmt.Sprintf("Verification Failed: CA file '%s' cannot verify SOCI cert '%s'", caFile, cFile), err
		}
	}
	verifyArgs := []string{"openssl", "dgst", "-sha256", "-verify", kFile, "-signature", sFile, iFile}
	verifyCmd := exec.Command(verifyArgs[0], verifyArgs[1:]...)
	// fmt.Printf("Running Command: %s\n", strings.Join(verifyArgs, " "))
	output, err := verifyCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Sprintf("Verification Failed: %s", err), err
	}
	return true, strings.TrimSpace(string(output)), nil
}
