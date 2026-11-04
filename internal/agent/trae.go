package agent

// Trae (ByteDance's IDE) and Trae CN keep no model list magpie can write.
// Their models are the vendor's, held server-side: the Add Model dialog
// posts to Trae's cloud (api/ide/v1/add_custom_model), and the local
// state.vscdb carries only a cache of what the cloud answered (#2907).
// There is no settings.json key, no chatLanguageModels.json, and no trae://
// route that adds one (#4365).
//
// So these rows write nothing and read nothing of Trae's. A row is the
// values the user types into Trae's own Add Model dialog — Custom (OpenAI
// Compatible), the endpoint, the key, a model — each to copy, and magpie is
// added by hand there.
//
// That a custom model's requests reach a loopback address, which #4365 asked
// before this was built, is settled: a model added with client_connect true
// is fetched by Trae's own process on the user's machine. Trae's chat
// modules read that flag as use_remote_service: !client_connect, and its
// CustomModelProxy is a cloud-to-client executor: the cloud pushes a request
// down a tunnel the client opened and the client dials, so a private address
// is reachable where the cloud could not reach it (#2907). Trae CN's build
// carries the same modules and the same strings.

import (
	"path/filepath"
	"runtime"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/provider"
)

func trae(home string) *Agent   { return traeSite(home, traeGlobal) }
func traeCN(home string) *Agent { return traeSite(home, traeChina) }

// traeBuild is what tells Trae's two builds apart: its own name, its own
// userData folder, and its own key on magpie.
type traeBuild struct {
	id, name, data string
	aliases        []string
}

var (
	traeGlobal = traeBuild{id: "trae", name: "Trae", data: "Trae", aliases: []string{"trae-ai"}}
	traeChina  = traeBuild{id: "trae-cn", name: "Trae CN", data: "Trae CN", aliases: []string{"traecn"}}
)

// traeSite is a build of Trae on this machine: its userData, for the row to
// show, and whether it is here — that folder, or the app installed and not
// opened once yet.
func traeSite(home string, b traeBuild) *Agent {
	dir := traeData(home, b)
	return &Agent{
		ID: b.id, Name: b.name, Icon: "trae", Aliases: b.aliases,
		Dir: dir,
		detect: func() bool {
			return isDir(dir) || traeApp(b).installed()
		},
		// no fields: nothing on it claims to be set in Trae, and
		// connectable() leaves a row with no fields and no models unconnected
		Guide: func() []Guide { return traeGuide(b) },
	}
}

// traeData is where b's userData is, whether or not it is there yet.
func traeData(home string, b traeBuild) string {
	switch runtime.GOOS {
	case "windows":
		if appdata := appdir.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, b.data)
		}
		return ""
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", b.data)
	}
	return filepath.Join(home, ".config", b.data)
}

// traeApp is b's desktop app, as desktopApps has it; the zero app when it
// has none, which installed() answers false for.
func traeApp(b traeBuild) desktopApp { return desktopApps[b.id] }

// traeGuide is what Trae's Add Model dialog is given, in the order it asks:
// the request URL (it appends /chat/completions to a base, so the row names
// the whole request), the key, and a model. The model is one magpie lists
// for this row; with none listed there is none to name, and naming one would
// be a name the gateway turns away as unknown.
func traeGuide(b traeBuild) []Guide {
	guide := []Guide{
		{Label: "Endpoint", Value: gateway.URL() + "/v1/chat/completions"},
		{Label: "API key", Value: agentKey(b.id)},
	}
	if shown, _ := provider.CatalogFor(b.id); len(shown) > 0 {
		return append(guide, Guide{Label: "Model", Value: shown[0].ID})
	}
	return append(guide, Guide{Label: "Model", Note: "Any model id magpie lists"})
}

// Guide is one value an app's own dialog is given by hand, where magpie has
// no way to write it: the label that dialog asks it by, the value to copy,
// and, where there is none of it yet, what to do about that instead.
type Guide struct {
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
	Note  string `json:"note,omitempty"`
}
