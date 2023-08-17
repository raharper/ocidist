package api

import (
	"fmt"
	"net/url"
	"os"

	"github.com/raharper/ocidist/pkg/image"
	"github.com/raharper/ocidist/pkg/noci"
	"github.com/raharper/ocidist/pkg/types"
)

func NewOCIAPI(rawURL string, config *types.OCIAPIConfig) (types.OCIAPI, error) {

	url, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url '%s': %s", rawURL, err)
	}

	switch url.Scheme {
	case "docker", "https", "http", "ocidist":
		return NewOCIDistRepo(url, config)
	case "oci":
		return NewOCIDirRepo(url, config)
	case "noci":
		return noci.NewRepo(url, config)
	}

	return nil, fmt.Errorf("Unknown URL scheme '%s' in url '%s'", url.Scheme, rawURL)
}

func ImageCopy(src, dest string, opts image.ImageCopyOpts) error {

	if opts.Src == "" {
		srcURL, err := url.Parse(src)
		if err != nil {
			return fmt.Errorf("Failed to parse source url '%s': %s", src, err)
		}

		opts.Src = srcURL.String()
		switch srcURL.Scheme {
		case "ocidist", "docker", "oci":
		default:
			return fmt.Errorf("source url has unsupported scheme '%s', must be 'docker', 'ocidist', or 'oci'", srcURL.Scheme)
		}
	}

	if opts.Dest == "" {
		destURL, err := url.Parse(dest)
		if err != nil {
			return fmt.Errorf("Failed to parse dest url '%s': %s", dest, err)
		}

		opts.Dest = destURL.String()
		switch destURL.Scheme {
		case "ocidist":
			opts.Dest = fmt.Sprintf("docker://%s", destURL.Path)
		case "docker", "oci":
		default:
			return fmt.Errorf("destination url has unsupported scheme '%s', must be 'docker' or 'oci'", destURL.Scheme)
		}
	}

	if opts.Progress == nil {
		opts.Progress = os.Stdout
	}

	if err := image.ImageCopy(opts); err != nil {
		return fmt.Errorf("failed to copy image '%s' to '%s': %s", opts.Src, opts.Dest, err)
	}

	return nil
}
