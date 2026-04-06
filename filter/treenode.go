// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"fmt"
	"strings"
)

type TreeNode struct {
	Value  Token
	parent *TreeNode
	items  []*TreeNode
}

// NewTreeElement Creates a new TreeElement.
func NewTreeNode(value Token) *TreeNode {
	return &TreeNode{value, nil, make([]*TreeNode, 0)}
}

// Parent Returns the current element parent
func (n *TreeNode) Parent() *TreeNode {
	return n.parent
}

func (n *TreeNode) Root() *TreeNode {
	p := n
	for p.parent != nil {
		p = p.parent
	}
	return p
}

// setParent sets the current nodes parent value.
// Warning: does not add the node as a child
func (n *TreeNode) setParent(element *TreeNode) {
	if n.parent != nil {
		panic("TreeNode already attached to a parent node")
	}
	n.parent = element
}

func (n *TreeNode) LastElement() *TreeNode {
	if len(n.items) == 0 {
		return nil
	}
	return n.items[len(n.items)-1]
}

func (n *TreeNode) Last() Token {
	last := n.LastElement()
	if last != nil {
		return last.Value
	}
	return nil
}

func (n *TreeNode) Items() []*TreeNode {
	return n.items
}

// Add adds a TreeElement to the end of the children items of the current node.
func (n *TreeNode) AddElement(element *TreeNode) *TreeNode {
	element.setParent(n)
	n.items = append(n.items, element)
	return element
}

// Add adds a value to the end of the children items of the current node.
func (n *TreeNode) Add(value Token) *TreeNode {
	element := NewTreeNode(value)
	return n.AddElement(element)
}

// Push, removes the current element from its current parent, place the new value
// in its place and add the current element to the new element. there by pushing the current
// element down the hierachy.
// Example:
// tree:  A(B)
// B.Push(C)
// tree:  A(C(B))
func (n *TreeNode) PushElement(element *TreeNode) *TreeNode {
	parent := n.Parent()
	if parent != nil {
		//replace the current node with the new node
		index := parent.indexOf(n)
		parent.items[index] = element
		element.setParent(parent)
		n.parent = nil
	}
	//add the current node to the new node
	element.AddElement(n)
	return element
}

func (n *TreeNode) Push(value Token) *TreeNode {
	return n.PushElement(NewTreeNode(value))
}

// FindChildElement Finds a child element in the current nodes children
func (n *TreeNode) indexOf(element *TreeNode) int {
	for i, v := range n.items {
		if v == element {
			return i
		}
	}
	return -1
}

func (n *TreeNode) StringContent() string {
	lines := make([]string, len(n.items))
	for i, v := range n.items {
		lines[i] = v.String()
	}
	if n.Value.Error() != nil {
		return fmt.Sprintf("[ERROR: %s]", n.Value.Error())
	} else if len(lines) > 0 {
		return strings.Join(lines, ",")
	} else {
		return ""
	}
}

func (n *TreeNode) String() string {
	if n.StringContent() == "" {
		return n.Value.String()
	}
	return fmt.Sprintf("[%s:%s]", n.Value.String(), n.StringContent())
}
