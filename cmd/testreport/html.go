// SPDX-License-Identifier: Apache-2.0
package main

import (
	"html/template"
	"io"
	"time"
)

const htmlTmpl = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>mxcli Test Report</title>
<style>
body{font-family:monospace;max-width:960px;margin:40px auto;padding:0 20px;background:#111;color:#eee}
h1{color:#fff}
table{width:100%;border-collapse:collapse;margin:16px 0}
th,td{padding:8px 12px;text-align:left;border-bottom:1px solid #333}
th{background:#222;color:#aaa}
.pass{color:#4caf50}.fail{color:#f44336}.warn{color:#ff9800}
details>summary{cursor:pointer;padding:4px;background:#1e1e1e}
pre{background:#1a1a1a;padding:12px;overflow-x:auto;font-size:12px}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px}
.badge-pass{background:#1b5e20;color:#a5d6a7}.badge-fail{background:#b71c1c;color:#ef9a9a}
</style>
</head>
<body>
<h1>mxcli Test Report</h1>
<p>Generated: {{.GeneratedAt}} | Git: <code>{{.GitHash}}</code></p>

<h2>Test Layers</h2>
<table>
<tr><th>Layer</th><th>Tests</th><th>Pass</th><th>Fail</th><th>Time</th><th>Status</th></tr>
{{range .Layers}}
<tr>
  <td>{{.Name}}</td>
  <td>{{.Total}}</td>
  <td class="pass">{{.Pass}}</td>
  <td class="{{if gt .Fail 0}}fail{{else}}pass{{end}}">{{.Fail}}</td>
  <td>{{printf "%.1fs" .Elapsed}}</td>
  <td>{{if gt .Fail 0}}<span class="badge badge-fail">FAIL</span>{{else}}<span class="badge badge-pass">PASS</span>{{end}}</td>
</tr>
{{end}}
</table>

{{if .Failures}}
<h2>Failures</h2>
{{range .Failures}}
<details open>
<summary class="fail">{{.Package}}/{{.Test}}</summary>
<pre>{{.Output}}</pre>
</details>
{{end}}
{{end}}

{{if .BenchDiff}}
<h2>Benchmark Regressions</h2>
<pre class="warn">{{.BenchDiff}}</pre>
{{end}}

<h2>Coverage</h2>
<p><a href="coverage.html" style="color:#90caf9">Open source coverage view →</a></p>
</body>
</html>`

type htmlLayerRow struct {
	Name    template.HTML
	Total   int
	Pass    int
	Fail    int
	Elapsed float64
}

type htmlData struct {
	GeneratedAt string
	GitHash     string
	Layers      []htmlLayerRow
	Failures    []FailureDetail
	BenchDiff   string
}

var htmlTemplate = template.Must(template.New("report").Parse(htmlTmpl))

func renderHTML(w io.Writer, layerMap map[string]*LayerSummary, benchDiff, gitHash string) error {
	sorted := sortedLayers(layerMap)
	rows := make([]htmlLayerRow, 0, len(sorted))
	var allFailures []FailureDetail
	for _, l := range sorted {
		rows = append(rows, htmlLayerRow{
			Name: template.HTML(l.Name), Total: l.Pass + l.Fail,
			Pass: l.Pass, Fail: l.Fail, Elapsed: l.Elapsed,
		})
		allFailures = append(allFailures, l.Failures...)
	}
	return htmlTemplate.Execute(w, htmlData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		GitHash:     gitHash,
		Layers:      rows,
		Failures:    allFailures,
		BenchDiff:   benchDiff,
	})
}
