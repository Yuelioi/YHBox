package container

import "testing"

// TestPinTypeCompat 守护单一源 matrix — 任何修改要同时反映 FE pinTypeCompat (TS).
func TestPinTypeCompat(t *testing.T) {
	cases := []struct {
		from, to            string
		wantAllow, wantWarn bool
	}{
		{"number", "number", true, false},
		{"number", "bool", true, true},     // implicit !=0
		{"number", "string", true, true},   // implicit ToString
		{"number", "point", false, false},  // hard reject
		{"string", "number", false, false}, // must use ToNumber
		{"any", "number", true, false},     // any flows freely
		{"point", "any", true, false},
		{"bool", "number", true, true}, // implicit 0/1
		{"bool", "string", true, true}, // implicit "true"/"false"
		{"bool", "point", false, false},
	}
	for _, c := range cases {
		allow, warn := PinTypeCompat(c.from, c.to)
		if allow != c.wantAllow || warn != c.wantWarn {
			t.Errorf("%s→%s: got (allow=%v warn=%v), want (%v %v)",
				c.from, c.to, allow, warn, c.wantAllow, c.wantWarn)
		}
	}
}
