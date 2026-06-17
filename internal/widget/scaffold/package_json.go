package scaffold

import "fmt"

type PackageJSONRenderer struct{}

func (PackageJSONRenderer) Render(spec Spec) []File {
	return []File{{
		Path: "package.json",
		Content: []byte(fmt.Sprintf(`{
  "name": %q,
  "widgetName": %q,
  "version": "1.0.0",
  "description": %q,
  "copyright": "© Mendix Technology BV 2026. All rights reserved.",
  "author": "",
  "engines": {
    "node": ">=16"
  },
  "license": "Apache-2.0",
  "config": {
    "projectPath": %q,
    "mendixHost": "http://localhost:8080",
    "developmentPort": 3000
  },
  "packagePath": %q,
  "scripts": {
    "start": "pluggable-widgets-tools start:server",
    "dev": "pluggable-widgets-tools start:web",
    "build": "pluggable-widgets-tools build:web",
    "lint": "pluggable-widgets-tools lint",
    "lint:fix": "pluggable-widgets-tools lint:fix",
    "prerelease": "npm run lint",
    "release": "pluggable-widgets-tools release:web"
  },
  "devDependencies": {
    "@mendix/pluggable-widgets-tools": "^11.11.0"
  },
  "dependencies": {
    "classnames": "^2.5.1"
  },
  "resolutions": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "overrides": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  }
}
`, spec.PackageName, spec.Name, spec.Description, spec.ProjectPath, spec.PackagePath)),
	}}
}
