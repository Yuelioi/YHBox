package runtime

import "testing"

func TestPinTypeCompat(t *testing.T) {
	cases := []struct {
		from, to       PinType
		wantAllow, wantWarn bool
	}{
		{PinNumber, PinNumber, true, false},
		{PinNumber, PinBool, true, true},     // implicit !=0
		{PinNumber, PinString, true, true},   // implicit ToString
		{PinNumber, PinPoint, false, false},  // hard reject
		{PinString, PinNumber, false, false}, // must use ToNumber
		{PinAny, PinNumber, true, false},     // any flows freely
		{PinPoint, PinAny, true, false},
	}
	for _, c := range cases {
		allow, warn := PinTypeCompat(c.from, c.to)
		if allow != c.wantAllow || warn != c.wantWarn {
			t.Errorf("%s→%s: got (allow=%v warn=%v), want (%v %v)",
				c.from, c.to, allow, warn, c.wantAllow, c.wantWarn)
		}
	}
}
