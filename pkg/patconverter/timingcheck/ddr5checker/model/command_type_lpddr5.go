package model

import "fmt"

type LPDDR5Commands struct {
	ACT1 CommandType
	ACT2 CommandType
	MRW1 CommandType
	MRW2 CommandType
	DES  CommandType
	RDC  CommandType
	NOP  CommandType
	PDE  CommandType

	PREPB CommandType
	PREAB CommandType

	REFPB CommandType
	REFAB CommandType

	MWR  CommandType
	WR16 CommandType
	RD16 CommandType
	MPC  CommandType

	SRE CommandType
	SRX CommandType
	MRR CommandType
	WFF CommandType
	RFF CommandType

	WS_WR CommandType
	WS_RD CommandType
	WS_FS CommandType

	roleMap map[CommandType]CommandRole
}

var LPDDR5 = func() *LPDDR5Commands {
	d := &LPDDR5Commands{
		DES:   "DES",
		NOP:   "NOP",
		PDE:   "PDE",
		ACT1:  "ACT1",
		ACT2:  "ACT2",
		PREPB: "PRE",
		PREAB: "PREA",
		REFPB: "REF",
		REFAB: "REFA",
		MWR:   "MWR",
		WR16:  "WR",
		RD16:  "RD",
		WS_FS: "WS_FS",
		WS_WR: "WS_WR",
		WS_RD: "WS_RD",
		MPC:   "MPC",
		SRE:   "SRE",
		SRX:   "SRX",
		MRW1:  "MRW1",
		MRW2:  "MRW2",
		MRR:   "MRR",
		WFF:   "WFF",
		RFF:   "RFF",
		RDC:   "RDC",
	}

	d.roleMap = map[CommandType]CommandRole{
		d.DES:   RoleDES,
		d.NOP:   RoleNOP,
		d.PDE:   RolePDE,
		d.ACT1:  RoleACT1,
		d.ACT2:  RoleACT2,
		d.PREPB: RolePREpb,
		d.PREAB: RolePREab,
		d.REFPB: RoleREFpb,
		d.REFAB: RoleREFab,
		d.MWR:   RoleMWR,
		d.WR16:  RoleWR16,
		d.RD16:  RoleRD16,
		d.SRE:   RoleSRE,
		d.SRX:   RoleSRX,
		d.MRW1:  RoleMRW1,
		d.MRW2:  RoleMRW2,
		d.MRR:   RoleMRR,
		d.WFF:   RoleWFF,
		d.RFF:   RoleRFF,
		d.RDC:   RoleRDC,
		d.WS_FS: RoleCAS_FS,
		d.WS_WR: RoleCAS_WR,
		d.WS_RD: RoleCAS_RD,
	}
	return d
}()

// 編譯期檢查：確保實作完整
var _ ProtocolCommands = (*LPDDR5Commands)(nil)

func (d *LPDDR5Commands) RoleOf(t CommandType) (CommandRole, bool) {
	r, ok := d.roleMap[t]
	return r, ok
}

func (d *LPDDR5Commands) Protocol() string { return "DDR5" }

func (d *LPDDR5Commands) IsCASCommand(t CommandType) bool {
	//return t == d.RD || t == d.RDA || t == d.WR || t == d.WRA
	return true
}

func (d *LPDDR5Commands) IsWriteCommand(t CommandType) bool {
	//return t == d.WR || t == d.WRA
	return true
}

func (d *LPDDR5Commands) IsReadCommand(t CommandType) bool {
	//return t == d.RD || t == d.RDA
	return true
}

func (d *LPDDR5Commands) IsDESOrNOP(t CommandType) bool {
	//return t == d.DES || t == d.NOP
	return true
}

func (d *LPDDR5Commands) IsPrecharge(t CommandType) bool {
	//return t == d.PREab || t == d.PREsb || t == d.PREpb || t == d.PRE
	return true
}

