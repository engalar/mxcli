package mxgraph

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func init() {
	gob.Register(&Node{})
	gob.Register(&Edge{})
	gob.Register(map[string]any{})
	gob.Register([]any{})
	gob.Register(NodeID(""))
	gob.Register(element.ID(""))
	gob.Register(bson.Binary{})
	gob.Register(bson.ObjectID{})
}

// ── Snapshot ────────────────────────────────────────────────

type graphSnapshot struct {
	Version int64
	Nodes   []*Node
	Edges   []*Edge
}

func MarshalSnapshot(g *Graph) ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	snap := graphSnapshot{
		Version: 1,
		Nodes:   make([]*Node, 0, len(g.nodes)),
		Edges:   make([]*Edge, 0, len(g.edges)),
	}
	for _, n := range g.nodes {
		snap.Nodes = append(snap.Nodes, n)
	}
	for _, e := range g.edges {
		snap.Edges = append(snap.Edges, e)
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(snap); err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

func UnmarshalSnapshot(data []byte) (*Graph, error) {
	var snap graphSnapshot
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&snap); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}

	g := New()
	for _, n := range snap.Nodes {
		g.AddNode(n.ID, n.Label, n.Props)
	}
	for _, e := range snap.Edges {
		g.AddEdge(e.ID, e.From, e.To, e.Type, e.Props)
	}
	return g, nil
}

// ── Delta Log ─────────────────────────────────────────────

type DeltaWriter struct {
	w io.Writer
}

func NewDeltaWriter(w io.Writer) *DeltaWriter {
	return &DeltaWriter{w: w}
}

func (dw *DeltaWriter) WriteEvent(ev Event) error {
	var body bytes.Buffer
	enc := gob.NewEncoder(&body)
	if err := enc.Encode(ev); err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	recordType := byte(ev.Type + 1)
	b := body.Bytes()
	header := make([]byte, 5)
	header[0] = recordType
	binary.LittleEndian.PutUint32(header[1:5], uint32(len(b)))
	if _, err := dw.w.Write(header); err != nil {
		return err
	}
	_, err := dw.w.Write(b)
	return err
}

func (dw *DeltaWriter) WriteCheckpoint() error {
	header := make([]byte, 5)
	header[0] = 0xFF
	binary.LittleEndian.PutUint32(header[1:5], 0)
	_, err := dw.w.Write(header)
	return err
}

func (dw *DeltaWriter) Close() error {
	if c, ok := dw.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

type DeltaReader struct {
	r io.Reader
}

func NewDeltaReader(r io.Reader) *DeltaReader {
	return &DeltaReader{r: r}
}

func (dr *DeltaReader) ReadEvent() (Event, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(dr.r, header); err != nil {
		return Event{}, err
	}
	recordType := header[0]
	length := binary.LittleEndian.Uint32(header[1:5])
	if recordType == 0xFF {
		return Event{}, io.EOF
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(dr.r, body); err != nil {
		return Event{}, err
	}
	var ev Event
	dec := gob.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&ev); err != nil {
		return Event{}, fmt.Errorf("decode event: %w", err)
	}
	return ev, nil
}
