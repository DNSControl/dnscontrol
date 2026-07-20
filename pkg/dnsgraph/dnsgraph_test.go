package dnsgraph_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_CreateGraph(t *testing.T) {
	changes := []testutils.StubRecord{
		{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{}},
		{NameFQDN: "mail.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}, {Type: dnsgraph.ForwardDependency, NameFQDN: "someserver.example.com"}}},
		{NameFQDN: "*.hq.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}},
		{NameFQDN: "someserver.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "a.hq.example.com"}, {Type: dnsgraph.ForwardDependency, NameFQDN: "b.hq.example.com"}}},
	}

	graph := dnsgraph.CreateGraph(testutils.StubRecordsAsGraphable(changes))

	nodes := graph.Tree.Get("example.com")
	assert.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Edges, 2)
	assert.Equal(t, "mail.example.com", nodes[0].Edges[0].Node.Data.GetName())
	assert.Equal(t, dnsgraph.IncomingEdge, nodes[0].Edges[0].Direction)
	assert.Equal(t, "*.hq.example.com", nodes[0].Edges[1].Node.Data.GetName())

	nodes = graph.Tree.Get("someserver.example.com")
	assert.Len(t, nodes, 1)
	// *.hq.example.com is added once
	assert.Len(t, nodes[0].Edges, 2)
	assert.Equal(t, "mail.example.com", nodes[0].Edges[0].Node.Data.GetName())
	assert.Equal(t, dnsgraph.IncomingEdge, nodes[0].Edges[0].Direction)
	assert.Equal(t, "*.hq.example.com", nodes[0].Edges[1].Node.Data.GetName())
	assert.Equal(t, dnsgraph.OutgoingEdge, nodes[0].Edges[1].Direction)

	nodes = graph.Tree.Get("a.hq.example.com")
	assert.Len(t, nodes, 1)
	// someserver.example.com is added once
	assert.Len(t, nodes[0].Edges, 2)
	assert.Equal(t, "example.com", nodes[0].Edges[0].Node.Data.GetName())
	assert.Equal(t, dnsgraph.OutgoingEdge, nodes[0].Edges[0].Direction)
	assert.Equal(t, "someserver.example.com", nodes[0].Edges[1].Node.Data.GetName())
	assert.Equal(t, dnsgraph.IncomingEdge, nodes[0].Edges[1].Direction)
}

func Test_RemoveNode(t *testing.T) {
	changes := []testutils.StubRecord{
		{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{}},
		{NameFQDN: "mail.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}, {Type: dnsgraph.ForwardDependency, NameFQDN: "someserver.example.com"}}},
		{NameFQDN: "*.hq.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}},
		{NameFQDN: "someserver.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "a.hq.example.com"}, {Type: dnsgraph.ForwardDependency, NameFQDN: "b.hq.example.com"}}},
	}

	graph := dnsgraph.CreateGraph(testutils.StubRecordsAsGraphable(changes))

	graph.RemoveNode(graph.Tree.Get("example.com")[0])

	// example.com change has been removed
	nodes := graph.Tree.Get("example.com")
	assert.Empty(t, nodes)

	nodes = graph.Tree.Get("a.hq.example.com")
	assert.Len(t, nodes, 1)

	assert.Len(t, nodes[0].Edges, 1)
	assert.Equal(t, "someserver.example.com", nodes[0].Edges[0].Node.Data.GetName())
}
