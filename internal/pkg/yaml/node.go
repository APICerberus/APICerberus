package yaml

// NodeKind identifies the kind of YAML node.
type NodeKind string

const (
	KindMap      NodeKind = "map"
	KindSequence NodeKind = "sequence"
	KindScalar   NodeKind = "scalar"
)

// Node is the parsed representation of a YAML value.
type Node interface {
	Kind() NodeKind
}

// NodeMap represents a YAML mapping.
type NodeMap struct {
	Entries map[string]Node
	Order   []string
}

func (n *NodeMap) Kind() NodeKind { return KindMap }

func newNodeMap() *NodeMap {
	return &NodeMap{
		Entries: make(map[string]Node),
		Order:   make([]string, 0),
	}
}

func (n *NodeMap) set(key string, value Node) {
	if _, exists := n.Entries[key]; !exists {
		n.Order = append(n.Order, key)
	}
	n.Entries[key] = value
}

// NodeSequence represents a YAML sequence.
type NodeSequence struct {
	Items []Node
}

func (n *NodeSequence) Kind() NodeKind { return KindSequence }

// NodeScalar represents a scalar string value.
type NodeScalar struct {
	Value string
}

func (n *NodeScalar) Kind() NodeKind { return KindScalar }
