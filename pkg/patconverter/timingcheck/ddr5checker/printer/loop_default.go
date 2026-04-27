package printer

import (
	"fmt"
	"strings"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/util"
)

type DefaultLoopStrategy struct{}

func NewDefaultLoopStrategy() *DefaultLoopStrategy {
	return &DefaultLoopStrategy{}
}

func (s *DefaultLoopStrategy) IsLoopOperator(op string) bool {
	return strings.HasPrefix(op, "JNI") || strings.HasPrefix(op, "IDXI")
}

func (s *DefaultLoopStrategy) ProcessLoop(ctx *LoopContext) int {
	ctx.Acc.flush()

	cmdCount := ctx.BP.countDirectCommandSubRows(ctx.Node.Children)
	wc := ctx.BP.effectiveWayCount()

	if wc > 1 && cmdCount == wc && !hasDirectChildLoop(ctx.Node.Children) {
		s.processInlineLoop(ctx)
	} else {
		s.processLabeledLoop(ctx)
	}

	return 1
}

func (s *DefaultLoopStrategy) processInlineLoop(ctx *LoopContext) {
	ctx.BP.processNodesWithLabel(ctx.Node.Children, ctx.Acc, "")

	//tags := ctx.BP.collectIncrementalTags(ctx.Node.Children)

	ctx.Acc.flush()

	if len(ctx.Acc.result) > 0 {
		lastPN := ctx.Acc.result[len(ctx.Acc.result)-1]
		jniOp := fmt.Sprintf("%s%d", ctx.BP.config.JNIKeyword, 1)
		lastPN.Header.Operator = jniOp
		lastPN.Header.OperatorValue = "."
		lastPN.Header.Interrupt = true
		lastPN.Header.LoopCount = ctx.Node.LoopCount
		lastPN.SourceLineNum = ctx.Node.LineNum

		// if len(tags) > 0 {
		// 	for i := len(lastPN.SubRows) - 1; i >= 0; i-- {
		// 		if lastPN.SubRows[i].Violation == nil {
		// 			lastPN.SubRows[i].Addr = append(lastPN.SubRows[i].Addr, tags...)
		// 			break
		// 		}
		// 	}
		// }
	}
}

func (s *DefaultLoopStrategy) processLabeledLoop(ctx *LoopContext) {
	var label string
	var childMergedLabel string

	if ctx.MergedLabel != "" {
		label = ctx.MergedLabel
		childMergedLabel = ctx.MergedLabel
	} else {
		chain := collectNestedLoopChain(ctx.Node)
		if len(chain) > 1 {
			label = ctx.BP.buildMergedLoopLabel(chain)
			childMergedLabel = label
		} else {
			label = ctx.BP.buildLoopLabel(ctx.Node)
			childMergedLabel = ""
		}
	}

	ctx.Acc.pendingLabel = label
	ctx.BP.processNodesWithLabel(ctx.Node.Children, ctx.Acc, childMergedLabel)
	ctx.Acc.flush()
	s.ResolveJNINode(ctx, label)
}

func (s *DefaultLoopStrategy) ResolveJNINode(ctx *LoopContext, label string) {
	jniOp := ctx.BP.config.JNIKeyword

	var jniNode *model.PrintNode

	if len(ctx.Acc.result) > 0 {
		lastPN := ctx.Acc.result[len(ctx.Acc.result)-1]

		if lastPN.Header.Label == "" && !s.IsLoopOperator(lastPN.Header.Operator) {
			lastPN.Header.Operator = jniOp
			lastPN.Header.OperatorValue = label
			lastPN.Header.Interrupt = true
			lastPN.SourceLineNum = ctx.Node.LineNum
			lastPN.Header.LoopCount = ctx.Node.LoopCount
			jniNode = lastPN
		}
	}

	if jniNode == nil {
		wc := ctx.BP.effectiveWayCount()

		jniNode = model.NewPrintNode(model.NodeHeader{
			Operator:      jniOp,
			OperatorValue: label,
			LoopCount:     ctx.Node.LoopCount,
		}, wc)
		jniNode.SourceLineNum = ctx.Node.LineNum

		if wc > 0 && ctx.BP.config.PadSubRow != nil {
			for i := 0; i < wc; i++ {
				pad := ctx.BP.config.PadSubRow()
				pad.Status = model.SubRowStatusAutoPad
				jniNode.SubRows = append(jniNode.SubRows, pad)
			}
		}

		ctx.Acc.result = append(ctx.Acc.result, jniNode)
	}

	// tags := ctx.BP.collectIncrementalTags(ctx.Node.Children)
	// if len(tags) > 0 && len(jniNode.SubRows) > 0 {
	// 	for i := len(jniNode.SubRows) - 1; i >= 0; i-- {
	// 		if jniNode.SubRows[i].Violation == nil {
	// 			jniNode.SubRows[i].Addr = append(jniNode.SubRows[i].Addr, tags...)
	// 			break
	// 		}
	// 	}
	// }
}

func (s *DefaultLoopStrategy) FlushAsIDXI(ctx *LoopContext, srs []model.SubRow, repeatCount int) {
	wc := ctx.BP.effectiveWayCount()
	v := util.HexValue(fmt.Sprintf("%d", repeatCount))

	pn := model.NewPrintNode(model.NodeHeader{
		Operator:      "IDXI5",
		OperatorValue: "#" + v.String(),
		Interrupt:     true,
	}, wc)

	if ctx.Acc.pendingLabel != "" {
		pn.Header.Label = ctx.Acc.pendingLabel
		ctx.Acc.pendingLabel = ""
	}

	normalCount := 0
	for _, sr := range srs {
		pn.SubRows = append(pn.SubRows, sr)
		if sr.Violation == nil {
			normalCount++
		}
	}

	ctx.BP.padSubRows(pn, normalCount)

	if len(ctx.Acc.pendingAssignTags) > 0 {
		tagLines := ctx.Acc.pendingAssignTagLines()
		log.Fatalf("assign SPECIAL_TAG (lines: %v) has no available SubRow to fill (next PrintNode is IDXI)",
			tagLines)
	}

	if len(ctx.Acc.pendingIncrementalTags) > 0 {
		log.Fatalf(
			"incremental SPECIAL_TAG (lines: %v) has no available SubRow to fill (next PrintNode is IDXI)",
			ctx.Acc.pendingIncrementalTagLines(),
		)
	}

	ctx.Acc.result = append(ctx.Acc.result, pn)
}
