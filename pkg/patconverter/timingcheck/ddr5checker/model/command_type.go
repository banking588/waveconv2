package model

type CommandType string

type CommandRole int

const (
	//active
	RoleACT CommandRole = iota
	RoleACTX
	RoleACT1
	RoleACT2

	//write
	RoleWR
	RoleWRA
	RoleWRP
	RoleWRPX
	RoleWRPA
	RoleWRPAX
	RoleWR16
	RoleWR32
	RoleMWR

	//read
	RoleRD
	RoleRDC
	RoleRDX
	RoleRDA
	RoleRDAX
	RoleRD16
	RoleRD32

	//mrw
	RoleMRW
	RoleMRWX
	RoleMRW1
	RoleMRW2

	//mrr
	RoleMRR
	RoleMRRX
	RoleDES
	RoleNOP

	//pre
	RolePRE
	RolePREab
	RolePREsb
	RolePREpb

	//ref
	RoleREF
	RoleREFab
	RoleREFsb
	RoleREFpb
	RoleRFMab
	RoleRFMsb

	//SRE
	RoleSRE
	RoleSREF
	RoleSRX
	RolePDE
	RolePDX
	RoleAPDX

	//others
	RoleMPC
	RoleVrefCA
	RoleVrefCS
	RoleRESET
	RoleCKE_H
	RoleCKE_L
	RoleTestModeEntry
	RoleTestModeExit
	RoleWFF
	RoleRFF

	//CAS
	RoleCAS_WR //LP5 cas write CA4
	RoleCAS
	RoleCAS_RD //LP5 cas read CA5
	RoleCAS_FS //LP5 cas fs CA6
)

// ProtocolCommands 加上 Parser()
type ProtocolCommands interface {
	Protocol() string
	All() []CommandType
	RoleOf(t CommandType) (CommandRole, bool)
	Parser() CommandParser
	IsCASCommand(t CommandType) bool
	IsWriteCommand(t CommandType) bool
	IsReadCommand(t CommandType) bool
	IsDESOrNOP(t CommandType) bool
	IsPrecharge(t CommandType) bool
	IsRefresh(t CommandType) bool
	IsModeRegister(t CommandType) bool
}

// Protocol 協議類型
type Protocol string

const (
	ProtocolDDR5   Protocol = "DDR5"
	ProtocolLPDDR5 Protocol = "LPDDR5"
	ProtocolDDR6   Protocol = "DDR6"
)

// Tester 機台類型
type TesterType string

const (
	TesterT5833  TesterType = "T5833"
	Tester5503HS TesterType = "5503HS"
	TesterM5SSV  TesterType = "M5SSV"
)
