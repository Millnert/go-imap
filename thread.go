package imap

// ThreadAlgorithm is a threading algorithm for the THREAD command (RFC 5256).
type ThreadAlgorithm string

const (
	// ThreadOrderedSubject is the ORDEREDSUBJECT threading algorithm.
	// Messages are sorted by base subject and then by sent date.
	// RFC 5256 section 3.
	ThreadOrderedSubject ThreadAlgorithm = "ORDEREDSUBJECT"

	// ThreadReferences is the REFERENCES threading algorithm.
	// Messages are threaded by References header and In-Reply-To.
	// RFC 5256 section 3.
	ThreadReferences ThreadAlgorithm = "REFERENCES"
)

// ThreadNode represents a node in the thread tree.
// A node can represent either a message or a "dummy" parent
// (when the parent message is not in the mailbox or doesn't match search criteria).
type ThreadNode struct {
	// Num is the message sequence number or UID (0 for dummy nodes).
	Num uint32

	// Children are the child nodes of this thread node.
	Children []ThreadNode
}

// ThreadData is the data returned by a THREAD command.
type ThreadData struct {
	// Threads contains the top-level thread nodes.
	Threads []ThreadNode
}

// Flatten returns all message numbers in the thread tree in depth-first order.
func (d *ThreadData) Flatten() []uint32 {
	var nums []uint32
	var flatten func(nodes []ThreadNode)
	flatten = func(nodes []ThreadNode) {
		for _, node := range nodes {
			if node.Num > 0 {
				nums = append(nums, node.Num)
			}
			flatten(node.Children)
		}
	}
	flatten(d.Threads)
	return nums
}

// Count returns the total number of messages in all threads.
func (d *ThreadData) Count() int {
	var count int
	var walk func(nodes []ThreadNode)
	walk = func(nodes []ThreadNode) {
		for _, node := range nodes {
			if node.Num > 0 {
				count++
			}
			walk(node.Children)
		}
	}
	walk(d.Threads)
	return count
}
