package ddr5

import (
	"fmt"

	"innotron.com/common/log"
	"innotron.com/waveconv/pkg/patconverter/convlib/ddr5conv/statusreg/mrstatusreg"
	"innotron.com/waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"innotron.com/waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller"
)

// 5503HS CommandWriter
// 不用的 cmd.Type 產生 各自的tokens
// 可以換成：往上 N 個、往下 N 個、跨 PN 時停止/改寫等
// 把Commnader拆出来是因为, 可能同一种机台会有多种不同的macrodef写法
// 因应这种情况, 所以把它从printer中独立出来, 方便各个产品用不同的maco实作, 互相解耦
type T5833CommandWriter struct {
	Mrs *mrstatusreg.MRS
}

func (w T5833CommandWriter) WriteCommand(nodes []*model.PrintNode, at model.Cursor, cmd *model.Command) {
	tokens, fillPlan := w.commandTokens(cmd)
	if len(tokens) == 0 && fillPlan.Mode != regcontroller.FillSpread {
		return
	}
	regcontroller.ApplyFillPlan(nodes, at, tokens, fillPlan)

	//subrow,

}

// commandTokens
func (w T5833CommandWriter) commandTokens(cmd *model.Command) ([]string, regcontroller.FillPlan) {
	if cmd == nil {
		return nil, regcontroller.FillPlan{}
	}

	bg := w.addrInfoToString(cmd.BankGroup)
	ba := w.addrInfoToString(cmd.Bank)
	row := w.addrInfoToString(cmd.Row)
	col := w.addrInfoToString(cmd.Column)
	ma := w.addrInfoToString(cmd.MA)
	op := w.addrInfoToString(cmd.OpCode)
	log.Debugf("cmd %v params: bg=%v, ba=%v, row=%v, col=%v, ma=%v, op=%v on line %v",
		cmd.Type, bg, ba, row, col, ma, op, cmd.LineNum,
	)

	switch cmd.Type {
	case nativeCmd.ACT:
		fillPlan := regcontroller.FillPlan{
			Mode: regcontroller.FillSpread,
			SpreadTokens: []regcontroller.SpreadToken{
				{Value: w.prefixToken("N<", bg), Offset: -3},   // → Cursor-3
				{Value: w.prefixToken("B<", ba), Offset: -2},   // → Cursor-2
				{Value: w.prefixToken("XC<", row), Offset: -1}, // → Cursor-1
			},
		}
		return nil, fillPlan

	case nativeCmd.RDA, nativeCmd.RD:
		fillPlan := regcontroller.FillPlan{
			Mode:      regcontroller.FillRW,
			SelfIndex: 4,
		}

		return w.joinNonEmpty(
			"TP<#FFFF\t\tTP<#0000\t",
			"WRDQLL\t",
			"WRDQLL\t",
			"WRDQLL\t",
			"WRDQHL\t", // cmd本身的锚点在这
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
		), fillPlan

	case nativeCmd.WR, nativeCmd.WRA:
		fillPlan := regcontroller.FillPlan{
			Mode:      regcontroller.FillRW,
			SelfIndex: 5,
		}

		return w.joinNonEmpty(
			"WRDQLL\t",
			"WRDQLL\t",
			"WRDQLL\t",
			"WRDQLL\t",
			"WRDQHL\t", // cmd本身的锚点在这
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
			"WRDQHL\t",
		), fillPlan

	case nativeCmd.MRW:
		fillPlan := regcontroller.FillPlan{
			IncludeSelf:     false,
			Up:              1,
			Down:            0,
			Mode:            regcontroller.FillBatch,
			CrossPNFallback: true,
		}

		w.Mrs.Add(1, 2) //填入MR

		return w.joinNonEmpty(
			w.prefixToken("XT<", ma),
			w.prefixToken("YT<", op),
		), fillPlan
	// case "PRE": ...
	// case "REF": ...
	default:
		log.Errorf("UNKNOWN COMMAND: %v", cmd.Type)
		return nil, regcontroller.FillPlan{}
	}
}

func (w T5833CommandWriter) addrInfoToString(ai *model.AddressInfo) string {
	if ai == nil {
		return ""
	}
	// TODO: 請改AddressInfo
	return fmt.Sprint(ai.Value)
}

func (w T5833CommandWriter) prefixToken(prefix, v string) string {
	if v == "" {
		return ""
	}
	return prefix + v
}

func (w T5833CommandWriter) joinNonEmpty(ss ...string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (w T5833CommandWriter) appendAddr(sr *model.SubRow, tokens []string) {
	if len(tokens) == 0 {
		return
	}
	sr.Addr = append(sr.Addr, tokens...)
}
