package model

import (
	"fmt"
)

type DDR5Commands struct {
	ACT    CommandType
	ACTX   CommandType
	WRP    CommandType
	WRPX   CommandType
	WRPA   CommandType
	WRPAX  CommandType
	MRW    CommandType
	MRWX   CommandType
	MRR    CommandType
	MRRX   CommandType
	WR     CommandType
	WRA    CommandType
	RD     CommandType
	RDX    CommandType
	RDA    CommandType
	RDAX   CommandType
	VrefCA CommandType
	VrefCS CommandType
	REFab  CommandType
	RFMab  CommandType
	REFsb  CommandType
	RFMsb  CommandType
	PREab  CommandType
	PREsb  CommandType
	PREpb  CommandType
	SRE    CommandType
	SREF   CommandType
	SRX    CommandType
	PDE    CommandType
	PDX    CommandType
	APDX   CommandType
	MPC    CommandType
	NOP    CommandType
	DES    CommandType

	PRE CommandType
	REF CommandType

	RESET CommandType
	CKE_H CommandType
	CKE_L CommandType

	TestModeEntry CommandType
	TestModeExit  CommandType

	roleMap map[CommandType]CommandRole
}

var DDR5 = func() *DDR5Commands {
	d := &DDR5Commands{
		ACT:           "ACT",
		ACTX:          "ACTX",
		WR:            "WR",
		WRA:           "WRA",
		WRP:           "WRP",
		WRPX:          "WRPX",
		WRPA:          "WRPA",
		WRPAX:         "WRPAX",
		RD:            "RD",
		RDX:           "RDX",
		RDA:           "RDA",
		RDAX:          "RDAX",
		MRW:           "MRW",
		MRWX:          "MRWX",
		MRR:           "MRR",
		MRRX:          "MRRX",
		DES:           "DES",
		NOP:           "NOP",
		PRE:           "PRE",
		PREab:         "PREab",
		PREsb:         "PREsb",
		PREpb:         "PREpb",
		REF:           "REF",
		REFab:         "REFab",
		REFsb:         "REFsb",
		RFMab:         "RFMab",
		RFMsb:         "RFMsb",
		MPC:           "MPC",
		VrefCA:        "VrefCA",
		VrefCS:        "VrefCS",
		SRE:           "SRE",
		SREF:          "SREF",
		SRX:           "SRX",
		PDE:           "PDE",
		PDX:           "PDX",
		APDX:          "APDX",
		RESET:         "RESET",
		CKE_H:         "CKE_H",
		CKE_L:         "CKE_L",
		TestModeEntry: "TM_MODE_ENTRY",
		TestModeExit:  "TM_MODE_EXIT",
	}
	d.roleMap = map[CommandType]CommandRole{
		d.ACT: RoleACT, d.ACTX: RoleACTX,
		d.WR: RoleWR, d.WRA: RoleWRA,
		d.WRP: RoleWRP, d.WRPX: RoleWRPX, d.WRPA: RoleWRPA, d.WRPAX: RoleWRPAX,
		d.RD: RoleRD, d.RDX: RoleRDX, d.RDA: RoleRDA, d.RDAX: RoleRDAX,
		d.MRW: RoleMRW, d.MRWX: RoleMRWX, d.MRR: RoleMRR, d.MRRX: RoleMRRX,
		d.DES: RoleDES, d.NOP: RoleNOP,
		d.PRE: RolePRE, d.PREab: RolePREab, d.PREsb: RolePREsb, d.PREpb: RolePREpb,
		d.REF: RoleREF, d.REFab: RoleREFab, d.REFsb: RoleREFsb,
		d.RFMab: RoleRFMab, d.RFMsb: RoleRFMsb,
		d.MPC:    RoleMPC,
		d.VrefCA: RoleVrefCA, d.VrefCS: RoleVrefCS,
		d.SRE: RoleSRE, d.SREF: RoleSREF, d.SRX: RoleSRX,
		d.PDE: RolePDE, d.PDX: RolePDX, d.APDX: RoleAPDX,
		d.RESET: RoleRESET, d.CKE_H: RoleCKE_H, d.CKE_L: RoleCKE_L,
		d.TestModeEntry: RoleTestModeEntry, d.TestModeExit: RoleTestModeExit,
	}
	return d
}()

func (d *DDR5Commands) RoleOf(t CommandType) (CommandRole, bool) {
	r, ok := d.roleMap[t]
	return r, ok
}

