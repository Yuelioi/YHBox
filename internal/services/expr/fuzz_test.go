package expr

import "testing"

func FuzzParseAndEval(f *testing.F) {
	for _, seed := range []string{"1 + 2 * 3", `name == "yotta" ? 1 : 0`, "max(1, 2)", "(((true)))"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 64<<10 {
			t.Skip()
		}
		node, err := Parse(source)
		if err != nil {
			return
		}
		IdentRefs(node)
		VarRefs(node)
		CallRefs(node)
		_, _ = Eval(node, MapEnv{"name": "yotta"})
	})
}
