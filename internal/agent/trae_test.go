package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/provider"
)

// #4365, #2907: Trae and Trae CN keep their models on the vendor's cloud
// and have none magpie may write — no settings.json key, no
// chatLanguageModels.json, and state.vscdb only a cache of what the cloud
// answered. So neither row writes into Trae: what it gives is the values
// Trae's own Add Model dialog is handed, which is what the user pastes
// there. Both builds are listed, under their own ids and userData folders.
func TestTraeGuide(t *testing.T) {
	home := traeHome(t)
	if err := provider.Save(provider.Provider{ID: "deepseek", Name: "DeepSeek", Chat: "https://api.deepseek.com/v1", Key: "k", Models: []string{"v4.1-flash"}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		mk     func(string) *Agent
		id, nm string
		dir    string
	}{
		{trae, "trae", "Trae", filepath.Join(home, "Library", "Application Support", "Trae")},
		{traeCN, "trae-cn", "Trae CN", filepath.Join(home, "Library", "Application Support", "Trae CN")},
	} {
		a := c.mk(home)
		if a.ID != c.id || a.Name != c.nm {
			t.Errorf("id %q name %q, want %q %q", a.ID, a.Name, c.id, c.nm)
		}
		if runtime.GOOS == "darwin" && a.Dir != c.dir {
			t.Errorf("%s: dir %q, want %q", c.id, a.Dir, c.dir)
		}
		// nothing to write: a field here would claim magpie sets something
		// in an app it can't touch
		if len(a.Fields) != 0 {
			t.Fatalf("%s: %d fields, want none", c.id, len(a.Fields))
		}
		guide := a.Guide()
		want := []Guide{
			{Label: "Endpoint", Value: gateway.URL() + "/v1/chat/completions"},
			// the key names the app, so the gateway attributes the requests
			{Label: "API key", Value: gateway.TokenFor(c.id)},
			{Label: "Model", Value: "deepseek/v4.1-flash"},
		}
		if len(guide) != len(want) {
			t.Fatalf("%s: guide %+v, want %d entries", c.id, guide, len(want))
		}
		for i, g := range guide {
			if g != want[i] {
				t.Errorf("%s: guide[%d] = %+v, want %+v", c.id, i, g, want[i])
			}
		}
	}
}

// Trae's Add Model dialog appends /chat/completions to a base URL it is
// given and takes a whole request URL as typed. The row names the whole
// request, so the URL pasted in is the one magpie answers on.
func TestTraeEndpointIsTheWholeRequest(t *testing.T) {
	home := traeHome(t)
	for _, a := range []*Agent{trae(home), traeCN(home)} {
		g := a.Guide()[0]
		if g.Label != "Endpoint" || !strings.HasSuffix(g.Value, "/v1/chat/completions") {
			t.Fatalf("%s: endpoint %q", a.ID, g.Value)
		}
		if strings.HasSuffix(g.Value, "/v1/chat/completions/chat/completions") {
			t.Fatalf("%s: the request URL given a base's ending: %q", a.ID, g.Value)
		}
	}
}

// A row with no model listed for it names none: a model magpie doesn't
// list is one its gateway turns away as unknown as soon as Trae asks for
// it, and a name typed by hand that 404s reads as the row being wrong.
func TestTraeModelOnlyWhenListed(t *testing.T) {
	home := traeHome(t)
	hidden := provider.SetHiddenModels
	g, _ := provider.CatalogFor("trae")
	var ids []string
	for _, e := range g {
		ids = append(ids, e.ID)
	}
	if err := hidden("trae", ids); err != nil {
		t.Fatal(err)
	}
	last := trae(home).Guide()[2]
	if last.Label != "Model" || last.Value != "" || last.Note == "" {
		t.Fatalf("no model listed: %+v, want a note and no value", last)
	}
}

// Trae is found by its userData folder, or by its app installed and not
// opened once yet — a Mac has ~/Library/Application Support/Trae only
// after the first run. Each build by its own bundle id: Trae CN's app is
// not Trae's, and another app under that name is neither.
func TestTraeDetected(t *testing.T) {
	judgeFor(t, "darwin")
	home := traeHome(t)
	apps, userApps := t.TempDir(), t.TempDir()
	was := appFolders
	appFolders = func() []string { return []string{apps, userApps} }
	t.Cleanup(func() { appFolders = was })
	plist := func(id string) string {
		return `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>` + id + `</string><key>CFBundleShortVersionString</key><string>3.6.13</string></dict></plist>`
	}
	intl, cn := trae(home), traeCN(home)
	if intl.Detected() || cn.Detected() {
		t.Fatalf("nothing here: Trae %v, Trae CN %v", intl.Detected(), cn.Detected())
	}
	// the app, not opened yet
	installApp(t, apps, "Trae.app", plist("com.trae.app"))
	if !intl.Detected() || cn.Detected() {
		t.Fatalf("Trae.app: Trae %v, Trae CN %v", intl.Detected(), cn.Detected())
	}
	// Trae CN's own, in ~/Applications
	installApp(t, userApps, "Trae CN.app", plist("cn.trae.app"))
	if !cn.Detected() {
		t.Fatal("Trae CN.app installed: not detected")
	}
	// another app under Trae's name is not Trae
	os.RemoveAll(filepath.Join(apps, "Trae.app"))
	installApp(t, apps, "Trae.app", plist("com.example.other"))
	if intl.Detected() {
		t.Fatal("another bundle id taken for Trae")
	}
	// the folder stays after the app is dragged away, and is the sign then
	if err := os.MkdirAll(intl.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !intl.Detected() {
		t.Fatal("Trae's userData folder: not detected")
	}
}

// Both builds answer by their own key on magpie, so a request from either
// is attributed to it rather than to the other, and one build's key isn't
// taken for the other's.
func TestTraeKeysAreItsOwn(t *testing.T) {
	home := traeHome(t)
	g, c := trae(home).Guide()[1].Value, traeCN(home).Guide()[1].Value
	if g == c {
		t.Fatalf("both builds given %q", g)
	}
	if !strings.HasSuffix(g, "-trae") || !strings.HasSuffix(c, "-trae-cn") {
		t.Fatalf("keys %q %q", g, c)
	}
}

// traeHome is a home of the test's own with no Trae in it.
func traeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	return home
}