// 編譯期檢查：確保實作完整
var _ ProtocolCommands = (*DDR5Commands)(nil)

func (d *DDR5Commands) Protocol() string { return "DDR5" }

func (d *DDR5Commands) IsCASCommand(t CommandType) bool {
	return t == d.RD || t == d.RDA || t == d.WR || t == d.WRA
}

func (d *DDR5Commands) IsWriteCommand(t CommandType) bool {
	return t == d.WR || t == d.WRA
}

func (d *DDR5Commands) IsReadCommand(t CommandType) bool {
	return t == d.RD || t == d.RDA
}

func (d *DDR5Commands) IsDESOrNOP(t CommandType) bool {
	return t == d.DES || t == d.NOP
}

func (d *DDR5Commands) IsPrecharge(t CommandType) bool {
	return t == d.PREab || t == d.PREsb || t == d.PREpb || t == d.PRE
}

func (d *DDR5Commands) IsRefresh(t CommandType) bool {
	return t == d.REFab || t == d.REFsb || t == d.REF || t == d.RFMab || t == d.RFMsb
}

func (d *DDR5Commands) IsModeRegister(t CommandType) bool {
	return t == d.MRW || t == d.MRWX || t == d.MRR || t == d.MRRX
}

// ////////////////////////////////////
type DDR5Parser struct {
	BaseParser
}

func (p *DDR5Parser) ParseACT(fields []string, cmd *Command, vars VarResolver) error {
	// DDR5: ACT BG BA ROW
	return p.ParseBGBAROW(fields, cmd, vars)
}

func (p *DDR5Parser) ParseRD(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// DDR5: RD BG BA ROW COL [DQ...]
	return p.ParseBGBAROWCOL(fields, cmd, vars, dqsetting)
}

func (p *DDR5Parser) ParseWR(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// DDR5: WR BG BA ROW COL [DQ...]
	return p.ParseBGBAROWCOL(fields, cmd, vars, dqsetting)
}

func (p *DDR5Parser) ParseWRP(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// DDR5: WRP BG BA ROW COL [DQ...]
	return p.ParseBGBAROWCOL(fields, cmd, vars, dqsetting)
}

func (p *DDR5Parser) ParseMRW(fields []string, cmd *Command, vars VarResolver) error {
	// DDR5: MRW MA OP
	if len(fields) < 3 {
		return fmt.Errorf("line %d: MRW requires MA OP, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.MA = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.OpCode = toAddr(fields[2], vars.Resolve(fields[2]))
	return nil
}

func (p *DDR5Parser) ParseMRR(fields []string, cmd *Command, vars VarResolver, dqsetting *[]string) error {
	// DDR5: MRR MA OP CW [dummy] [DQ...]
	if len(fields) < 4 {
		return fmt.Errorf("line %d: MRR requires MA OP CW, got: %s", cmd.LineNum, cmd.Raw)
	}
	cmd.MA = toAddr(fields[1], vars.Resolve(fields[1]))
	cmd.OpCode = toAddr(fields[2], vars.Resolve(fields[2]))
	cmd.CW = toAddr(fields[3], vars.Resolve(fields[3]))
	if len(fields) > 5 {
		parseDQ(fields[5:], cmd, dqsetting)
	}
	return nil
}

func (p *DDR5Parser) ParseMPC(fields []string, cmd *Command, vars VarResolver) error {
	return p.ParseOpCode(fields, cmd, vars)
}

func (p *DDR5Parser) ParsePRE(fields []string, cmd *Command, vars VarResolver) error {
	// DDR5: PREsb/PREpb BG BA
	return p.ParseBGBA(fields, cmd, vars)
}

func (p *DDR5Parser) ParseREFsb(fields []string, cmd *Command, vars VarResolver) error {
	// DDR5: REFsb BG BA
	return p.ParseBGBA(fields, cmd, vars)
}

func (p *DDR5Parser) ParseVref(fields []string, cmd *Command, vars VarResolver) error {
	return p.ParseOpCode(fields, cmd, vars)
}

func (d *DDR5Commands) Parser() CommandParser {
	return &DDR5Parser{}
}

func (d *DDR5Commands) All() []CommandType {
	result := make([]CommandType, 0, len(d.roleMap))
	for cmd := range d.roleMap {
		result = append(result, cmd)
	}
	return result
}
