package ds

import (
	"github.com/15mga/kiwi/util"
)

const (
	red   = true
	black = false
)

type RBTreeNode[KT comparable, VT any] struct {
	Key    KT
	Value  VT
	color  bool
	left   *RBTreeNode[KT, VT]
	right  *RBTreeNode[KT, VT]
	parent *RBTreeNode[KT, VT]
}

func (n *RBTreeNode[KT, VT]) isRed() bool {
	if n == nil {
		return false
	}
	return n.color
}

func (n *RBTreeNode[KT, VT]) grandparent() *RBTreeNode[KT, VT] {
	if n != nil && n.parent != nil {
		return n.parent.parent
	}
	return nil
}

func (n *RBTreeNode[KT, VT]) uncle() *RBTreeNode[KT, VT] {
	if n == nil || n.parent == nil || n.parent.parent == nil {
		return nil
	}
	return n.parent.sibling()
}

func (n *RBTreeNode[KT, VT]) maximumNode() *RBTreeNode[KT, VT] {
	if n == nil {
		return nil
	}
	for n.right != nil {
		return n.right
	}
	return n
}

func (n *RBTreeNode[KT, VT]) sibling() *RBTreeNode[KT, VT] {
	if n == nil || n.parent == nil {
		return nil
	}
	if n == n.parent.left {
		return n.parent.right
	}
	return n.parent.left
}

type RBTree[KT comparable, VT any] struct {
	root    *RBTreeNode[KT, VT]
	size    int
	compare util.Compare[KT]
}

func NewRBTree[KT comparable, VT any](compare util.Compare[KT]) *RBTree[KT, VT] {
	return &RBTree[KT, VT]{
		compare: compare,
	}
}

func NewRBTreeFromM[KT comparable, VT any](m map[KT]VT, compare util.Compare[KT]) *RBTree[KT, VT] {
	tree := NewRBTree[KT, VT](compare)
	for k, v := range m {
		tree.Set(k, v)
	}
	return tree
}

func (t *RBTree[KT, VT]) Set(k KT, v VT) {
	var nn *RBTreeNode[KT, VT]
	if t.root == nil {
		nn = &RBTreeNode[KT, VT]{
			Key:   k,
			Value: v,
			color: red,
		}
		t.root = nn
	} else {
		ok := true
		n := t.root
		for ok {
			cpr := t.compare(k, n.Key)
			switch {
			case cpr == 0:
				n.Value = v
				return
			case cpr < 0:
				if n.left == nil {
					nn = &RBTreeNode[KT, VT]{
						Key:   k,
						Value: v,
						color: true,
					}
					n.left = nn
					ok = false
				} else {
					n = n.left
				}
			case cpr > 0:
				if n.right == nil {
					nn = &RBTreeNode[KT, VT]{
						Key:   k,
						Value: v,
						color: red,
					}
					n.right = nn
					ok = false
				} else {
					n = n.right
				}
			}
		}
		nn.parent = n
	}
	t.insert1(nn)
	t.size++
}

func (t *RBTree[KT, VT]) Reset() {
	t.root = nil
	t.size = 0
}

func (t *RBTree[KT, VT]) Update(m map[KT]VT) {
	for k, v := range m {
		t.Set(k, v)
	}
}

func (t *RBTree[KT, VT]) Get(k KT) (VT, bool) {
	node, ok := t.getNode(k)
	if ok {
		return node.Value, true
	}
	return util.Default[VT](), false
}

func (t *RBTree[KT, VT]) Del(key KT) (value VT) {
	child := (*RBTreeNode[KT, VT])(nil)
	node, ok := t.getNode(key)
	if !ok {
		return
	}
	value = node.Value
	if node.left != nil && node.right != nil {
		p := node.left.maximumNode()
		node.Key = p.Key
		node.Value = p.Value
		node = p
	}
	if node.left == nil || node.right == nil {
		if node.right == nil {
			child = node.left
		} else {
			child = node.right
		}
		if !node.color {
			node.color = child.isRed()
			t.deleteCase1(node)
		}
		t.updateNode(node, child)
		if node.parent == nil && child != nil {
			child.color = black
		}
	}
	t.size--
	return
}

