package tui

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

// --- View ---

func (a App) View() string {
	if a.width == 0 {
		return "mxcli tui — loading...\n\nPress q to quit"
	}

	if a.picker != nil {
		return a.picker.View()
	}

	active := a.views.Active()

	// For non-browser views, delegate rendering entirely
	if active.Mode() != ModeBrowser {
		contentH := a.height
		content := active.Render(a.width, contentH)

		if a.showHelp {
			helpView := renderHelp(a.width, a.height)
			content = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, helpView,
				lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
		}

		return content
	}

	// Browser mode: App renders chrome (tab bar, hint bar, status bar)

	// Tab bar (line 1) with mode badge + context summary on the right
	tabLine := a.tabBar.View(a.width)
	tab := a.activeTabPtr()
	if tab != nil {
		modeBadge := kernel.AccentStyle.Render(active.Mode().String())
		summary := renderContextSummary(tab.AllNodes)
		rightSide := modeBadge
		if summary != "" {
			rightSide += kernel.BreadcrumbDimStyle.Render(" │ ") + kernel.BreadcrumbDimStyle.Render(summary)
		}
		rightWidth := lipgloss.Width(rightSide) + 1 // 1 char right padding
		tabWidth := lipgloss.Width(tabLine)
		if tabWidth+rightWidth <= a.width {
			// Replace trailing spaces with gap + right side
			trimmed := strings.TrimRight(tabLine, " ")
			trimmedWidth := lipgloss.Width(trimmed)
			gap := a.width - trimmedWidth - rightWidth
			if gap < 2 {
				gap = 2
			}
			tabLine = trimmed + strings.Repeat(" ", gap) + rightSide + " "
		}
	}

	// Content area (chromeHeight + 1 for the LLM anchor line)
	contentH := a.height - chromeHeight - 1
	if contentH < 5 {
		contentH = 5
	}
	content := active.Render(a.width, contentH)

	// Hint bar — declarative from active view
	a.hintBar.SetHints(active.Hints())
	hintLine := a.hintBar.View(a.width)

	// Status bar — declarative from active view
	info := active.StatusInfo()
	a.statusBar.SetBreadcrumb(info.Breadcrumb)
	a.statusBar.SetPosition(info.Position)
	a.statusBar.SetMode(info.Mode)
	if a.checkNavActive && len(a.checkNavLocations) > 0 {
		loc := a.checkNavLocations[a.checkNavIndex]
		navInfo := fmt.Sprintf("[%d/%d] %s: %s  ]e next  [e prev",
			a.checkNavIndex+1, len(a.checkNavLocations),
			loc.Code, docNameToQualifiedName(loc.ModuleName, loc.DocumentName))
		a.statusBar.SetCheckBadge(kernel.CheckWarnStyle.Render(navInfo))
	} else {
		a.statusBar.SetCheckBadge(formatCheckBadge(a.checkErrors, a.checkRunning))
	}
	// Agent activity badge
	if a.agentExecCtx != nil {
		a.statusBar.SetAgentBadge(kernel.AgentBadgeStyle.Render("⚡agent"))
	} else {
		a.statusBar.SetAgentBadge("")
	}
	viewModeNames := a.collectViewModeNames()
	a.statusBar.SetViewDepth(a.views.Depth(), viewModeNames)
	statusLine := kernel.StatusBarStyle.Width(a.width).Render(a.statusBar.View(a.width))

	// LLM anchor: machine-readable command list (Faint, not visible to users in practice)
	anchorStyle := lipgloss.NewStyle().Foreground(kernel.MutedColor).Faint(true)
	anchorLine := anchorStyle.Render("[mxcli:commands] h:back l:open Space:jump /:filter b:bson c:compare d:diagram z:zen Tab:toggle x:exec r:refresh y:copy !:check ]e:next-error [e:prev-error t:tab T:new-tab W:close-tab 1-9:switch ?:help ::palette")

	rendered := anchorLine + "\n" + tabLine + "\n" + content + "\n" + hintLine + "\n" + statusLine

	if a.showHelp {
		helpView := renderHelp(a.width, a.height)
		rendered = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, helpView,
			lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
	}

	return rendered
}

// --- Load helpers ---