func (d *LPDDR5Commands) IsRefresh(t CommandType) bool {
	//return t == d.REFab || t == d.REFsb || t == d.REF || t == d.RFMab || t == d.RFMsb
	return true
}

func (d *LPDDR5Commands) IsModeRegister(t CommandType) bool {
	//return t == d.MRW || t == d.MRWX || t == d.MRR || t == d.MRRX
	return true
}

// ///////////////////////////////////////////////////////
type LPDDR5Parser struct {
	BaseParser
}

func (p *LPDDR5Parser) ParseACT(fields []string, cmd *Command, vars VarResolver) error {
	// LPDDR5: ACT BA ROW（沒有 BG）
	if len(fields) < 3 {
		return fmt.Errorf("line %d: ACT requires BA ROW, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.Bank = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.Row = toAddr(fields[2], vars.Resolve(fields[2]))
	return nil
}

func (p *LPDDR5Parser) ParseRD(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// LPDDR5: RD BA COL [DQ...]（沒有 BG、沒有 ROW）
	if len(fields) < 3 {
		return fmt.Errorf("line %d: RD requires BA COL, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.Bank = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.Column = toAddr(fields[2], vars.Resolve(fields[2]))
	parseDQ(fields[3:], cmd, dqsetting)
	return nil
}

func (p *LPDDR5Parser) ParseWR(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// LPDDR5: WR BA COL [DQ...]（沒有 BG、沒有 ROW）
	if len(fields) < 3 {
		return fmt.Errorf("line %d: WR requires BA COL, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.Bank = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.Column = toAddr(fields[2], vars.Resolve(fields[2]))
	parseDQ(fields[3:], cmd, dqsetting)
	return nil
}

func (p *LPDDR5Parser) ParseWRP(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	return p.ParseWR(fields, cmd, vars, dqsetting)
}

func (p *LPDDR5Parser) ParseMRW(fields []string, cmd *Command, vars VarResolver) error {
	// LPDDR5: MRW MA OP CW
	if len(fields) < 3 {
		return fmt.Errorf("line %d: MRW requires MA OP CW, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.MA = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.OpCode = toAddr(fields[2], vars.Resolve(fields[2]))
	return nil
}

func (p *LPDDR5Parser) ParseMRR(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// LPDDR5: MRR MA [DQ...]
	if len(fields) < 2 {
		return fmt.Errorf("line %d: MRR requires MA, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.MA = toAddr(fields[1], vars.Resolve(fields[1]))
	if len(fields) > 2 {
		parseDQ(fields[2:], cmd, dqsetting)
	}
	return nil
}

func (p *LPDDR5Parser) ParseMPC(fields []string, cmd *Command, vars VarResolver) error {
	return p.ParseOpCode(fields, cmd, vars)
}

func (p *LPDDR5Parser) ParseDES(fields []string, cmd *Command) error {
	return p.BaseParser.ParseDES(fields, cmd)
}

func (p *LPDDR5Parser) ParsePRE(fields []string, cmd *Command, vars VarResolver) error {
	// LPDDR5: PREpb BA（沒有 BG）
	if len(fields) >= 2 {
		cmd.Bank = toAddr(fields[1], vars.Resolve(fields[1]))
	}
	return nil
}

func (p *LPDDR5Parser) ParseREFsb(fields []string, cmd *Command, vars VarResolver) error {
	// LPDDR5: REFpb BA
	if len(fields) >= 2 {
		cmd.Bank = toAddr(fields[1], vars.Resolve(fields[1]))
	}
	return nil
}

func (p *LPDDR5Parser) ParseVref(fields []string, cmd *Command, vars VarResolver) error {
	return p.ParseOpCode(fields, cmd, vars)
}

func (d *LPDDR5Commands) Parser() CommandParser {
	return &LPDDR5Parser{}
}

func (d *LPDDR5Commands) All() []CommandType {
	result := make([]CommandType, 0, len(d.roleMap))
	for cmd := range d.roleMap {
		result = append(result, cmd)
	}
	return result
}