func (t *RBTree[KT, VT]) AnyAsc(fn func(KT, VT) bool) bool {
	return t.anyAsc(t.leftNode(), fn)
}

func (t *RBTree[KT, VT]) AnyAscFrom(k KT, fn func(KT, VT) bool) (match bool, ok bool) {
	node, ok := t.getNode(k)
	if !ok {
		return
	}
	match = true
	ok = t.anyAsc(node, fn)
	return
}

func (t *RBTree[KT, VT]) AnyDesc(fn func(KT, VT) bool) bool {
	return t.anyAsc(t.rightNode(), fn)
}

func (t *RBTree[KT, VT]) AnyDescFrom(k KT, fn func(KT, VT) bool) (match bool, ok bool) {
	node, ok := t.getNode(k)
	if !ok {
		return
	}
	match = true
	ok = t.anyDesc(node, fn)
	return
}

func (t *RBTree[KT, VT]) M() map[KT]VT {
	m := make(map[KT]VT, t.size)
	_ = t.AnyAsc(func(k KT, v VT) bool {
		m[k] = v
		return false
	})
	return m
}

func (t *RBTree[KT, VT]) getNode(k KT) (node *RBTreeNode[KT, VT], ok bool) {
	node = t.root
	for node != nil {
		cpr := t.compare(k, node.Key)
		switch {
		case cpr == 0:
			return node, true
		case cpr < 0:
			node = node.left
		case cpr > 0:
			node = node.right
		}
	}
	return node, false
}

func (t *RBTree[KT, VT]) leftNode() *RBTreeNode[KT, VT] {
	p := (*RBTreeNode[KT, VT])(nil)
	n := t.root
	for n != nil {
		p = n
		n = n.left
	}
	return p
}

// rightNode returns the right-most (max) node or nil if tree is empty.
func (t *RBTree[KT, VT]) rightNode() *RBTreeNode[KT, VT] {
	p := (*RBTreeNode[KT, VT])(nil)
	n := t.root
	for n != nil {
		p = n
		n = n.right
	}
	return p
}

func (t *RBTree[KT, VT]) insert1(node *RBTreeNode[KT, VT]) {
	if node.parent == nil {
		node.color = black
	} else {
		t.insert2(node)
	}
}

func (t *RBTree[KT, VT]) insert2(node *RBTreeNode[KT, VT]) {
	if !node.parent.isRed() {
		return
	}
	t.insert3(node)
}

func (t *RBTree[KT, VT]) insert3(node *RBTreeNode[KT, VT]) {
	uncle := node.uncle()
	if uncle.isRed() {
		node.parent.color = black
		uncle.color = black
		node.grandparent().color = red
		t.insert1(node.grandparent())
	} else {
		t.insert4(node)
	}
}

func (t *RBTree[KT, VT]) insert4(node *RBTreeNode[KT, VT]) {
	grandparent := node.grandparent()
	if node == node.parent.right && node.parent == grandparent.left {
		t.rotateLeft(node.parent)
		node = node.left
	} else if node == node.parent.left && node.parent == grandparent.right {
		t.rotateRight(node.parent)
		node = node.right
	}
	t.insert5(node)
}

func (t *RBTree[KT, VT]) insert5(node *RBTreeNode[KT, VT]) {
	node.parent.color = black
	grandparent := node.grandparent()
	grandparent.color = red
	if node == node.parent.left && node.parent == grandparent.left {
		t.rotateRight(grandparent)
	} else if node == node.parent.right && node.parent == grandparent.right {
		t.rotateLeft(grandparent)
	}
}

func (t *RBTree[KT, VT]) deleteCase1(node *RBTreeNode[KT, VT]) {
	if node.parent == nil {
		return
	}
	t.deleteCase2(node)
}

func (t *RBTree[KT, VT]) deleteCase2(node *RBTreeNode[KT, VT]) {
	sibling := node.sibling()
	if sibling.isRed() {
		node.parent.color = red
		sibling.color = black
		if node == node.parent.left {
			t.rotateLeft(node.parent)
		} else {
			t.rotateRight(node.parent)
		}
	}
	t.deleteCase3(node)
}

