package printer

import (
	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

func (bp *basePrinter) Convert(nodes []*core.SeqNode) []*model.PrintNode {
	bp.checkResult = nil
	return bp.doConvert(nodes)
}

func (bp *basePrinter) ConvertWithCheck(nodes []*core.SeqNode, checkResult *model.CheckResult) []*model.PrintNode {
	bp.checkResult = checkResult
	return bp.doConvert(nodes)
}

func (bp *basePrinter) doConvert(nodes []*core.SeqNode) []*model.PrintNode {
	acc := &accumulator{bp: bp}
	bp.processNodes(nodes, acc)
	acc.flush()
	return acc.result
}

func (bp *basePrinter) processNodes(nodes []*core.SeqNode, acc *accumulator) {
	bp.processNodesWithLabel(nodes, acc, "")

	if len(acc.pendingAssignTags) > 0 {
		tagLines := acc.pendingAssignTagLines()
		log.Fatalf("assign SPECIAL_TAG (lines: %v) has no following SubRow to fill", tagLines)
	}

	if len(acc.pendingIncrementalTags) > 0 {
		log.Fatalf("incremental SPECIAL_TAG (lines: %v) has no following SubRow to fill",
			acc.pendingIncrementalTagLines())
	}
}

func (bp *basePrinter) processNodesWithLabel(nodes []*core.SeqNode, acc *accumulator, mergedLabel string) {
	i := 0
	cmdIndex := 0

	for i < len(nodes) {
		node := nodes[i]

		switch node.Type {
		case core.NodeCommand:
			i, cmdIndex = bp.printerStrategy.ProcessCommand(nodes, bp, acc, i, cmdIndex)

		case core.NodeLoop:
			i = bp.printerStrategy.ProcessLoop(nodes, bp, acc, i, mergedLabel)

		case core.NodeSpecialTag:
			i = bp.processSpecialTag(nodes, acc, i)

		default:
			i++
		}
	}
}

func (bp *basePrinter) processSpecialTag(nodes []*core.SeqNode, acc *accumulator, i int) int {
	node := nodes[i]
	// 累加型
	if core.IsRelativeType(node.SpecialTagOpType) {
		var incTags []pendingAssignTag
		for i < len(nodes) && nodes[i].Type == core.NodeSpecialTag {
			if !core.IsRelativeType(nodes[i].SpecialTagOpType) {
				break
			}
			addrText := bp.config.ExpressionConv(nodes[i].ExprString)
			if addrText != "" {
				incTags = append(incTags, pendingAssignTag{
					addr:    addrText,
					lineNum: nodes[i].LineNum,
				})
			}
			i++
		}

		if len(incTags) == 0 {
			return i
		}

		// 偵測下一個非 SpecialTag 節點型別
		nextIsLoop := false
		for j := i; j < len(nodes); j++ {
			switch nodes[j].Type {
			case core.NodeCommand:
				nextIsLoop = false
				goto incDecided
			case core.NodeLoop:
				nextIsLoop = true
				goto incDecided
			case core.NodeSpecialTag:
				continue
			default:
				continue
			}
		}
	incDecided:

		if nextIsLoop {
			acc.flush()

			if len(acc.result) > 0 {
				lastPN := acc.result[len(acc.result)-1]
				for si := len(lastPN.SubRows) - 1; si >= 0; si-- {
					if lastPN.SubRows[si].Violation == nil {
						for _, t := range incTags {
							lastPN.SubRows[si].Addr = append(lastPN.SubRows[si].Addr, t.addr)
						}
						break
					}
				}
			} else {
				tagLines := make([]int, len(incTags))
				for ti, t := range incTags {
					tagLines[ti] = t.lineNum
				}
				log.Fatalf("incremental SPECIAL_TAG (lines: %v) has no available SubRow to fill", tagLines)
			}
		} else {
			acc.pendingIncrementalTags = append(acc.pendingIncrementalTags, incTags...)
		}

		return i
	}

	var assignTags []pendingAssignTag
	for i < len(nodes) && nodes[i].Type == core.NodeSpecialTag {
		// if nodes[i].IsIncrementalTag {
		// 	break
		// }
		if core.IsRelativeType(nodes[i].SpecialTagOpType) {
			break
		}
		addrText := bp.config.ExpressionConv(nodes[i].ExprString)
		if addrText != "" {
			assignTags = append(assignTags, pendingAssignTag{
				addr:    addrText,
				lineNum: nodes[i].LineNum,
			})
		}
		i++
	}

	if len(assignTags) == 0 {
		return i
	}

	nextIsLoop := false
	for j := i; j < len(nodes); j++ {
		switch nodes[j].Type {
		case core.NodeCommand:
			nextIsLoop = false
			goto decided
		case core.NodeLoop:
			nextIsLoop = true
			goto decided
		case core.NodeSpecialTag:
			continue
		default:
			continue
		}
	}
decided:

	if nextIsLoop {
		acc.flush()

		if len(acc.result) > 0 {
			lastPN := acc.result[len(acc.result)-1]
			for si := len(lastPN.SubRows) - 1; si >= 0; si-- {
				if lastPN.SubRows[si].Violation == nil {
					for _, t := range assignTags {
						lastPN.SubRows[si].Addr = append(lastPN.SubRows[si].Addr, t.addr)
					}
					break
				}
			}
		} else {
			tagLines := make([]int, len(assignTags))
			for ti, t := range assignTags {
				tagLines[ti] = t.lineNum
			}
			log.Fatalf("assign SPECIAL_TAG (lines: %v) has no available SubRow to fill", tagLines)
		}
	} else {
		acc.pendingAssignTags = append(acc.pendingAssignTags, assignTags...)
	}

	return i
}
