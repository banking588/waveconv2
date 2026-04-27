package core

import (
	"fmt"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	"waveconv/pkg/util"
)

// ExpandedCommand 展開後的命令
type ExpandedCommand struct {
	LineNum   int
	CmdType   model.CommandType
	BankGroup util.HexValue
	Bank      util.HexValue
	Row       util.HexValue
	Column    util.HexValue
	Reg       util.HexValue
	OpCode    util.HexValue
	CW        util.HexValue
	DQData    []string
	HasDQ     bool
	RepeatCnt int // 命令佔幾拍
	Raw       string
}

// intToHex 將 int 轉換為 AddressInfo
func intToHexWithName(name string, v int) *model.AddressInfo {
	addr := &model.AddressInfo{
		Name:  name,
		Value: util.HexValue(fmt.Sprintf("0x%X", v)),
	}
	return addr
}

// intToHex 將 int 轉換為 util.HexValue
func intToHex(v int) util.HexValue {
	return util.HexValue(fmt.Sprintf("0x%X", v))

}

// ParseSeqNodes 遞迴遍歷所有 SeqNode，對 NodeCommand 類型填入 Cmd，
// 對 NodeSpecialTag 類型則套用變數運算
// 回傳過濾後的 nodes（移除不需要的節點）
var penddingDqSetting []string //dqsetting from previous node

// ParseSeqNodes 加上 proto 參數，往下傳
func ParseSeqNodes(nodes []*SeqNode, vars *VariableStore, mrs mrstatusreg.MRS, proto model.ProtocolCommands) ([]*SeqNode, error) {

	result := make([]*SeqNode, 0, len(nodes))

	for _, node := range nodes {
		switch node.Type {
		case NodeSpecialTag:
			vars.Apply(node)
			if node.SkipThisSpecialTagNode {
				continue
			}

			if penddingDqSetting != nil && len(penddingDqSetting) > 0 {
				log.Errorf("there is legacy data setting %v on line %v", penddingDqSetting, node.LineNum)
			}

			if node.SpecialTagDQSetting != nil && len(node.SpecialTagDQSetting) > 0 {
				penddingDqSetting = node.SpecialTagDQSetting
				continue
			}

		case NodeCommand:
			if err := ParseNodeCommand(node, vars, &penddingDqSetting, mrs, proto); err != nil {
				return nil, err
			}

		case NodeLoop:
			filtered, err := ParseSeqNodes(node.Children, vars, mrs, proto)
			if err != nil {
				return nil, err
			}
			node.Children = filtered
		}

		result = append(result, node)
	}

	return result, nil
}

// ParseNodeCommand 用 Role 路由 + Parser 解析，產品無關
func ParseNodeCommand(node *SeqNode, vars *VariableStore, dqsetting *[]string, mrs mrstatusreg.MRS, proto model.ProtocolCommands) error {
	fields := node.CmdFields
	if len(fields) == 0 {
		return fmt.Errorf("line %d: empty command", node.LineNum)
	}

	cmdType := model.CommandType(fields[0])
	cmd := &model.Command{
		LineNum:   node.LineNum,
		LineCount: 1,
		Type:      cmdType,
		RepeatCnt: 1,
		Raw:       node.Raw,
	}

	// 步驟 1：查 Role
	role, known := proto.RoleOf(cmdType)
	if !known {
		log.Errorf("UNKNOWN COMMAND INPUT FROM SEQ: %v on line %v", cmdType, node.LineNum)
		node.Cmd = cmd
		return nil
	}

	// 步驟 2：拿 Parser
	parser := proto.Parser()
	var err error

	// 步驟 3：用 Role 路由到對應的 Parser 方法
	switch role {
	case model.RoleACT, model.RoleACTX:
		err = parser.ParseACT(fields, cmd, vars)

	case model.RoleACT1, model.RoleACT2:
		err = parser.ParseACT(fields, cmd, vars)

	case model.RoleRD, model.RoleRDA, model.RoleRDX, model.RoleRDAX:
		err = parser.ParseRD(fields, cmd, vars, dqsetting)

	case model.RoleRDC:
		err = parser.ParseRD(fields, cmd, vars, dqsetting)

	case model.RoleWR, model.RoleWRA:
		err = parser.ParseWR(fields, cmd, vars, dqsetting)

	case model.RoleWRP, model.RoleWRPX, model.RoleWRPA, model.RoleWRPAX:
		err = parser.ParseWRP(fields, cmd, vars, dqsetting)

	case model.RoleMRW1, model.RoleMRW2:
		err = parser.ParseMRW(fields, cmd, vars)
		// MRS
		if err == nil && mrs != nil && len(fields) >= 3 {
			maVal := uint32(vars.Resolve(fields[1]))
			opVal := uint8(vars.Resolve(fields[2]))
			if !mrs.Add(maVal, opVal) {
				log.Warnf("line %d: MRW MA=%d not a known MR register, skipped", node.LineNum, maVal)
			}

		}

	case model.RoleMRW, model.RoleMRWX:
		err = parser.ParseMRW(fields, cmd, vars)

		// MRS
		if err == nil && mrs != nil && len(fields) >= 3 {
			maVal := uint32(vars.Resolve(fields[1]))
			opVal := uint8(vars.Resolve(fields[2]))
			if !mrs.Add(maVal, opVal) {
				log.Warnf("line %d: MRW MA=%d not a known MR register, skipped", node.LineNum, maVal)
			}
		}

	case model.RoleMRR, model.RoleMRRX:
		err = parser.ParseMRR(fields, cmd, vars, dqsetting)

	case model.RoleMPC:
		err = parser.ParseMPC(fields, cmd, vars)

	case model.RoleDES, model.RoleNOP:
		err = parser.ParseDES(fields, cmd)

	case model.RolePREsb, model.RolePREpb:
		err = parser.ParsePRE(fields, cmd, vars)

	case model.RolePREab, model.RolePRE:
		// no param

	case model.RoleREFsb:
		err = parser.ParseREFsb(fields, cmd, vars)

	case model.RoleREF, model.RoleREFab:
		// no param

	case model.RoleVrefCA, model.RoleVrefCS:
		err = parser.ParseVref(fields, cmd, vars)

	case model.RoleRESET,
		model.RoleRFMab, model.RoleRFMsb,
		model.RoleCKE_H, model.RoleCKE_L,
		model.RoleSRE, model.RoleSRX, model.RoleSREF,
		model.RolePDE, model.RolePDX, model.RoleAPDX,
		model.RoleTestModeEntry, model.RoleTestModeExit:
		// no params
	}

	if err != nil {
		return err
	}

	node.Cmd = cmd
	return nil
}