func (t *RBTree[KT, VT]) deleteCase3(node *RBTreeNode[KT, VT]) {
	sibling := node.sibling()
	if node.parent.isRed() ||
		sibling.isRed() ||
		sibling.left.isRed() ||
		sibling.right.isRed() {
		t.deleteCase4(node)
	} else {
		sibling.color = red
		t.deleteCase1(node.parent)
	}
}

func (t *RBTree[KT, VT]) deleteCase4(node *RBTreeNode[KT, VT]) {
	sibling := node.sibling()
	if !node.parent.isRed() ||
		sibling.isRed() ||
		sibling.left.isRed() ||
		sibling.right.isRed() {
		t.deleteCase5(node)
	} else {
		sibling.color = red
		node.parent.color = black
	}
}

func (t *RBTree[KT, VT]) deleteCase5(node *RBTreeNode[KT, VT]) {
	sibling := node.sibling()
	if node == node.parent.left &&
		!sibling.isRed() &&
		sibling.left.isRed() &&
		!sibling.right.isRed() {
		sibling.color = red
		sibling.left.color = black
		t.rotateRight(sibling)
	} else if node == node.parent.right &&
		!sibling.isRed() &&
		sibling.right.isRed() &&
		!sibling.left.isRed() {
		sibling.color = red
		sibling.right.color = black
		t.rotateLeft(sibling)
	}
	t.deleteCase6(node)
}

func (t *RBTree[KT, VT]) deleteCase6(node *RBTreeNode[KT, VT]) {
	sibling := node.sibling()
	sibling.color = node.parent.isRed()
	node.parent.color = black
	if node == node.parent.left && sibling.right.isRed() {
		sibling.right.color = black
		t.rotateLeft(node.parent)
	} else if sibling.left.isRed() {
		sibling.left.color = black
		t.rotateRight(node.parent)
	}
}

func (t *RBTree[KT, VT]) rotateLeft(node *RBTreeNode[KT, VT]) {
	right := node.right
	t.updateNode(node, right)
	node.right = right.left
	if right.left != nil {
		right.left.parent = node
	}
	right.left = node
	node.parent = right
}

func (t *RBTree[KT, VT]) rotateRight(node *RBTreeNode[KT, VT]) {
	left := node.left
	t.updateNode(node, left)
	node.left = left.right
	if left.right != nil {
		left.right.parent = node
	}
	left.right = node
	node.parent = left
}

func (t *RBTree[KT, VT]) updateNode(old *RBTreeNode[KT, VT], new *RBTreeNode[KT, VT]) {
	if old.parent == nil {
		t.root = new
	} else {
		if old == old.parent.left {
			old.parent.left = new
		} else {
			old.parent.right = new
		}
	}
	if new != nil {
		new.parent = old.parent
	}
}

func (t *RBTree[KT, VT]) anyAsc(node *RBTreeNode[KT, VT], f func(KT, VT) bool) bool {
loop:
	if node == nil {
		return false
	}
	if f(node.Key, node.Value) {
		return true
	}
	if node.right != nil {
		node = node.right
		for node.left != nil {
			node = node.left
		}
		goto loop
	}
	if node.parent != nil {
		old := node
		for node.parent != nil {
			node = node.parent
			if t.compare(old.Key, node.Key) <= 0 {
				goto loop
			}
		}
	}
	return false
}

func (t *RBTree[KT, VT]) anyDesc(node *RBTreeNode[KT, VT], f func(KT, VT) bool) bool {
loop:
	if node == nil {
		return false
	}
	if f(node.Key, node.Value) {
		return true
	}
	if node.left != nil {
		node = node.left
		for node.right != nil {
			node = node.right
		}
		goto loop
	}
	if node.parent != nil {
		old := node
		for node.parent != nil {
			node = node.parent
			if t.compare(old.Key, node.Key) >= 0 {
				goto loop
			}
		}
	}
	return false
}
