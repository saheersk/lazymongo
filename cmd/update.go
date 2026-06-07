package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update lazymongo to the latest release",
	RunE:  runUpdate,
}

func runUpdate(_ *cobra.Command, _ []string) error {
	fmt.Println("Checking for updates…")

	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}

	current := strings.TrimPrefix(buildVersion, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	if current == "dev" {
		fmt.Printf("Current: dev build\nLatest:  v%s\n\n", latestClean)
	} else {
		fmt.Printf("Current: v%s\nLatest:  v%s\n\n", current, latestClean)
		if current == latestClean {
			fmt.Println("Already up to date.")
			return nil
		}
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}

	url := fmt.Sprintf(
		"https://github.com/saheersk/lazymongo/releases/download/v%s/lazymongo_%s_%s_%s%s",
		latestClean, latestClean, goos, goarch, ext,
	)

	fmt.Printf("Downloading v%s (%s/%s)…\n", latestClean, goos, goarch)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed (HTTP %d) — check https://github.com/saheersk/lazymongo/releases", resp.StatusCode)
	}

	newBin, err := extractFromTarGz(resp.Body)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	defer os.Remove(newBin)

	if err := os.Chmod(newBin, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	if err := swapBinary(execPath, newBin); err != nil {
		return fmt.Errorf("replace binary (try with sudo?): %w", err)
	}

	fmt.Printf("✓  Updated to v%s — restart lazymongo.\n", latestClean)
	return nil
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func fetchLatestVersion() (string, error) {
	req, err := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/saheersk/lazymongo/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API: HTTP %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no releases found")
	}
	return rel.TagName, nil
}

// extractFromTarGz pulls the lazymongo binary out of a .tar.gz stream and
// writes it to a temp file, returning the temp file path.
func extractFromTarGz(r io.Reader) (string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		base := strings.TrimPrefix(hdr.Name, "./")
		if base != "lazymongo" && base != "lazymongo.exe" {
			continue
		}

		tmp, err := os.CreateTemp("", "lazymongo-update-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
		tmp.Close()
		return tmp.Name(), nil
	}
	return "", fmt.Errorf("lazymongo binary not found in archive")
}

// swapBinary atomically replaces dest with src.
// On Windows it renames the old binary to .old first (running binaries are locked).
func swapBinary(dest, src string) error {
	if runtime.GOOS == "windows" {
		old := dest + ".old"
		_ = os.Remove(old)
		if err := os.Rename(dest, old); err != nil {
			return err
		}
	}

	if err := os.Rename(src, dest); err != nil {
		// Cross-device rename (e.g. /tmp → /usr/local/bin) — fall back to copy.
		return copyReplace(src, dest)
	}
	return nil
}

func copyReplace(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
