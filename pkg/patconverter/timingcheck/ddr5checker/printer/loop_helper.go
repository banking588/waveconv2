package printer

import (
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
)

func collectNestedLoopChain(loop *core.SeqNode) []*core.SeqNode {
	chain := []*core.SeqNode{loop}

	current := loop
	for {
		innerLoop := findFirstLoopWithoutCommand(current.Children)
		if innerLoop == nil {
			break
		}
		chain = append(chain, innerLoop)
		current = innerLoop
	}

	return chain
}

func findFirstLoopWithoutCommand(children []*core.SeqNode) *core.SeqNode {
	for _, child := range children {
		switch child.Type {
		case core.NodeCommand:
			return nil
		case core.NodeLoop:
			return child
		case core.NodeSpecialTag:
			continue
		default:
			continue
		}
	}
	return nil
}

func hasDirectChildLoop(children []*core.SeqNode) bool {
	for _, child := range children {
		if child.Type == core.NodeLoop {
			return true
		}
	}
	return false
}
