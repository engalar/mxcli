package kernel

var typeIconMap = map[string]string{
	"systemoverview":  "🗺",
	"navigation":      "🧭",
	"projectsecurity": "🛡",
	"module":          "⬡",
	"folder":          "📁",
	"category":        "⊟",
	"domainmodel":     "⊞",
	"entity":          "▣",
	"externalentity":  "⊡",
	"association":     "↔",
	"enumeration":     "≡",
	"microflow":       "⚙",
	"nanoflow":        "⚡",
	"workflow":        "🔀",
	"page":            "▤",
	"snippet":         "⬔",
	"layout":          "⬕",
	"imagecollection": "🖼️",
	"constant":        "π",
	"scheduledevent":  "⏰",
	"javaaction":      "☕",
	"javascriptaction": "JS",
	"security":        "🔒",
	"modulerole":      "👤",
	"userrole":        "👥",
	"businesseventservice": "📡",
	"databaseconnection":   "🗄",
	"odataservice":         "🌐",
	"odataclient":          "🔗",
	"publishedrestservice": "🌍",
	"consumedrestservice":  "🔌",
	"navprofile":           "⊕",
	"navhome":              "⌂",
	"navmenu":              "☰",
	"navmenuitem":          "→",
}

func IconFor(nodeType string) string {
	if icon, ok := typeIconMap[nodeType]; ok {
		return icon
	}
	return "·"
}
