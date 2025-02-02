package client

import "fmt"

// create subject strings for various types of messages

// SubjectNodePoints constructs a NATS subject for node points
func SubjectNodePoints(nodeID string) string {
	return fmt.Sprintf("node.%v.points", nodeID)
}

// SubjectEdgePoints constructs a NATS subject for edge points
func SubjectEdgePoints(nodeID, parentID string) string {
	return fmt.Sprintf("node.%v.%v.points", nodeID, parentID)
}

// SubjectNodeAllPoints provides subject for all points for any node
func SubjectNodeAllPoints() string {
	return "node.*.points"
}

// SubjectEdgeAllPoints provides subject for all edge points for any node
func SubjectEdgeAllPoints() string {
	return "node.*.*.points"
}

// SubjectNodeHRPoints constructs a NATS subject for high rate node points
func SubjectNodeHRPoints(nodeID string) string {
	return fmt.Sprintf("phr.%v", nodeID)
}
