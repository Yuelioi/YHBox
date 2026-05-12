package fish

type State int

const (
	IDLE       State = iota // 0
	SETUP                   // 1
	WAITING                 // 2
	FISHING                 // 3
	RESULT                  // 4
	RECOVERING              // 5
	SHOPSELL                // 6  鱼仓已满后开商店一键出售流程
	BUYBAIT                 // 7  鱼饵不足后开鱼饵商店购买流程
	CHANGEBAIT              // 8  买完鱼饵后按 E 进更换 UI 装备万能鱼饵
)

const (
	DebugIDLE       = 1 << IDLE       // 1
	DebugSETUP      = 1 << SETUP      // 2
	DebugWAITING    = 1 << WAITING    // 4
	DebugFISHING    = 1 << FISHING    // 8
	DebugRESULT     = 1 << RESULT     // 16
	DebugRECOVERING = 1 << RECOVERING // 32
	DebugSHOPSELL   = 1 << SHOPSELL   // 64
	DebugBUYBAIT    = 1 << BUYBAIT    // 128
	DebugCHANGEBAIT = 1 << CHANGEBAIT // 256
	DebugALL        = DebugIDLE | DebugSETUP | DebugWAITING | DebugFISHING | DebugRESULT | DebugRECOVERING | DebugSHOPSELL | DebugBUYBAIT | DebugCHANGEBAIT
)

func (s State) String() string {
	switch s {
	case IDLE:
		return "READY "
	case SETUP:
		return "SETUP "
	case WAITING:
		return "CAST  "
	case FISHING:
		return "FIGHT "
	case RESULT:
		return "RESULT"
	case RECOVERING:
		return "RECOV "
	case SHOPSELL:
		return "SELL  "
	case BUYBAIT:
		return "BUY   "
	case CHANGEBAIT:
		return "EQUIP "
	}
	return "??????"
}
