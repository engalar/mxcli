package scaffold

type ConfigFilesRenderer struct{}

func (ConfigFilesRenderer) Render(spec Spec) []File {
	return []File{
		{Path: ".eslintrc.js", Content: []byte(`const base = require("@mendix/pluggable-widgets-tools/configs/eslint.js.base.json");

module.exports = {
    ...base
};
`)},
		{Path: "prettier.config.js", Content: []byte(`const base = require("@mendix/pluggable-widgets-tools/configs/prettier.base.json");

module.exports = {
    ...base,
    plugins: [require.resolve("@prettier/plugin-xml")],
};
`)},
		{Path: ".prettierignore", Content: []byte("tests/testProject/\n")},
		{Path: ".gitattributes", Content: []byte(`# Set the default behavior, in case people don't have core.autocrlf set.
* text=auto

# Explicitly declare text files you want to always be normalized and converted
# to native line endings on checkout.
*.ts text eol=lf
*.tsx text eol=lf
*.js text eol=lf
*.jsx text eol=lf
*.css text eol=lf
*.scss text eol=lf
*.json text eol=lf
*.xml text eol=lf
*.md text eol=lf
*.gitattributes eol=lf
*.gitignore eol=lf

# Denote all files that are truly binary and should not be modified.
*.png binary
*.jpg binary
*.gif binary
`)},
		{Path: "LICENSE", Content: []byte(`The Apache License v2.0

Copyright © Mendix Technology BV 2026. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
`)},
	}
}
