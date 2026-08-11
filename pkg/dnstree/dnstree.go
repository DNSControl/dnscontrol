package dnstree

import (
	"strings"
)

// Create creates a tree like structure to add arbitrary data to DNS names.
// The DomainTree splits the domain name based on the dot (.), reverses the resulting list and add all strings the tree in order.
// It has support for wildcard domain names the tree nodes (`Set`), but not during retrieval (Get and Has).
// Get always returns the most specific node; it doesn't immediately return the node upon finding a wildcard node.
func Create[T any]() *DomainTree[T] {
	return &DomainTree[T]{
		IsLeaf:     false,
		IsWildcard: false,
		Name:       "",
		Children:   map[string]*domainNode[T]{},
	}
}

// DomainTree is a domain tree.
type DomainTree[T any] domainNode[T]

type domainNode[T any] struct {
	IsLeaf     bool
	IsWildcard bool
	Name       string
	Children   map[string]*domainNode[T]
	data       T
}

func createNode[T any](name string) *domainNode[T] {
	return &domainNode[T]{
		IsLeaf:   false,
		Name:     name,
		Children: map[string]*domainNode[T]{},
	}
}

// Set adds given data to the given fqdn.
// The FQDN can contain a wildcard on the start.
// Example fqdn: *.example.com.
func (tree *DomainTree[T]) Set(fqdn string, data T) {
	domainParts := splitFQDN(fqdn)

	isWildcard := domainParts[0] == "*"
	if isWildcard {
		domainParts = domainParts[1:]
	}

	ptr := (*domainNode[T])(tree)
	for iX := len(domainParts) - 1; iX > 0; iX-- {
		ptr = ptr.addIntermediate(domainParts[iX])
	}

	ptr.addLeaf(domainParts[0], isWildcard, data)
}

// Get retrieves the attached data from a given FQDN.
// The tree will return the data entry for the most specific FQDN entry.
// If no entry is found Get will return the default value for the specific type.
//
// tree.Set("*.example.com", 1)
// tree.Set("a.example.com", 2)
// tree.Get("a.example.com") // 2
// tree.Get("a.a.example.com") // 1
// tree.Get("other.com") // 0.
func (tree *DomainTree[T]) Get(fqdn string) T {
	domainParts := splitFQDN(fqdn)

	var mostSpecificNode *domainNode[T]
	ptr := (*domainNode[T])(tree)

	for iX := len(domainParts) - 1; iX >= 0; iX-- {
		node, ok := ptr.Children[domainParts[iX]]
		if !ok {
			if mostSpecificNode != nil {
				return mostSpecificNode.data
			}
			return *new(T)
		}

		if node.IsWildcard {
			mostSpecificNode = node
		}

		ptr = node
	}

	if ptr.IsLeaf || ptr.IsWildcard {
		return ptr.data
	}

	if mostSpecificNode != nil {
		return mostSpecificNode.data
	}

	return *new(T)
}

// Has returns if the tree contains data for given FQDN.
func (tree *DomainTree[T]) Has(fqdn string) bool {
	domainParts := splitFQDN(fqdn)

	var mostSpecificNode *domainNode[T]
	ptr := (*domainNode[T])(tree)

	for iX := len(domainParts) - 1; iX >= 0; iX-- {
		node, ok := ptr.Children[domainParts[iX]]
		if !ok {
			return mostSpecificNode != nil
		}

		if node.IsWildcard {
			mostSpecificNode = node
		}

		ptr = node
	}

	return ptr.IsLeaf || ptr.IsWildcard || mostSpecificNode != nil
}

func splitFQDN(fqdn string) []string {
	normalizedFQDN := strings.TrimSuffix(fqdn, ".")

	return strings.Split(normalizedFQDN, ".")
}

func (tree *domainNode[T]) addIntermediate(name string) *domainNode[T] {
	if _, ok := tree.Children[name]; !ok {
		tree.Children[name] = createNode[T](name)
	}

	return tree.Children[name]
}

func (tree *domainNode[T]) addLeaf(name string, isWildcard bool, data T) *domainNode[T] {
	node := tree.addIntermediate(name)

	node.data = data
	node.IsLeaf = true
	node.IsWildcard = node.IsWildcard || isWildcard

	return node
}
