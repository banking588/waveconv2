package ddr5

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	ddr5mr "waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg/ddr5"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller"
	"waveconv/pkg/util"
	"waveconv/pkg/util/atelib"
	"waveconv/pkg/util/queue"
)

// 5503HS CommandWriter
// 不用的 cmd.Type 產生 各自的tokens
// 可以換成：往上 N 個、往下 N 個、跨 PN 時停止/改寫等
// 把Commnader拆出来是因为, 可能同一种机台会有多种不同的macrodef写法
// 因应这种情况, 所以把它从printer中独立出来, 方便各个产品用不同的maco实作, 互相解耦
type HS5503CommandWriter struct {
	Mrs           mrstatusreg.MRS
	XAddressPageQ *queue.FixedCycleQueue[atelib.PageRegisterName]
	YAddressPageQ *queue.FixedCycleQueue[atelib.PageRegisterName]
	TPRegisterQ   *queue.FixedCycleQueue[[]string]
}

func NewHS5503CommandWriter(mrs mrstatusreg.MRS) *HS5503CommandWriter {
	// X address page register 初始化
	xaddressCycler := queue.NewPageRegisterNameCycler(atelib.GetPageRegisterNames("x"), 0)
	XAddressPageQ := queue.NewFixedCycleQueue(xaddressCycler.Len(), xaddressCycler)

	// Y address page register 初始化
	yaddressCycler := queue.NewPageRegisterNameCycler(atelib.GetPageRegisterNames("y"), 0)
	YAddressPageQ := queue.NewFixedCycleQueue(yaddressCycler.Len(), yaddressCycler)

	// Data TP 初始化
	tpRegCycler := queue.NewStringSliceCycler(atelib.GetDataTopoRegisterNames(), 0)
	TPRegisterQ := queue.NewFixedCycleQueue(tpRegCycler.Len(), tpRegCycler)

	return &HS5503CommandWriter{
		Mrs:           mrs,
		XAddressPageQ: XAddressPageQ,
		YAddressPageQ: YAddressPageQ,
		TPRegisterQ:   TPRegisterQ,
	}
}

type FillInfo struct {
	Tokens   []string
	FillPlan regcontroller.FillPlan
}

type DualFillInfo struct {
	Addr FillInfo
	Data FillInfo
}

func (w HS5503CommandWriter) WriteCommand(nodes []*model.PrintNode, at model.Cursor, cmd *model.Command) {
	tokens, fillPlan, dualInfo := w.commandTokens(cmd, at, nodes)

	// 针对需要返回datatoken和addrtoken字段
	if dualInfo != nil {
		if len(dualInfo.Addr.Tokens) > 0 || dualInfo.Addr.FillPlan.Mode == regcontroller.FillSpread {
			regcontroller.ApplyFillPlan(nodes, at, dualInfo.Addr.Tokens, dualInfo.Addr.FillPlan)
		}

		if len(dualInfo.Data.Tokens) > 0 || dualInfo.Data.FillPlan.Mode == regcontroller.FillSpread {
			regcontroller.ApplyFillPlan(nodes, at, dualInfo.Data.Tokens, dualInfo.Data.FillPlan)
		}
		return
	}

	// 只需要返回单一token
	if len(tokens) == 0 && fillPlan.Mode != regcontroller.FillSpread {
		return
	}
	regcontroller.ApplyFillPlan(nodes, at, tokens, fillPlan)
}

