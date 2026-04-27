package printer

import (
	"fmt"
	"strings"

	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	hs2ddr5 "waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller/hs5503/ddr5"
	m5ddr5 "waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller/m5ssv/ddr5"
	t5833ddr5 "waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller/t5833/ddr5"
	t5833lpddr5 "waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller/t5833/lpddr5"
)

type wrappedPrinter struct {
	*basePrinter
	formatter model.Formatter
}

func (w *wrappedPrinter) Name() string                           { return w.formatter.Name() }
func (w *wrappedPrinter) Format(nodes []*model.PrintNode) string { return w.formatter.Format(nodes) }
func (w *wrappedPrinter) GetVarMapper() model.VarMapper          { return w.formatter.GetVarMapper() }

type basePrinter struct {
	config          *model.Config
	checkResult     *model.CheckResult
	wayCount        int
	printerStrategy PrinterStrategy
}

func (bp *basePrinter) effectiveWayCount() int {
	if bp.wayCount > 0 {
		return bp.wayCount
	}
	return bp.config.WayCount
}

func (w *wrappedPrinter) SetMRS(mrs mrstatusreg.MRS) {
	w.formatter.SetMRS(mrs)
}

func (bp *basePrinter) GetConfig() *model.Config { return bp.config }

func (bp *basePrinter) SetWayCount(n int) {
	bp.wayCount = n
	if bp.effectiveWayCount() > 1 {
		bp.printerStrategy = NewMultiWayStrategy()
	} else {
		bp.printerStrategy = NewSingleWayStrategy()
	}
}

func (bp *basePrinter) SetLoopStrategy(s LoopStrategy) {
	bp.printerStrategy.SetLoopStrategy(s)
}

// --- Label ---

func (bp *basePrinter) buildLoopLabel(loopNode *core.SeqNode) string {
	if loopNode.LoopName != "" {
		return bp.config.LabelPrefix + loopNode.LoopName
	}
	return fmt.Sprintf("%sL%d", bp.config.LabelPrefix, loopNode.LineNum)
}

func (bp *basePrinter) buildMergedLoopLabel(chain []*core.SeqNode) string {
	var names []string
	for _, l := range chain {
		if l.LoopName != "" {
			names = append(names, l.LoopName)
		} else {
			names = append(names, fmt.Sprintf("L%d", l.LineNum))
		}
	}
	return bp.config.LabelPrefix + strings.Join(names, "_")
}

// --- Pad ---

func (bp *basePrinter) padSubRows(pn *model.PrintNode, normalCount int) {
	if !bp.printerStrategy.ShouldPad() {
		return
	}
	wc := bp.effectiveWayCount()
	if wc <= 0 || bp.config.PadSubRow == nil {
		return
	}
	for normalCount < wc {
		pad := bp.config.PadSubRow()
		pad.Status = model.SubRowStatusAutoPad
		pn.SubRows = append(pn.SubRows, pad)
		normalCount++
	}
}

// --- IncrementalTags ---

func (bp *basePrinter) collectIncrementalTags(children []*core.SeqNode) []string {
	var addrs []string
	for _, child := range children {
		// if child.Type != core.NodeSpecialTag || !child.IsIncrementalTag {
		// 	continue
		// }
		if child.Type != core.NodeSpecialTag || !core.IsRelativeType(child.SpecialTagOpType) {
			continue
		}
		addrText := bp.config.ExpressionConv(child.ExprString)
		if addrText != "" {
			addrs = append(addrs, addrText)
		}
	}
	return addrs
}

// --- CountDirectCommandSubRows ---

func (bp *basePrinter) countDirectCommandSubRows(children []*core.SeqNode) int {
	count := 0
	for _, child := range children {
		if child.Type != core.NodeCommand {
			continue
		}
		srs := bp.config.RowMapCmd(child.Cmd, bp.config.WayCount)
		if len(srs) > 0 && srs[0].RepeatCnt > 1 {
			continue
		}
		for _, sr := range srs {
			if sr.Violation == nil {
				count++
			}
		}
	}
	return count
}

// --- 建構 ---

func WrapPrinter(f model.Formatter) Printer {
	cfg := f.GetConfig()
	bp := &basePrinter{
		config: cfg,
	}
	if cfg.WayCount > 1 {
		bp.printerStrategy = NewMultiWayStrategy()
	} else {
		bp.printerStrategy = NewSingleWayStrategy()
	}
	return &wrappedPrinter{
		basePrinter: bp,
		formatter:   f,
	}
}

// --- 工廠 ---

func New(name model.TesterType, protocol model.Protocol) (Printer, error) {
	return NewWithWay(name, 0, protocol)
}

func NewWithWay(name model.TesterType, wayCount int, protocol model.Protocol) (Printer, error) {
	//machine := strings.ToUpper(string(name))

	switch name {
	case model.Tester5503HS:
		switch protocol {
		case model.ProtocolDDR5:
			return WrapPrinter(hs2ddr5.NewHS5503FormatterWithWay(wayCount)), nil
		case model.ProtocolLPDDR5:
			//return WrapPrinter(hs2lpddr5.NewHS5503LPFormatterWithWay(wayCount)), nil
			return nil, nil
		default:
			return nil, fmt.Errorf("printer %q does not support protocol %q", name, protocol)
		}

	case model.TesterM5SSV:
		switch protocol {
		case model.ProtocolDDR5:
			return WrapPrinter(m5ddr5.NewM5SSVFormatterWithWay(wayCount)), nil
		case model.ProtocolLPDDR5:
			//return WrapPrinter(m5lpddr5.NewM5SSVLPFormatterWithWay(wayCount)), nil
			return nil, nil
		default:
			return nil, fmt.Errorf("printer %q does not support protocol %q", name, protocol)
		}

	case model.TesterT5833:
		switch protocol {
		case model.ProtocolDDR5:
			return WrapPrinter(t5833ddr5.NewT5833FormatterWithWay(wayCount)), nil
		case model.ProtocolLPDDR5:
			return WrapPrinter(t5833lpddr5.NewT5833FormatterWithWay(wayCount)), nil
		default:
			return nil, fmt.Errorf("printer %q does not support protocol %q", name, protocol)
		}

	default:
		return nil, fmt.Errorf("unknown printer: %q (supported: 5503HS, M5SSV, T5833)", name)
	}
}
