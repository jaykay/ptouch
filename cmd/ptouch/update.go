package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	githubOwner   = "jaykay"
	githubRepo    = "ptouch"
	checkInterval = 24 * time.Hour
)

type updateCache struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// updateResult is sent from the background check goroutine.
type updateResult struct {
	Latest string
	Err    error
}

// updateCh receives the result of the background update check.
var updateCh chan updateResult

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ptouch to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		latest, err := fetchLatestVersion()
		if err != nil {
			return fmt.Errorf("check for updates: %w", err)
		}
		writeCache(latest)

		if !isNewer(latest, version) {
			fmt.Fprintf(out, "Already up to date (%s)\n", version)
			return nil
		}

		fmt.Fprintf(out, "Updating %s → %s ...\n", version, latest)

		if gobin, err := exec.LookPath("go"); err == nil {
			pkg := fmt.Sprintf("github.com/%s/%s/cmd/ptouch@%s", githubOwner, githubRepo, latest)
			c := exec.Command(gobin, "install", pkg)
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			if err := c.Run(); err != nil {
				return fmt.Errorf("go install failed: %w", err)
			}
			fmt.Fprintf(out, "Updated to %s\n", latest)
			return nil
		}

		fmt.Fprintf(out, "Go is not installed. Download the latest release manually:\n")
		fmt.Fprintf(out, "  https://github.com/%s/%s/releases/tag/%s\n", githubOwner, githubRepo, latest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// startUpdateCheck launches a background goroutine that checks for updates.
// It respects the cache TTL and is a no-op for dev builds.
func startUpdateCheck() {
	if version == "dev" {
		return
	}

	cached, err := readCache()
	if err == nil && time.Since(cached.CheckedAt) < checkInterval {
		if isNewer(cached.LatestVersion, version) {
			updateCh = make(chan updateResult, 1)
			updateCh <- updateResult{Latest: cached.LatestVersion}
		}
		return
	}

	updateCh = make(chan updateResult, 1)
	go func() {
		latest, err := fetchLatestVersion()
		if err != nil {
			updateCh <- updateResult{Err: err}
			return
		}
		writeCache(latest)
		updateCh <- updateResult{Latest: latest}
	}()
}

// printUpdateNotice prints an update hint to stderr if a newer version was found.
func printUpdateNotice() {
	if updateCh == nil {
		return
	}

	select {
	case res := <-updateCh:
		if res.Err != nil || !isNewer(res.Latest, version) {
			return
		}
		fmt.Fprintf(os.Stderr, "\nA new version of ptouch is available: %s → %s\n", version, res.Latest)
		fmt.Fprintf(os.Stderr, "Run `ptouch update` to upgrade.\n")
	case <-time.After(500 * time.Millisecond):
		// Don't block the CLI waiting for the network.
	}
}

// fetchLatestVersion queries the GitHub API for the latest release tag.
func fetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no tag in release")
	}

	return release.TagName, nil
}

// isNewer returns true if latest is a newer semver than current.
func isNewer(latest, current string) bool {
	l := normalizeSemver(latest)
	c := normalizeSemver(current)
	if l == "" || c == "" {
		return false
	}
	return compareSemver(l, c) > 0
}

// normalizeSemver strips a leading "v" and validates the format.
func normalizeSemver(v string) string {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return ""
	}
	return v
}

// compareSemver compares two semver strings (without "v" prefix).
// Returns >0 if a > b, <0 if a < b, 0 if equal.
func compareSemver(a, b string) int {
	pa := strings.SplitN(a, ".", 3)
	pb := strings.SplitN(b, ".", 3)
	for i := range 3 {
		na := atoi(pa[i])
		nb := atoi(pb[i])
		if na != nb {
			return na - nb
		}
	}
	return 0
}

func atoi(s string) int {
	// Strip anything after "-" (pre-release) for comparison.
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		s = s[:idx]
	}
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func cachePath() string {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "ptouch", "update-check.json")
}

func readCache() (*updateCache, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, err
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func writeCache(latest string) {
	c := updateCache{
		LatestVersion: latest,
		CheckedAt:     time.Now(),
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	p := cachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
		_ = os.WriteFile(p, data, 0o644)
	}
}
