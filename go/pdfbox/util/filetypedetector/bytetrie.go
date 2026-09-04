package filetypedetector

// byteTrie maps byte prefixes to values.
//
// Port of org.apache.pdfbox.util.filetypedetector.ByteTrie, whose type
// parameter the port fixes to FileType: it has one instantiation, and Go
// generics would buy nothing here.
type byteTrie struct {
	root     *byteTrieNode
	maxDepth int
}

// byteTrieNode is one node of the trie.
//
// Port of ByteTrie.ByteTrieNode. Java's value is a nullable T and a node with
// no value returns null; the port keeps a bool beside the value, because
// FileType's zero is Unknown and Unknown is a value the root really holds.
type byteTrieNode struct {
	children map[byte]*byteTrieNode
	value    FileType
	hasValue bool
}

func newByteTrieNode() *byteTrieNode {
	return &byteTrieNode{children: map[byte]*byteTrieNode{}}
}

// setValue records the value of a node.
//
// Java throws IllegalStateException where a node already has one, which is
// unchecked, so the port panics.
func (n *byteTrieNode) setValue(value FileType) {
	if n.hasValue {
		panic("Value already set for this trie node")
	}
	n.value = value
	n.hasValue = true
}

func newByteTrie() *byteTrie {
	return &byteTrie{root: newByteTrieNode()}
}

// find returns the value of the deepest node the bytes reach that has one.
func (t *byteTrie) find(bytes []byte) FileType {
	node := t.root
	val := node.value
	for _, b := range bytes {
		child := node.children[b]
		if child == nil {
			break
		}
		node = child
		if node.hasValue {
			val = node.value
		}
	}
	return val
}

// addPath adds one signature, given in as many parts as the caller likes.
func (t *byteTrie) addPath(value FileType, parts ...[]byte) {
	depth := 0
	node := t.root
	for _, part := range parts {
		for _, b := range part {
			child := node.children[b]
			if child == nil {
				child = newByteTrieNode()
				node.children[b] = child
			}
			node = child
			depth++
		}
	}
	node.setValue(value)
	if depth > t.maxDepth {
		t.maxDepth = depth
	}
}

func (t *byteTrie) setDefaultValue(defaultValue FileType) {
	t.root.setValue(defaultValue)
}

func (t *byteTrie) getMaxDepth() int { return t.maxDepth }