func buildDiagramHTML(elkJSON, nodeType, qualifiedName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><title>%s %s</title>
<script src="https://cdn.jsdelivr.net/npm/elkjs@0.9.3/lib/elk.bundled.js"></script>
<style>
body{margin:0;background:#1e1e2e;color:#cdd6f4;font-family:monospace;overflow:hidden}
svg{width:100vw;height:100vh}
</style>
</head><body><div id="diagram"></div><script>
const RAW = %s;
const NS = "http://www.w3.org/2000/svg";

// Colours keyed by entity category
const CAT_FILL = {persistent:"#313244",nonpersistent:"#2a2a3e",external:"#2d2b3e",view:"#1e2d3e"};
const CAT_HEAD  = {persistent:"#89b4fa",nonpersistent:"#a6e3a1",external:"#fab387",view:"#94e2d5"};
const ASSOC_STROKE = "#585b70";
const GEN_STROKE   = "#f5c2e7";
const ATTR_H = 18, HDR_H = 28, PAD = 12;

function toELKGraph(d) {
  const children = (d.entities||[]).map(e=>({
    id: e.id, width: e.width, height: e.height,
    layoutOptions:{"elk.portConstraints":"FIXED_SIDE"},
    ports: [
      {id:e.id+":L", x:0,          y:e.height/2, width:0,height:0,
       layoutOptions:{"elk.port.side":"WEST"}},
      {id:e.id+":R", x:e.width,    y:e.height/2, width:0,height:0,
       layoutOptions:{"elk.port.side":"EAST"}},
      {id:e.id+":T", x:e.width/2,  y:0,          width:0,height:0,
       layoutOptions:{"elk.port.side":"NORTH"}},
      {id:e.id+":B", x:e.width/2,  y:e.height,   width:0,height:0,
       layoutOptions:{"elk.port.side":"SOUTH"}},
    ]
  }));
  const edges = [];
  (d.associations||[]).forEach(a=>{
    edges.push({id:a.id, sources:[a.sourceId], targets:[a.targetId], _assoc:a});
  });
  (d.generalizations||[]).forEach((g,i)=>{
    edges.push({id:"gen-"+i, sources:[g.childId], targets:[g.parentId], _gen:true});
  });
  return {
    id:"root",
    layoutOptions:{
      "elk.algorithm":"layered",
      "elk.direction":"RIGHT",
      "elk.spacing.nodeNode":"40",
      "elk.layered.spacing.nodeNodeBetweenLayers":"80",
      "elk.edgeRouting":"ORTHOGONAL",
    },
    children, edges
  };
}

function svgEl(tag, attrs, parent) {
  const el = document.createElementNS(NS, tag);
  for (const [k,v] of Object.entries(attrs)) el.setAttribute(k,v);
  if (parent) parent.appendChild(el);
  return el;
}

function renderEdgePath(edge, svg) {
  if (!edge.sections||!edge.sections.length) return;
  const isGen = !!edge._gen;
  edge.sections.forEach(sec=>{
    const pts = [sec.startPoint, ...(sec.bendPoints||[]), sec.endPoint];
    const d = pts.map((p,i)=>(i===0?"M":"L")+p.x+" "+p.y).join(" ");
    svgEl("path",{d, stroke: isGen?GEN_STROKE:ASSOC_STROKE,
      "stroke-width":"1.5", fill:"none",
      "marker-end": isGen?"url(#arr-gen)":"url(#arr-assoc)"
    }, svg);
  });
  // Label
  const a = edge._assoc;
  if (a&&a.name&&edge.sections[0]) {
    const mid = edge.sections[0].startPoint;
    const end = edge.sections[edge.sections.length-1].endPoint;
    const lx = (mid.x+end.x)/2, ly = (mid.y+end.y)/2;
    const t = svgEl("text",{x:lx,y:ly-4,"text-anchor":"middle",
      "font-size":"10","fill":"#a6adc8"}, svg);
    t.textContent = a.name;
  }
}

function renderEntity(node, svg, byID) {
  const raw = byID[node.id]||{name:node.id,category:"persistent",attributes:[]};
  const cat = raw.category||"persistent";
  const fill = CAT_FILL[cat]||"#313244";
  const hdr  = CAT_HEAD[cat]||"#89b4fa";
  const g = svgEl("g",{transform:"translate("+node.x+","+node.y+")"}, svg);

  // Shadow
  svgEl("rect",{x:2,y:2,width:node.width,height:node.height,rx:4,
    fill:"rgba(0,0,0,.4)"}, g);
  // Body
  svgEl("rect",{x:0,y:0,width:node.width,height:node.height,rx:4,
    fill, stroke: raw.isFocus?"#cba6f7":"#45475a","stroke-width": raw.isFocus?"2":"1"}, g);
  // Header band
  svgEl("rect",{x:0,y:0,width:node.width,height:HDR_H,rx:4,fill:hdr,"fill-opacity":"0.25"}, g);
  svgEl("rect",{x:0,y:HDR_H-2,width:node.width,height:2,fill:hdr,"fill-opacity":"0.3"}, g);

  const nameEl = svgEl("text",{x:node.width/2,y:HDR_H-8,
    "text-anchor":"middle","font-size":"12","font-weight":"bold",
    fill:hdr}, g);
  nameEl.textContent = raw.name;

  (raw.attributes||[]).forEach((attr,i)=>{
    const y = HDR_H + 4 + i*ATTR_H;
    const row = svgEl("g",{transform:"translate(0,"+y+")"}, g);
    const t = svgEl("text",{x:PAD,y:12,"font-size":"10",fill:"#cdd6f4"}, row);
    t.textContent = attr.name;
    const tt = svgEl("text",{x:node.width-PAD,y:12,"text-anchor":"end",
      "font-size":"10",fill:"#6c7086"}, row);
    tt.textContent = attr.type;
  });
}

const ELK = new ELKConstructor();
ELK.layout(toELKGraph(RAW)).then(graph=>{
  const byID = {};
  (RAW.entities||[]).forEach(e=>{ byID[e.id]=e; });

  let minX=Infinity,minY=Infinity,maxX=0,maxY=0;
  (graph.children||[]).forEach(n=>{
    minX=Math.min(minX,n.x); minY=Math.min(minY,n.y);
    maxX=Math.max(maxX,n.x+n.width); maxY=Math.max(maxY,n.y+n.height);
  });
  const pad=40;
  const vw=maxX-minX+pad*2, vh=maxY-minY+pad*2;

  const svg=svgEl("svg",{
    viewBox:(minX-pad)+" "+(minY-pad)+" "+vw+" "+vh,
    xmlns:NS
  });

  // Arrowhead markers
  const defs=svgEl("defs",{},svg);
  function marker(id,color){
    const m=svgEl("marker",{id,markerWidth:"8",markerHeight:"8",
      refX:"6",refY:"3",orient:"auto"},defs);
    svgEl("path",{d:"M0,0 L0,6 L8,3 z",fill:color},m);
  }
  marker("arr-assoc",ASSOC_STROKE);
  marker("arr-gen",GEN_STROKE);

  // Edges first (drawn behind entities)
  (graph.edges||[]).forEach(e=>renderEdgePath(e,svg));

  // Entities on top
  (graph.children||[]).forEach(n=>renderEntity(n,svg,byID));

  document.getElementById("diagram").appendChild(svg);
}).catch(err=>{
  document.getElementById("diagram").textContent="Layout error: "+err;
});
</script></body></html>`, nodeType, qualifiedName, elkJSON)
}

// CmdResultMsg carries output from any mxcli command.
type CmdResultMsg struct {
	Output string
	Err    error
}

// irregularPlurals maps singular type names to their correct plural forms
// for types where simply appending "s" produces incorrect English.
var irregularPlurals = map[string]string{
	"Index": "indexes",
}

// renderContextSummary counts top-level node types and returns a compact summary.
func renderContextSummary(nodes []*TreeNode) string {
	if len(nodes) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, n := range nodes {
		counts[n.Type]++
	}
	// Display in a predictable order
	order := []struct {
		key    string
		plural string
	}{
		{"Module", "modules"},
		{"Entity", "entities"},
		{"Microflow", "microflows"},
		{"Page", "pages"},
		{"Nanoflow", "nanoflows"},
		{"Enumeration", "enumerations"},
	}
	var parts []string
	used := map[string]bool{}
	for _, o := range order {
		if c, ok := counts[o.key]; ok {
			parts = append(parts, fmt.Sprintf("%d %s", c, o.plural))
			used[o.key] = true
		}
	}
	// Add remaining types not in the predefined order
	for k, c := range counts {
		if !used[k] {
			plural, ok := irregularPlurals[k]
			if !ok {
				plural = strings.ToLower(k) + "s"
			}
			parts = append(parts, fmt.Sprintf("%d %s", c, plural))
		}
	}
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return strings.Join(parts, ", ")
}

// collectViewModeNames returns the mode names for all views in the stack.
func (a App) collectViewModeNames() []string {
	return a.views.ModeNames()
}

// inferBsonType maps tree node types to valid bson object types.
func inferBsonType(nodeType string) string {
	switch strings.ToLower(nodeType) {
	case "page", "microflow", "nanoflow", "workflow",
		"enumeration", "snippet", "layout", "entity", "association",
		"imagecollection", "javaaction", "javascriptaction", "constant":
		return strings.ToLower(nodeType)
	default:
		return ""
	}
}

// agentStateInfo is the structured state returned by the "state" action.
type agentStateInfo struct {
	Mode         string         `json:"mode"`
	Project      string         `json:"project"`
	SelectedNode *agentNodeInfo `json:"selectedNode,omitempty"`
	PreviewMode  string         `json:"previewMode,omitempty"`
	CheckErrors  int            `json:"checkErrors"`
	CheckRunning bool           `json:"checkRunning"`
}

// agentNodeInfo describes the currently selected tree node.
type agentNodeInfo struct {
	Type          string `json:"type"`
	QualifiedName string `json:"qualifiedName"`
}

// agentBuildState extracts rich TUI state for the agent.
func agentBuildState(a App) agentStateInfo {
	info := agentStateInfo{
		Mode:         a.views.Active().Mode().String(),
		Project:      a.activeTabProjectPath(),
		CheckErrors:  len(a.checkErrors),
		CheckRunning: a.checkRunning,
	}
	if bv, ok := a.views.Base().(BrowserView); ok {
		info.PreviewMode = "MDL"
		if bv.miller.preview.mode == PreviewNDSL {
			info.PreviewMode = "NDSL"
		}
		if node := bv.miller.SelectedNode(); node != nil {
			qname := node.QualifiedName
			if qname == "" {
				qname = node.Label
			}
			info.SelectedNode = &agentNodeInfo{
				Type:          node.Type,
				QualifiedName: qname,
			}
		}
	}
	return info
}

// agentExecChanges is a structured summary of exec output changes.
type agentExecChange struct {
	Action string `json:"action"` // "created", "modified", "dropped"
	Target string `json:"target"` // e.g. "entity Module.Entity"
}

// agentChangePattern matches exec output lines like "Created entity MyModule.Customer".
// Requires a known Mendix type keyword after the verb to avoid matching log noise
// such as "Removed trailing whitespace".
var agentChangePattern = regexp.MustCompile(`(?im)^(Created|Modified|Dropped|Deleted|Added|Removed)\s+(entity|association|attribute|enumeration|microflow|nanoflow|page|layout|snippet|module|folder|constant|workflow|image collection|java action|user role|module role|demo user|business event service)\s+(.+)$`)

// agentParseChanges extracts structured changes from exec output.
func agentParseChanges(output string) []agentExecChange {
	matches := agentChangePattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}
	changes := make([]agentExecChange, 0, len(matches))
	for _, m := range matches {
		changes = append(changes, agentExecChange{
			Action: strings.ToLower(m[1]),
			Target: strings.ToLower(m[2]) + " " + strings.TrimSpace(m[3]),
		})
	}
	return changes
}

// Shared by BrowserView and CompareView to avoid duplicate implementations.
func loadBsonNDSL(mxcliPath, projectPath, qname, nodeType string, side CompareFocus) tea.Cmd {
	return func() tea.Msg {
		bsonType := inferBsonType(nodeType)
		if bsonType == "" {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType,
				Content: fmt.Sprintf("Error: type %q not supported for BSON dump", nodeType),
				Err:     fmt.Errorf("unsupported type")}
		}
		args := []string{"bson", "dump", "-p", projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(mxcliPath, args...)
		out = StripBanner(out)
		if err != nil {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: "Error: " + out, Err: err}
		}
		return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: HighlightNDSL(out)}
	}
}
