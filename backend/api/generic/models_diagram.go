package generic

// DiagramLayout represents the full diagram state stored in the 'data' column
type DiagramLayout struct {
	Nodes []DiagramNode `json:"nodes"`
	Edges []DiagramEdge `json:"edges"`
}

// DiagramNode represents a node in the flow diagram
type DiagramNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position Position               `json:"position"`
	Data     map[string]interface{} `json:"data"`
	Measured *Measure               `json:"measured,omitempty"`
}

// Position represents the x,y coordinates of a node
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DiagramEdge represents a connection between nodes
type DiagramEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type,omitempty"`
	Animated bool   `json:"animated,omitempty"`
	Label    string `json:"label,omitempty"`
}

// Measure represents the dimensions of a node
type Measure struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