// commandTokens
func (w HS5503CommandWriter) commandTokens(cmd *model.Command, cursor model.Cursor, nodes []*model.PrintNode) ([]string, regcontroller.FillPlan, *DualFillInfo) {
	if cmd == nil {
		return nil, regcontroller.FillPlan{}, nil
	}

	bg := w.addrInfoToString(cmd.BankGroup)
	ba := w.addrInfoToString(cmd.Bank)
	row := w.addrInfoToString(cmd.Row)
	col := w.addrInfoToString(cmd.Column)
	ma := w.addrInfoToString(cmd.MA)
	op := w.addrInfoToString(cmd.OpCode)
	// log.Debugf("cmd %v params: bg=%v, ba=%v, row=%v, col=%v, ma=%v, op=%v on line %v",
	// 	cmd.Type, bg, ba, row, col, ma, op, cmd.LineNum,
	// )

	switch cmd.Type {
	case nativeCmd.DES:
		if cursor.PN >= 0 && cursor.PN < len(nodes) &&
			cursor.SR >= 0 && cursor.SR < len(nodes[cursor.PN].SubRows) {

			subrow := &nodes[cursor.PN].SubRows[cursor.SR]

			// 判断是否为保留格式的函数
			isReservedPattern := func(token string) bool {
				return token == "Z<Z+1" ||
					token == "B<B+1" ||
					token == "N<N+1" ||
					token == "XC<XC+1" ||
					token == "YC<YC+1" ||
					token == "XC<#0" ||
					token == "YC<#0"
			}

			// 如果 Addr 为空，使用原来的 FillBatch 模式
			if len(subrow.Addr) == 0 {
				tokens := w.joinNonEmpty()
				fillPlan := regcontroller.FillPlan{
					IncludeSelf:     true,
					Up:              0,
					Down:            0,
					Mode:            regcontroller.FillBatch,
					CrossPNFallback: false,
				}
				return tokens, fillPlan, nil
			}

			// 检查是否所有token都是保留格式
			allReserved := true
			for _, addrToken := range subrow.Addr {
				if !isReservedPattern(addrToken) {
					allReserved = false
					break
				}
			}

			// 如果所有token都是保留格式，使用 FillBatch 模式
			if allReserved {
				fillPlan := regcontroller.FillPlan{
					IncludeSelf:     true,
					Up:              0,
					Down:            0,
					Mode:            regcontroller.FillBatch,
					CrossPNFallback: false,
				}
				return nil, fillPlan, nil
			}

			// 否则使用 FillSpread 模式处理
			var spreadTokens []regcontroller.SpreadToken

			for _, addrToken := range subrow.Addr {
				originalVal := w.extractOriginalValueWithHash(addrToken)

				if originalVal != "" {
					// 创建 DES token 生成器
					generator := NewDesTokenGenerator(originalVal)

					// 判断是 XC 还是 YC
					if strings.HasPrefix(addrToken, "XC<") {
						xcTokens := generator.GenerateXcTokens()
						spreadTokens = append(spreadTokens, xcTokens...)
					} else if strings.HasPrefix(addrToken, "YC<") {
						ycTokens := generator.GenerateYcTokens()
						spreadTokens = append(spreadTokens, ycTokens...)
					}
				}
			}

			subrow.Addr = nil // 只有在使用FillSpread时才清空
			fillPlan := regcontroller.FillPlan{
				Mode:         regcontroller.FillSpread,
				SpreadTokens: spreadTokens,
			}
			return nil, fillPlan, nil
		}

		// 默认情况使用 FillBatch 模式
		tokens := w.joinNonEmpty()

		fillPlan := regcontroller.FillPlan{
			IncludeSelf:     true,
			Up:              0,
			Down:            0,
			Mode:            regcontroller.FillBatch,
			CrossPNFallback: false,
		}

		return tokens, fillPlan, nil

	case nativeCmd.ACT:
		log.Debugf("cmd %v params: bg=%v, ba=%v, row=%v, col=%v, ma=%v, op=%v on line %v",
			cmd.Type, bg, ba, row, col, ma, op, cmd.LineNum,
		)
		var spreadTokens []regcontroller.SpreadToken

		// 处理 row 逻辑
		shouldAddRowTokens := true
		if cmd.Row != nil && strings.HasPrefix(cmd.Row.Name, "ROW") {
			if w.isZeroValue(cmd.Row.Value) {
				shouldAddRowTokens = false
			}
		}

		// 处理 bg 逻辑
		shouldAddBGToken := true
		if cmd.BankGroup != nil && strings.HasPrefix(cmd.BankGroup.Name, "BG") {
			if w.isZeroValue(cmd.BankGroup.Value) {
				shouldAddBGToken = false
			}
		}

		// 处理 ba 逻辑
		shouldAddBAToken := true
		if cmd.Bank != nil && strings.HasPrefix(cmd.Bank.Name, "BA") {
			if w.isZeroValue(cmd.Bank.Value) {
				shouldAddBAToken = false
			}
		}

		xc, _ := w.XAddressPageQ.Pop(cursor)
		// offset -2: D1<row
		if shouldAddRowTokens && row != "" {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  xc.GetDName(row),
				Offset: -2, // 往上 2 行
			})
		}

		// offset -1: XC<D1
		if shouldAddRowTokens && row != "" {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  xc.GetCName(),
				Offset: -1, // 往上 1 行
			})
		}

		// offset 0: NSEL<ND(bg) BSEL<BD(ba)
		var offsetMinus0Tokens []string

		if shouldAddBGToken && bg != "" {
			cleanBG := strings.TrimPrefix(bg, "#")
			if cleanBG == "0" || cleanBG == "00" {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<NH") // BG为0时，对应NH
			} else if len(cleanBG) > 0 {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<ND"+cleanBG) // BG为1~7，对应ND1~ND7
			}
		}

		if shouldAddBAToken && ba != "" {
			cleanBA := strings.TrimPrefix(ba, "#")
			if len(cleanBA) > 0 {
				// BA为0~3，对应BD1~BD4
				if baValue, err := strconv.Atoi(cleanBA); err == nil && baValue >= 0 && baValue <= 3 {
					offsetMinus0Tokens = append(offsetMinus0Tokens, fmt.Sprintf("BSEL<BD%d", baValue+1))
				} else {
					log.Errorf("BA value out of range (0-3) or invalid: %s", cleanBA)
				}
			}
		}

		if len(offsetMinus0Tokens) > 0 {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  strings.Join(offsetMinus0Tokens, " "),
				Offset: 0,
			})
		}

		fillPlan := regcontroller.FillPlan{
			Mode:         regcontroller.FillSpread,
			SpreadTokens: spreadTokens,
		}
		return nil, fillPlan, nil

	case nativeCmd.RDA, nativeCmd.RD:
		var selfmacroNames []string

		// mr0用于确定BL
		mr0, err := mrstatusreg.GetTyped[*ddr5mr.MR0](w.Mrs, ddr5mr.Name_MR_0)
		if err != nil {
			log.Error(err)
		}

		bl := mr0.BurstLength()

		// self部分根据BL来决定RDQSHL R 有多少个
		numSelfMacros := int(bl / 2)
		for i := 0; i < numSelfMacros; i++ {
			dqs := atelib.NewDQS(atelib.Read, atelib.Strobe_data)
			macroName, _ := dqs.MacroName()
			selfmacroNames = append(selfmacroNames, macroName)
		}

		// 处理条件逻辑
		shouldAddColTokens := true
		if cmd.Column != nil && strings.HasPrefix(cmd.Column.Name, "COL") {
			if w.isZeroValue(cmd.Column.Value) {
				shouldAddColTokens = false
			}
		}

		shouldAddBGToken := true
		if cmd.BankGroup != nil && strings.HasPrefix(cmd.BankGroup.Name, "BG") {
			if w.isZeroValue(cmd.BankGroup.Value) {
				shouldAddBGToken = false
			}
		}

		shouldAddBAToken := true
		if cmd.Bank != nil && strings.HasPrefix(cmd.Bank.Name, "BA") {
			if w.isZeroValue(cmd.Bank.Value) {
				shouldAddBAToken = false
			}
		}

		var spreadTokens []regcontroller.SpreadToken

		yc, _ := w.YAddressPageQ.Pop(cursor)
		// offset -2: D2<col
		if shouldAddColTokens && col != "" {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  yc.GetDName(col),
				Offset: -2, // 往上 2 行
			})
		}

		// offset -1: YC<D2
		if shouldAddColTokens && row != "" {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  yc.GetCName(),
				Offset: -1, // 往上 1 行
			})
		}

		// offset 0: NSEL<ND(bg) BSEL<BD(ba)
		var offsetMinus0Tokens []string

		if shouldAddBGToken && bg != "" {
			cleanBG := strings.TrimPrefix(bg, "#")
			if cleanBG == "0" || cleanBG == "00" {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<NH") // BG为0时，对应NH
			} else if len(cleanBG) > 0 {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<ND"+cleanBG) // BG为1~7，对应ND1~ND7
			}
		}

		if shouldAddBAToken && ba != "" {
			cleanBA := strings.TrimPrefix(ba, "#")
			if len(cleanBA) > 0 {
				// BA为0~3，对应BD1~BD4
				if baValue, err := strconv.Atoi(cleanBA); err == nil && baValue >= 0 && baValue <= 3 {
					offsetMinus0Tokens = append(offsetMinus0Tokens, fmt.Sprintf("BSEL<BD%d", baValue+1))
				} else {
					log.Errorf("BA value out of range (0-3) or invalid: %s", cleanBA)
				}
			}
		}

		if len(offsetMinus0Tokens) > 0 {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  strings.Join(offsetMinus0Tokens, " "),
				Offset: 0,
			})
		}

		// 构造 DualFillInfo - 分离 addr 和 data
		addrFillPlan := regcontroller.FillPlan{
			Mode:         regcontroller.FillSpread,
			SpreadTokens: spreadTokens,
			WriteToData:  false,
		}

		dataFillPlan := regcontroller.FillPlan{
			Mode:        regcontroller.FillRW,
			SelfIndex:   0,
			IncludeSelf: true,
			WriteToData: true,
		}

		dualInfo := &DualFillInfo{
			Addr: FillInfo{
				FillPlan: addrFillPlan,
			},
			Data: FillInfo{
				Tokens:   selfmacroNames,
				FillPlan: dataFillPlan,
			},
		}

		return nil, regcontroller.FillPlan{}, dualInfo

	case nativeCmd.WR, nativeCmd.WRA:

		var premacroNames []string
		var pstmacroNames []string
		var selfmacroNames []string

		// === 1. 生成 preamble/toggle/postamble ===
		mr0, err := mrstatusreg.GetTyped[*ddr5mr.MR0](w.Mrs, ddr5mr.Name_MR_0)
		if err != nil {
			log.Error(err)
		}
		bl := mr0.BurstLength()

		numSelfMacros := int(bl / 2)
		for i := 0; i < numSelfMacros; i++ {
			dqs := atelib.NewDQS(atelib.Write, atelib.Strobe_data)
			macroName, _ := dqs.MacroName()
			selfmacroNames = append(selfmacroNames, macroName)
		}

		mr8, err := mrstatusreg.GetTyped[*ddr5mr.MR8](w.Mrs, ddr5mr.Name_MR_8)
		if err != nil {
			log.Error(err)
		}

		preamble := mr8.WritePreambleSetting()
		for _, pre := range preamble {
			switch pre {
			case "00":
				dqs := atelib.NewDQS(atelib.Write, atelib.Preamble)
				premacroName, _ := dqs.MacroName()
				premacroNames = append(premacroNames, premacroName)
			case "10":
				dqs := atelib.NewDQS(atelib.Write, atelib.Strobe)
				macroName, _ := dqs.MacroName()
				premacroNames = append(premacroNames, macroName)
			}
		}

		postamble := mr8.WritePostambleSetting()
		for _, pst := range postamble {
			switch pst {
			case "00":
				dqs := atelib.NewDQS(atelib.Write, atelib.Postamble)
				pstmacroName, _ := dqs.MacroName()
				pstmacroNames = append(pstmacroNames, pstmacroName)
			}
		}

		// === 2. 地址部分逻辑 ===

		shouldAddColTokens := true
		if cmd.Column != nil && strings.HasPrefix(cmd.Column.Name, "COL") {
			if w.isZeroValue(cmd.Column.Value) {
				shouldAddColTokens = false
			}
		}

		shouldAddBGToken := true
		if cmd.BankGroup != nil && strings.HasPrefix(cmd.BankGroup.Name, "BG") {
			if w.isZeroValue(cmd.BankGroup.Value) {
				shouldAddBGToken = false
			}
		}

		shouldAddBAToken := true
		if cmd.Bank != nil && strings.HasPrefix(cmd.Bank.Name, "BA") {
			if w.isZeroValue(cmd.Bank.Value) {
				shouldAddBAToken = false
			}
		}

		var spreadTokens []regcontroller.SpreadToken

		yc, _ := w.YAddressPageQ.Pop(cursor)
		// offset -2: D2<col
		if shouldAddColTokens && col != "" {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  yc.GetDName(col),
				Offset: -2,
			})
		}

		// offset -1: YC<YC+D2
		if shouldAddColTokens && row != "" {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  yc.GetCName(),
				Offset: -1,
			})
		}

		// offset 0: NSEL<ND(bg) BSEL<BD(ba)
		var offsetMinus0Tokens []string

		if shouldAddBGToken && bg != "" {
			cleanBG := strings.TrimPrefix(bg, "#")
			if cleanBG == "0" || cleanBG == "00" {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<NH")
			} else if len(cleanBG) > 0 {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<ND"+cleanBG)
			}
		}

		if shouldAddBAToken && ba != "" {
			cleanBA := strings.TrimPrefix(ba, "#")
			if len(cleanBA) > 0 {
				if baValue, err := strconv.Atoi(cleanBA); err == nil && baValue >= 0 && baValue <= 3 {
					offsetMinus0Tokens = append(offsetMinus0Tokens, fmt.Sprintf("BSEL<BD%d", baValue+1))
				} else {
					log.Errorf("BA value out of range (0-3) or invalid: %s", cleanBA)
				}
			}
		}

		if len(offsetMinus0Tokens) > 0 {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  strings.Join(offsetMinus0Tokens, " "),
				Offset: 0,
			})
		}

		// === 3. 找到所有空位置 ===
		preambleLen := len(premacroNames)
		type Position struct {
			PN     int
			SR     int
			Offset int // 相对于当前位置的偏移量
		}
		var emptyPositions []Position

		for i := 1; i <= preambleLen; i++ {
			checkSR := cursor.SR - i
			checkPN := cursor.PN
			offset := -i

			// 处理跨 PN 的情况
			for checkSR < 0 {
				checkPN--
				if checkPN < 0 {
					break
				}
				if checkPN < len(nodes) {
					checkSR += len(nodes[checkPN].SubRows)
				} else {
					break
				}
			}

			if checkPN >= 0 && checkPN < len(nodes) &&
				checkSR >= 0 && checkSR < len(nodes[checkPN].SubRows) {
				subrow := &nodes[checkPN].SubRows[checkSR]
				if len(subrow.Data) == 0 {
					emptyPositions = append(emptyPositions, Position{
						PN:     checkPN,
						SR:     checkSR,
						Offset: offset,
					})
				}
			}
		}

		sort.Slice(emptyPositions, func(i, j int) bool {
			return emptyPositions[i].Offset > emptyPositions[j].Offset
		})

		// === 4. 将 preamble 分配给前面的空位置 ===
		if len(emptyPositions) > 0 && preambleLen > 0 {
			for i, pos := range emptyPositions {
				if i < len(premacroNames) {
					// cursor.SR-1 的位置放最后一个 preamble
					preambleIndex := len(premacroNames) - 1 - i
					if preambleIndex >= 0 && preambleIndex < len(premacroNames) {
						preambleToken := premacroNames[preambleIndex]
						nodes[pos.PN].SubRows[pos.SR].Data = []string{preambleToken}
					}
				}
			}
		}
		// === 5. 当前位置永远只放 data 和 postamble ===
		finalDataTokens := append(selfmacroNames, pstmacroNames...)

		// === 6. 构造 FillPlan 和 DualFillInfo ===
		dataFillPlan := regcontroller.FillPlan{
			Mode:        regcontroller.FillRW,
			SelfIndex:   0,
			IncludeSelf: true,
			WriteToData: true,
		}

		addrFillPlan := regcontroller.FillPlan{
			Mode:         regcontroller.FillSpread,
			SpreadTokens: spreadTokens,
			WriteToData:  false,
		}

		dualInfo := &DualFillInfo{
			Addr: FillInfo{FillPlan: addrFillPlan},
			Data: FillInfo{
				Tokens:   finalDataTokens,
				FillPlan: dataFillPlan,
			},
		}

		return nil, regcontroller.FillPlan{}, dualInfo

	case nativeCmd.MRW:
		fillPlan := regcontroller.FillPlan{
			IncludeSelf:             false,
			Up:                      1,
			Down:                    0,
			Mode:                    regcontroller.FillBatch,
			CrossPNFallback:         true,
			OverrideCrossPNFallback: true,
		}

		return w.joinNonEmpty(
			w.prefixToken("XT<", ma),
			w.prefixToken("YT<", op),
		), fillPlan, nil

	case nativeCmd.MPC:
		fillPlan := regcontroller.FillPlan{
			IncludeSelf:             false,
			Up:                      1,
			Down:                    0,
			Mode:                    regcontroller.FillBatch,
			CrossPNFallback:         true,
			OverrideCrossPNFallback: true,
		}

		return w.joinNonEmpty(
			w.prefixToken("YT<", op),
		), fillPlan, nil

	case nativeCmd.VrefCA:
		fillPlan := regcontroller.FillPlan{
			IncludeSelf:             false,
			Up:                      1,
			Down:                    0,
			Mode:                    regcontroller.FillBatch,
			CrossPNFallback:         true,
			OverrideCrossPNFallback: true,
		}

		return w.joinNonEmpty(
			w.prefixToken("YT<", op),
		), fillPlan, nil

	case nativeCmd.VrefCS:
		fillPlan := regcontroller.FillPlan{
			IncludeSelf:             false,
			Up:                      1,
			Down:                    0,
			Mode:                    regcontroller.FillBatch,
			CrossPNFallback:         true,
			OverrideCrossPNFallback: true,
		}

		return w.joinNonEmpty(
			w.prefixToken("YT<", op),
		), fillPlan, nil

	case nativeCmd.MRR:
		fillPlan := regcontroller.FillPlan{
			IncludeSelf:             false,
			Up:                      1,
			Down:                    0,
			Mode:                    regcontroller.FillBatch,
			CrossPNFallback:         true,
			OverrideCrossPNFallback: true,
		}

		return w.joinNonEmpty(
			w.prefixToken("XT<", ma),
		), fillPlan, nil

	case nativeCmd.PREpb:

		var spreadTokens []regcontroller.SpreadToken

		// 处理 bg 逻辑
		shouldAddBGToken := true
		if cmd.BankGroup != nil && strings.HasPrefix(cmd.BankGroup.Name, "BG") {
			if w.isZeroValue(cmd.BankGroup.Value) {
				shouldAddBGToken = false
			}
		}

		// 处理 ba 逻辑
		shouldAddBAToken := true
		if cmd.Bank != nil && strings.HasPrefix(cmd.Bank.Name, "BA") {
			if w.isZeroValue(cmd.Bank.Value) {
				shouldAddBAToken = false
			}
		}

		// offset 0: NSEL<ND(bg) BSEL<BD(ba)
		var offsetMinus0Tokens []string

		if shouldAddBGToken && bg != "" {
			cleanBG := strings.TrimPrefix(bg, "#")
			if cleanBG == "0" || cleanBG == "00" {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<NH") // BG为0时，对应NH
			} else if len(cleanBG) > 0 {
				offsetMinus0Tokens = append(offsetMinus0Tokens, "NSEL<ND"+cleanBG) // BG为1~7，对应ND1~ND7
			}
		}

		if shouldAddBAToken && ba != "" {
			cleanBA := strings.TrimPrefix(ba, "#")
			if len(cleanBA) > 0 {
				// BA为0~3，对应BD1~BD4
				if baValue, err := strconv.Atoi(cleanBA); err == nil && baValue >= 0 && baValue <= 3 {
					offsetMinus0Tokens = append(offsetMinus0Tokens, fmt.Sprintf("BSEL<BD%d", baValue+1))
				} else {
					log.Errorf("BA value out of range (0-3) or invalid: %s", cleanBA)
				}
			}
		}

		if len(offsetMinus0Tokens) > 0 {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  strings.Join(offsetMinus0Tokens, " "),
				Offset: 0,
			})
		}

		fillPlan := regcontroller.FillPlan{
			Mode:         regcontroller.FillSpread,
			SpreadTokens: spreadTokens,
		}
		return nil, fillPlan, nil

	case nativeCmd.PREsb:

		var spreadTokens []regcontroller.SpreadToken

		// 处理 ba 逻辑
		shouldAddBAToken := true
		if cmd.Bank != nil && strings.HasPrefix(cmd.Bank.Name, "BA") {
			if w.isZeroValue(cmd.Bank.Value) {
				shouldAddBAToken = false
			}
		}

		// offset 0:  BSEL<BD(ba)
		var offsetMinus0Tokens []string

		if shouldAddBAToken && ba != "" {
			cleanBA := strings.TrimPrefix(ba, "#")
			if len(cleanBA) > 0 {
				// BA为0~3，对应BD1~BD4
				if baValue, err := strconv.Atoi(cleanBA); err == nil && baValue >= 0 && baValue <= 3 {
					offsetMinus0Tokens = append(offsetMinus0Tokens, fmt.Sprintf("BSEL<BD%d", baValue+1))
				} else {
					log.Errorf("BA value out of range (0-3) or invalid: %s", cleanBA)
				}
			}
		}

		if len(offsetMinus0Tokens) > 0 {
			spreadTokens = append(spreadTokens, regcontroller.SpreadToken{
				Value:  strings.Join(offsetMinus0Tokens, " "),
				Offset: 0,
			})
		}

		fillPlan := regcontroller.FillPlan{
			Mode:         regcontroller.FillSpread,
			SpreadTokens: spreadTokens,
		}
		return nil, fillPlan, nil

	// case "REF": ...
	default:
		log.Errorf("UNKNOWN COMMAND: %v", cmd.Type)
		return nil, regcontroller.FillPlan{}, nil
	}
}

