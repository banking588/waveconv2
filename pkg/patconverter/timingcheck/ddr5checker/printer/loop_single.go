package printer

import (
	"fmt"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/util"
)

type SingleLoopStrategy struct{}

func NewSingleLoopStrategy() *SingleLoopStrategy {
	return &SingleLoopStrategy{}
}

func (s *SingleLoopStrategy) IsLoopOperator(op string) bool {
	return false
}

func (s *SingleLoopStrategy) ProcessLoop(ctx *LoopContext) int {
	ctx.Acc.flush()
	s.processLabeledLoop(ctx)
	return 1
}

func (s *SingleLoopStrategy) processLabeledLoop(ctx *LoopContext) {
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

func (s *SingleLoopStrategy) ResolveJNINode(ctx *LoopContext, label string) {
	jniOp := ctx.BP.config.JNIKeyword

	var jniNode *model.PrintNode

	if len(ctx.Acc.result) > 0 {
		lastPN := ctx.Acc.result[len(ctx.Acc.result)-1]

		if lastPN.Header.Label == "" {
			lastPN.Header.Operator = jniOp
			lastPN.Header.OperatorValue = label
			lastPN.Header.Interrupt = true
			lastPN.SourceLineNum = ctx.Node.LineNum
			lastPN.Header.LoopCount = ctx.Node.LoopCount
			jniNode = lastPN
		}
	}

	if jniNode == nil {
		jniNode = model.NewPrintNode(model.NodeHeader{
			Operator:      jniOp,
			OperatorValue: label,
			LoopCount:     ctx.Node.LoopCount,
		}, 0)
		jniNode.SourceLineNum = ctx.Node.LineNum
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

func (s *SingleLoopStrategy) FlushAsIDXI(ctx *LoopContext, srs []model.SubRow, repeatCount int) {
	v := util.HexValue(fmt.Sprintf("%d", repeatCount))

	pn := model.NewPrintNode(model.NodeHeader{
		Operator:      "IDXI5",
		OperatorValue: "#" + v.String(),
		Interrupt:     true,
	}, 0)

	if ctx.Acc.pendingLabel != "" {
		pn.Header.Label = ctx.Acc.pendingLabel
		ctx.Acc.pendingLabel = ""
	}

	for _, sr := range srs {
		pn.SubRows = append(pn.SubRows, sr)
	}

	if len(ctx.Acc.pendingAssignTags) > 0 {
		tagLines := ctx.Acc.pendingAssignTagLines()
		log.Fatalf("assign SPECIAL_TAG (lines: %v) has no available SubRow to fill (next PrintNode is IDXI)",
			tagLines)
	}
	ctx.Acc.result = append(ctx.Acc.result, pn)
}
