package dnsgraph

import "github.com/DNSControl/dnscontrol/v4/pkg/dnstree"

type edgeDirection uint8

const (
	// IncomingEdge is an edge inbound.
	IncomingEdge edgeDirection = iota
	// OutgoingEdge is an edge outbound.
	OutgoingEdge
)

// Edge an edge on the graph.
type Edge[T Graphable] struct {
	Dependency Dependency
	Node       *Node[T]
	Direction  edgeDirection
}

// Edges a list of edges.
type Edges[T Graphable] []Edge[T]

// Node a node in the graph.
type Node[T Graphable] struct {
	Data  T
	Edges Edges[T]
}

type dnsGraphNodes[T Graphable] []*Node[T]

// Graph a graph.
type Graph[T Graphable] struct {
	All  dnsGraphNodes[T]
	Tree *dnstree.DomainTree[dnsGraphNodes[T]]
}

// CreateGraph returns a graph.
func CreateGraph[T Graphable](entries []T) *Graph[T] {
	graph := &Graph[T]{
		All:  dnsGraphNodes[T]{},
		Tree: dnstree.Create[dnsGraphNodes[T]](),
	}

	for _, data := range entries {
		graph.AddNode(data)
	}

	for _, sourceNode := range graph.All {
		for _, dependency := range sourceNode.Data.GetDependencies() {
			graph.AddEdge(sourceNode, dependency)
		}
	}

	return graph
}

// RemoveNode removes a node from a graph.
func (graph *Graph[T]) RemoveNode(toRemove *Node[T]) {
	for _, edge := range toRemove.Edges {
		edge.Node.Edges = edge.Node.Edges.RemoveNode(toRemove)
	}

	graph.All = graph.All.RemoveNode(toRemove)

	nodes := graph.Tree.Get(toRemove.Data.GetName())
	if nodes != nil {
		nodes = nodes.RemoveNode(toRemove)
		graph.Tree.Set(toRemove.Data.GetName(), nodes)
	}
}

// AddNode adds a node to a graph.
func (graph *Graph[T]) AddNode(data T) {
	nodes := graph.Tree.Get(data.GetName())
	node := &Node[T]{
		Data:  data,
		Edges: Edges[T]{},
	}
	if nodes == nil {
		nodes = dnsGraphNodes[T]{}
	}
	nodes = append(nodes, node)

	graph.All = append(graph.All, node)
	graph.Tree.Set(data.GetName(), nodes)
}

// AddEdge adds an edge to a graph.
func (graph *Graph[T]) AddEdge(sourceNode *Node[T], dependency Dependency) {
	destinationNodes := graph.Tree.Get(dependency.NameFQDN)

	if destinationNodes == nil {
		return
	}

	for _, destinationNode := range destinationNodes {
		if sourceNode == destinationNode {
			continue
		}

		if sourceNode.Edges.Contains(destinationNode, OutgoingEdge) {
			continue
		}

		sourceNode.Edges = append(sourceNode.Edges, Edge[T]{
			Dependency: dependency,
			Node:       destinationNode,
			Direction:  OutgoingEdge,
		})

		destinationNode.Edges = append(destinationNode.Edges, Edge[T]{
			Dependency: dependency,
			Node:       sourceNode,
			Direction:  IncomingEdge,
		})
	}
}

// RemoveNode removes a node from a graph.
func (nodes dnsGraphNodes[T]) RemoveNode(toRemove *Node[T]) dnsGraphNodes[T] {
	var newNodes dnsGraphNodes[T]

	for _, node := range nodes {
		if node != toRemove {
			newNodes = append(newNodes, node)
		}
	}

	return newNodes
}

// RemoveNode removes a node from a graph.
func (edges Edges[T]) RemoveNode(toRemove *Node[T]) Edges[T] {
	var newEdges Edges[T]

	for _, edge := range edges {
		if edge.Node != toRemove {
			newEdges = append(newEdges, edge)
		}
	}

	return newEdges
}

// Contains returns true if a node is in the graph AND is in that direction.
func (edges Edges[T]) Contains(toFind *Node[T], direction edgeDirection) bool {
	for _, edge := range edges {
		if edge.Node == toFind && edge.Direction == direction {
			return true
		}
	}

	return false
}