func (w HS5503CommandWriter) addrInfoToString(ai *model.AddressInfo) string {
	if ai == nil {
		return ""
	}
	// TODO: 請改AddressInfo
	return fmt.Sprint(ai.Value)
}

func (w HS5503CommandWriter) prefixToken(prefix, v string) string {
	if v == "" {
		return ""
	}
	return prefix + v
}

func (w HS5503CommandWriter) joinNonEmpty(ss ...string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (w HS5503CommandWriter) isZeroValue(value util.HexValue) bool {
	if value.String() == "" {
		return true
	}
	valueStr := value.String()
	return valueStr == "#0" || valueStr == "0" || valueStr == "0x0" || valueStr == "0x00"
}

// 判断上一个WR是否填充了Data（dqs）
func IsDataFilled(nodes []*model.PrintNode, at model.Cursor) bool {
	if at.PN < 0 || at.PN >= len(nodes) {
		return false
	}
	node := nodes[at.PN]

	if at.SR < 0 || at.SR >= len(node.SubRows) {
		return false
	}

	subrow := &node.SubRows[at.SR]
	return len(subrow.Data) > 0
}

// For special_tag，提取值信息，转换成寄存器操作的值格式
func (w HS5503CommandWriter) extractOriginalValueWithHash(addrToken string) string {
	if !strings.Contains(addrToken, "<") {
		return ""
	}

	parts := strings.Split(addrToken, "<")
	if len(parts) != 2 {
		return ""
	}

	rightSide := parts[1]

	// 分别查找 '+' 或 '-' 的位置
	var operatorIndex int
	var operator rune

	for i, char := range rightSide {
		if char == '+' || char == '-' {
			operatorIndex = i
			operator = char
			break
		}
	}
	if operator == 0 {
		return ""
	}
	valStr := rightSide[operatorIndex+1:]

	valStr = strings.TrimSpace(valStr)

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return ""
	}
	decValue := util.DecValue(val)
	return decValue.String()
}

