package model

import (
	"fmt"
	"strconv"
	"strings"

	"waveconv/pkg/util"
)

// VarResolver 讓 parser 不依賴 core.VariableStore
type VarResolver interface {
	Resolve(name string) int
}

// CommandParser 每個協議實作此介面
type CommandParser interface {
	ParseACT(fields []string, cmd *Command, vars VarResolver) error
	ParseRD(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error
	ParseWR(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error
	ParseWRP(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error
	ParseMRW(fields []string, cmd *Command, vars VarResolver) error
	ParseMRR(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error
	ParseMPC(fields []string, cmd *Command, vars VarResolver) error
	ParseDES(fields []string, cmd *Command) error
	ParsePRE(fields []string, cmd *Command, vars VarResolver) error
	ParseREFsb(fields []string, cmd *Command, vars VarResolver) error
	ParseVref(fields []string, cmd *Command, vars VarResolver) error
}

func toAddr(name string, v int) *AddressInfo {
	return &AddressInfo{
		Name:  name,
		Value: util.HexValue(fmt.Sprintf("0x%X", v)),
	}
}

func parseDQ(dqfields []string, cmd *Command, dqsetting *[]string) {
	if len(dqfields) > 0 {
		cmd.HasDQ = true
		cmd.DQSetting = *dqsetting
		*dqsetting = nil

		dqlen := 0
		for _, v := range dqfields {
			if strings.HasPrefix(v, "0x") {
				cmd.DQDataTopo = append(cmd.DQDataTopo, v)
				dqlen++
			}
		}

		if dqlen > 0 {
			dqfields = dqfields[dqlen:]

			//如果全0 ,不要存, 預設為空
			needDm := false
			for _, v := range dqfields {
				if v != "0" {
					needDm = true
					break
				}
			}

			if needDm {
				cmd.DQDmTopo = append(cmd.DQDmTopo, dqfields...)
			} else {
				cmd.DQDmTopo = nil
			}

		}
	}
}

// --- BaseParser：共用解析邏輯 ---

type BaseParser struct{}

// ParseBGBAROW 解析 BG BA ROW（ACT 等命令共用）
func (p *BaseParser) ParseBGBAROW(fields []string, cmd *Command, vars VarResolver) error {
	if len(fields) < 4 {
		return fmt.Errorf("line %d: %s requires BG BA ROW, got: %s", cmd.LineNum, cmd.Type, cmd.Raw)
	}
	cmd.BankGroup = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.Bank = toAddr(fields[2], vars.Resolve(fields[2]))
	cmd.Row = toAddr(fields[3], vars.Resolve(fields[3]))
	return nil
}

// ParseBGBAROWCOL 解析 BG BA ROW COL + 可選 DQ（RD/WR 等命令共用）
func (p *BaseParser) ParseBGBAROWCOL(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	if len(fields) < 5 {
		return fmt.Errorf("line %d: %s requires BG BA ROW COL, got: %s", cmd.LineNum, cmd.Type, cmd.Raw)
	}
	cmd.BankGroup = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.Bank = toAddr(fields[2], vars.Resolve(fields[2]))
	cmd.Row = toAddr(fields[3], vars.Resolve(fields[3]))
	cmd.Column = toAddr(fields[4], vars.Resolve(fields[4]))
	parseDQ(fields[5:], cmd, dqsetting)
	return nil
}

// ParseBGBA 解析 BG BA（PREsb 等命令共用）
func (p *BaseParser) ParseBGBA(fields []string, cmd *Command, vars VarResolver) error {
	if len(fields) >= 3 {
		cmd.BankGroup = toAddr(fields[1], vars.Resolve(fields[1]))
		cmd.Bank = toAddr(fields[2], vars.Resolve(fields[2]))
	}
	return nil
}

// ParseDES 解析 DES/NOP 的 RepeatCnt
func (p *BaseParser) ParseDES(fields []string, cmd *Command) error {
	if len(fields) >= 2 {
		if n, err := strconv.Atoi(fields[1]); err == nil && n > 1 {
			cmd.RepeatCnt = n
		}
	}
	return nil
}

// ParseOpCode 解析只帶 OpCode 的命令（MPC/VrefCA/VrefCS）
func (p *BaseParser) ParseOpCode(fields []string, cmd *Command, vars VarResolver) error {
	if len(fields) >= 2 {
		cmd.OpCode = toAddr(fields[1], vars.Resolve(fields[1]))
	}
	return nil
}
