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

// appInitHeaders are the headers the game client sends; the endpoint requires
// them to return the build settings.
var appInitHeaders = map[string]string{
	"game_net":         "Unity",
	"play_platform":    "Unity",
	"game_net_user_id": "",
}

// BuildType identifies which build of a platform is being processed.
type BuildType string

const (
	Client   BuildType = "Client"
	Launcher BuildType = "Launcher"
)

// Platform is a downloadable client platform. The Windows and macOS builds are
// served from different domains and platform query parameters, and yield
// different BuildIds (and launcher installer formats), so each is watched
// independently.
type Platform struct {
	Name    string // "windows", "macos"
	InitURL string // full /app/init URL incl. key + platform query params
}

// Platforms are the known platforms, keyed by config name.
var Platforms = map[string]Platform{
	"windows": {
		Name:    "windows",
		InitURL: "https://realmofthemadgodhrd.appspot.com/app/init?key=9KnJFxtTvLu2frXv&platform=standalonewindows64",
	},
	"macos": {
		Name:    "macos",
		InitURL: "https://www.realmofthemadgod.com/app/init?platform=standaloneosxuniversal&key=9KnJFxtTvLu2frXv",
	},
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

// FetchAppSettings retrieves and parses the app settings for a platform.
func FetchAppSettings(ctx context.Context, p Platform) (AppSettings, error) {
	url := p.InitURL

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return AppSettings{}, fmt.Errorf("building request: %w", err)
	}
	for k, v := range appInitHeaders {
		req.Header.Set(k, v)
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