// 生成最终的 DES token（完整的格式）
func (w HS5503CommandWriter) generateFinalDesToken(addrToken string) string {
	if !strings.Contains(addrToken, "<") {
		return addrToken
	}

	parts := strings.Split(addrToken, "<")
	if len(parts) != 2 {
		return addrToken
	}

	variable := parts[0]  // 例如: "XC" 或 "YC"
	rightSide := parts[1] // 例如: "#1A" 或 "YC+#1A" 或 "YC-#1A"

	// 情况1: 简单赋值 "XC<#1A" -> "XC<D4"
	if !strings.Contains(rightSide, "+") && !strings.Contains(rightSide, "-") {
		return fmt.Sprintf("%s<%s+%s", variable, variable, "D4")
	}

	// 情况2: += 操作 "YC<YC+#1A" -> "YC<YC+D4"
	if strings.Contains(rightSide, "+") {
		plusParts := strings.SplitN(rightSide, "+", 2)
		if len(plusParts) == 2 {
			leftVar := plusParts[0]
			return fmt.Sprintf("%s<%s+%s", variable, leftVar, "D4")
		}
	}

	// 情况3: -= 操作 "YC<YC-#1A" -> "YC<YC-D4"
	if strings.Contains(rightSide, "-") {
		minusParts := strings.SplitN(rightSide, "-", 2)
		if len(minusParts) == 2 {
			leftVar := minusParts[0]
			return fmt.Sprintf("%s<%s-%s", variable, leftVar, "D4")
		}
	}

	return addrToken
}

