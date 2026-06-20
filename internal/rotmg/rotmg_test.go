package rotmg

import "testing"

// sampleXML mirrors the real /app/init response: game-settings fields (which
// must be ignored) precede the build fields, exactly as the live endpoint
// returns them.
const sampleXML = `<AppSettings>
  <UseExternalPayments>1</UseExternalPayments>
  <MaxStackablePotions>6</MaxStackablePotions>
  <PotionPurchaseCosts><cost>5</cost><cost>10</cost></PotionPurchaseCosts>
  <FilterList>
some\bregex
END</FilterList>
  <ExtendViewRadius/>
  <BuildId>rotmg-exalt-win-64</BuildId>
  <BuildHash>a1c8d9ae2a2508dcc3994b33dd6a803a</BuildHash>
  <BuildVersion>a9cb33d6944a7f8bbf7181c71cc932f11ed85ba3</BuildVersion>
  <BuildCDN>https://rotmg-build.decagames.com/build-release/</BuildCDN>
  <LauncherBuildId>RotMG-Exalt-Installer</LauncherBuildId>
  <LauncherBuildHash>d554e291899750f9d36c750798e85646</LauncherBuildHash>
  <LauncherBuildVersion>386777c109b1f7385ab1636bc7e82a1d7f451352</LauncherBuildVersion>
  <LauncherBuildCDN>https://rotmg-build.decagames.com/launcher-release/</LauncherBuildCDN>
</AppSettings>`

func TestParseAppSettings(t *testing.T) {
	got, err := parseAppSettings([]byte(sampleXML))
	if err != nil {
		t.Fatalf("parseAppSettings: %v", err)
	}

	if got.Client.BuildID != "rotmg-exalt-win-64" {
		t.Errorf("client BuildID = %q", got.Client.BuildID)
	}
	if got.Client.BuildHash != "a1c8d9ae2a2508dcc3994b33dd6a803a" {
		t.Errorf("client BuildHash = %q", got.Client.BuildHash)
	}
	if got.Launcher.BuildID != "RotMG-Exalt-Installer" {
		t.Errorf("launcher BuildID = %q", got.Launcher.BuildID)
	}
	if !got.Client.Available() {
		t.Error("client should be Available")
	}

	wantURL := "https://rotmg-build.decagames.com/build-release/a1c8d9ae2a2508dcc3994b33dd6a803a/rotmg-exalt-win-64"
	if got := got.Client.BuildURL(); got != wantURL {
		t.Errorf("BuildURL = %q, want %q", got, wantURL)
	}
}

func TestUnavailableBuild(t *testing.T) {
	var empty BuildInfo
	if empty.Available() {
		t.Error("empty BuildInfo should not be Available")
	}
}
