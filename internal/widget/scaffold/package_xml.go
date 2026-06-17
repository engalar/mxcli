package scaffold

import (
	"fmt"
	"strings"
)

type PackageXMLRenderer struct{}

func (PackageXMLRenderer) Render(spec Spec) []File {
	// PackagePath is dot-separated (e.g. "com.helpdesk.widget"). Convert to slash path.
	// File path uses lowercase name to match pluggable-widgets-tools build output.
	pkgPath := strings.ReplaceAll(spec.PackagePath, ".", "/")
	filePath := fmt.Sprintf("%s/%s/", pkgPath, strings.ToLower(spec.Name))
	content := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" ?>
<package xmlns="http://www.mendix.com/package/1.0/">
    <clientModule name=%q version="1.0.0" xmlns="http://www.mendix.com/clientModule/1.0/">
        <widgetFiles>
            <widgetFile path=%q/>
        </widgetFiles>
        <files>
            <file path=%q/>
        </files>
    </clientModule>
</package>
`, spec.Name, spec.Name+".xml", filePath)
	return []File{{
		Path:    "src/package.xml",
		Content: []byte(content),
	}}
}