// DesTokenGenerator DES token 生成器
type DesTokenGenerator struct {
	// row自增信息
	XcTargetReg string // D3<D3B
	XcBasetReg  string // D3B, 直接赋值

	// col自增信息
	YcTagetReg string // D4<D4B
	YcBasetReg string // D4B, 直接赋值

	// 通用配置
	Value string // 原始值，如 "#10"
}

// NewDesTokenGenerator 创建新的 DES token 生成器
func NewDesTokenGenerator(value string) *DesTokenGenerator {
	return &DesTokenGenerator{
		XcTargetReg: "D3",
		XcBasetReg:  "D3B",
		YcTagetReg:  "D4",
		YcBasetReg:  "D4B",
		Value:       value,
	}
}

// GenerateXcTokens 生成 XC 相关的 tokens
func (g *DesTokenGenerator) GenerateXcTokens() []regcontroller.SpreadToken {
	var tokens []regcontroller.SpreadToken

	// offset -2: D3B<#10
	tokens = append(tokens, regcontroller.SpreadToken{
		Value:  fmt.Sprintf("%s<%s", g.XcBasetReg, g.Value),
		Offset: -2,
	})

	// offset -1: D3<D4B
	tokens = append(tokens, regcontroller.SpreadToken{
		Value:  fmt.Sprintf("%s<%s", g.XcTargetReg, g.XcBasetReg),
		Offset: -1,
	})

	// offset 0: XC<XC+D3
	tokens = append(tokens, regcontroller.SpreadToken{
		Value:  fmt.Sprintf("XC<%s", g.XcTargetReg),
		Offset: 0,
	})

	return tokens
}

// GenerateYcTokens 生成 YC 相关的 tokens
func (g *DesTokenGenerator) GenerateYcTokens() []regcontroller.SpreadToken {
	var tokens []regcontroller.SpreadToken

	// offset -2: D4B<#10
	tokens = append(tokens, regcontroller.SpreadToken{
		Value:  fmt.Sprintf("%s<%s", g.YcBasetReg, g.Value),
		Offset: -2,
	})

	// offset -1: D4<D4B
	tokens = append(tokens, regcontroller.SpreadToken{
		Value:  fmt.Sprintf("%s<%s", g.YcTagetReg, g.YcBasetReg),
		Offset: -1,
	})

	// offset 0: YC<YC+D4
	tokens = append(tokens, regcontroller.SpreadToken{
		Value:  fmt.Sprintf("YC<%s", g.YcTagetReg),
		Offset: 0,
	})

	return tokens
}
