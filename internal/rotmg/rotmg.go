// Package rotmg holds the RotMG domain model: the AppEngine environments, the
// per-build settings published at each environment's /app/init endpoint, and
// helpers to fetch and parse them.
package rotmg

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// appInitPath is the AppEngine endpoint that returns the current build
// settings as XML.
const appInitPath = "/app/init?platform=standalonewindows64&key=9KnJFxtTvLu2frXv"

// BuildType identifies which build of an environment is being processed.
type BuildType string

const (
	Client   BuildType = "Client"
	Launcher BuildType = "Launcher"
)

// Environment is a single RotMG AppEngine deployment.
type Environment struct {
	Name string
	URL  string
}

// Environments lists every environment the extractor polls, in order.
var Environments = []Environment{
	{Name: "Production", URL: "https://www.realmofthemadgod.com"},
}

// BuildInfo describes a single downloadable build.
type BuildInfo struct {
	BuildID      string
	BuildHash    string
	BuildVersion string
	BuildCDN     string
}

// Available reports whether the environment advertises this build.
func (b BuildInfo) Available() bool {
	return b.BuildHash != ""
}

// BuildURL is the CDN base URL for this build's files.
func (b BuildInfo) BuildURL() string {
	return b.BuildCDN + b.BuildHash + "/" + b.BuildID
}

// AppSettings holds the client and launcher build settings for an environment.
type AppSettings struct {
	Client   BuildInfo
	Launcher BuildInfo
}

// Build returns the BuildInfo for the given build type.
func (s AppSettings) Build(bt BuildType) BuildInfo {
	if bt == Launcher {
		return s.Launcher
	}
	return s.Client
}

// appSettingsXML mirrors the <AppSettings> document returned by /app/init.
type appSettingsXML struct {
	XMLName           xml.Name `xml:"AppSettings"`
	BuildID           string   `xml:"BuildId"`
	BuildHash         string   `xml:"BuildHash"`
	BuildVersion      string   `xml:"BuildVersion"`
	BuildCDN          string   `xml:"BuildCDN"`
	LauncherBuildID   string   `xml:"LauncherBuildId"`
	LauncherBuildHash string   `xml:"LauncherBuildHash"`
	LauncherBuildVer  string   `xml:"LauncherBuildVersion"`
	LauncherBuildCDN  string   `xml:"LauncherBuildCDN"`
}

// FetchAppSettings retrieves and parses the app settings for an environment.
func FetchAppSettings(ctx context.Context, env Environment) (AppSettings, error) {
	url := env.URL + appInitPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return AppSettings{}, fmt.Errorf("building request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AppSettings{}, fmt.Errorf("requesting %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AppSettings{}, fmt.Errorf("requesting %s: unexpected status %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AppSettings{}, fmt.Errorf("reading response: %w", err)
	}

	return parseAppSettings(body)
}

func parseAppSettings(data []byte) (AppSettings, error) {
	var doc appSettingsXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return AppSettings{}, fmt.Errorf("parsing app settings xml: %w", err)
	}

	return AppSettings{
		Client: BuildInfo{
			BuildID:      doc.BuildID,
			BuildHash:    doc.BuildHash,
			BuildVersion: doc.BuildVersion,
			BuildCDN:     doc.BuildCDN,
		},
		Launcher: BuildInfo{
			BuildID:      doc.LauncherBuildID,
			BuildHash:    doc.LauncherBuildHash,
			BuildVersion: doc.LauncherBuildVer,
			BuildCDN:     doc.LauncherBuildCDN,
		},
	}, nil
}
