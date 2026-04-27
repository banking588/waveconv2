package regcontroller

import (
	"fmt"

	"innotron.com/waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

var nativeCmd = model.DDR5

// CommandWriter：處理command（ACT/PRE/RD/WR...）的 Row/Col/BG/BA所產生的 tokens 要怎麼寫入哪些 subrow
type CommandWriter interface {
	WriteCommand(nodes []*model.PrintNode, at model.Cursor, cmd *model.Command)
}

// DefaultCommandWriter, interface的實作, 只需要实作WriteCommand
// 如果不知道自己怎么新建一个writer, 可以参考这个写法建立,
// - switch cmd.Type 產生 tokens
// - 預設把 tokens 寫回「當前 subrow」
// 可以換成：往上 N 個、往下 N 個、跨 PN 時停止/改寫等
//
//	新建方式:
//	op := regcontroller.NewRegOperator(
//	regcontroller.WithCommandWriter(DefaultCommandWriter{}),
//	)
type DefaultCommandWriter struct{}

func (w DefaultCommandWriter) WriteCommand(nodes []*model.PrintNode, at model.Cursor, cmd *model.Command) {
	tokens, fillPlan := w.commandTokens(cmd)
	if len(tokens) == 0 {
		return
	}

	ApplyFillPlan(nodes, at, tokens, fillPlan)
}

// commandTokens
func (w DefaultCommandWriter) commandTokens(cmd *model.Command) ([]string, FillPlan) {
	if cmd == nil {
		return nil, FillPlan{}
	}

	bg := w.addrInfoToString(cmd.BankGroup)
	ba := w.addrInfoToString(cmd.Bank)
	row := w.addrInfoToString(cmd.Row)
	col := w.addrInfoToString(cmd.Column)
	ma := w.addrInfoToString(cmd.MA)
	op := w.addrInfoToString(cmd.OpCode)

	switch cmd.Type {
	case nativeCmd.ACT:
		return w.joinNonEmpty(
			w.prefixToken("N<", bg),
			w.prefixToken("B<", ba),
			w.prefixToken("XC<", row),
		), FillPlan{}

	case nativeCmd.RD, nativeCmd.RDA:
		return w.joinNonEmpty(
			w.prefixToken("N<", bg),
			w.prefixToken("B<", ba),
			w.prefixToken("YC<", col),
		), FillPlan{}

	case nativeCmd.WR, nativeCmd.WRA:
		return w.joinNonEmpty(
			w.prefixToken("N<", bg),
			w.prefixToken("B<", ba),
			w.prefixToken("YC<", col),
		), FillPlan{}
	case nativeCmd.MRW:
		return w.joinNonEmpty(
			w.prefixToken("XT<", ma),
			w.prefixToken("YT<", op),
		), FillPlan{}
	case nativeCmd.MPC:
		return w.joinNonEmpty(
			w.prefixToken("YT<", op),
		), FillPlan{}
	// case "PRE": ...
	// case "REF": ...
	default:
		return nil, FillPlan{}
	}
}

func (w DefaultCommandWriter) addrInfoToString(ai *model.AddressInfo) string {
	if ai == nil {
		return ""
	}
	// TODO: 請改AddressInfo
	return fmt.Sprint(ai.Value)
}

func (w DefaultCommandWriter) prefixToken(prefix, v string) string {
	if v == "" {
		return ""
	}
	return prefix + v
}

func (w DefaultCommandWriter) joinNonEmpty(ss ...string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
