package runtime

type PinType string

const (
	PinNumber PinType = "number"
	PinBool   PinType = "bool"
	PinString PinType = "string"
	PinPoint  PinType = "point"
	PinAny    PinType = "any"
)

// PinTypeCompat returns (allowed, isCoercionWarning).
// Matrix per spec §2.3.
func PinTypeCompat(from, to PinType) (allow, warn bool) {
	if from == to || from == PinAny || to == PinAny {
		return true, false
	}
	switch from {
	case PinNumber:
		switch to {
		case PinBool, PinString:
			return true, true
		}
	case PinBool:
		switch to {
		case PinNumber, PinString:
			return true, true
		}
	}
	return false, false
}
