package model

type Component string

const (
	Koko      Component = "koko"
	Guacamole Component = "guacamole"
	Omnidb    Component = "omnidb"
	Xrdp      Component = "xrdp"
	Lion      Component = "lion"
	Magnus    Component = "magnus"
	Razor     Component = "razor"

	UnknownComponent Component = "Unknown"
)

var componentMap = map[string]Component{
	"koko":      Koko,
	"guacamole": Guacamole,
	"omnidb":    Omnidb,
	"xrdp":      Xrdp,
	"lion":      Lion,
	"magnus":    Magnus,
	"razor":     Razor,
}

func SupportedComponent(s string) (Component, bool) {
	comp, ok := componentMap[s]
	return comp, ok
}
