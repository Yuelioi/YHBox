package ai

import (
	"encoding/json"

	"yotta/internal/node"
	"yotta/internal/services/llm"
)

func init() { node.Register(&AI{}) }

const (
	pinIn  = "In"
	pinDone = "Done"
	pinFail = "Fail"

	inConnection  = "Connection"
	inModel       = "Model"
	inSystem      = "System"
	inUser        = "User"
	inMode        = "Mode"
	inTemperature = "Temperature"
	inMaxTokens   = "MaxTokens"

	outText  = "Text"
	outError = "Error"
	outCode  = "Code"
)

// staticAIKeys — merged inputs 里非动态输入的 key(静态 pin + config 元数据), 收集动态输入时跳过。
var staticAIKeys = map[string]bool{
	inConnection: true, inModel: true, inSystem: true, inUser: true,
	inMode: true, inTemperature: true, inMaxTokens: true,
	"Inputs": true, "Outputs": true,
}

type AI struct{}

func (AI) Spec() node.Spec {
	return node.Spec{
		Kind:     "AI",
		Category: "AI",
		Inputs: []node.InputSpec{
			{Name: pinIn, Type: node.TypeExec},
			{Name: inConnection, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "ai-connection"}},
			{Name: inModel, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: inSystem, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "textarea"}},
			{Name: inUser, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "textarea"}},
			{Name: inMode, Type: "String", Default: "auto", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown", Props: node.MarshalProps(node.DropdownProps{
					Options: []node.EnumOption{{Value: "auto"}, {Value: "native"}, {Value: "prompt"}},
				})}},
			{Name: inTemperature, Type: "Number", Default: json.Number("0")},
			{Name: inMaxTokens, Type: "Integer", Default: json.Number("0")},
		},
		Outputs: []node.OutputSpec{
			{Name: pinDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: outText, Type: "String"},
			}},
			{Name: pinFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: outError, Type: "String"},
				{Name: outCode, Type: "String"},
				{Name: outText, Type: "String", Optional: true},
			}},
		},
		DynamicInputs:     true,
		DynamicDataFields: true,
	}
}

func (AI) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	svc := ctx.AI()
	if svc == nil {
		return nil, node.Failf(node.CodeError, nil, "AI provider service not wired")
	}
	provider, err := svc.Provider(in.String(inConnection))
	if err != nil {
		return fireFail(ctx, err, ""), nil
	}

	dyn := collectDynamicInputs(in)
	imageNames := map[string]bool{} // C6: Image 类型动态输入名集
	system, err := renderPromptTemplate(in.String(inSystem), dyn, imageNames)
	if err != nil {
		return nil, node.Failf(node.CodeError, err, "AI system template: %v", err)
	}
	user, err := renderPromptTemplate(in.String(inUser), dyn, imageNames)
	if err != nil {
		return nil, node.Failf(node.CodeError, err, "AI user template: %v", err)
	}

	var msgs []llm.Message
	if system != "" {
		msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: system})
	}
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: user})

	req := llm.ChatRequest{
		Model:       in.String(inModel),
		Messages:    msgs,
		Temperature: in.Float64(inTemperature),
		MaxTokens:   in.Int(inMaxTokens),
	}

	decls := parseOutputDecls(in)
	if len(decls) == 0 {
		resp, err := provider.Chat(ctx.Context(), req)
		if err != nil {
			return fireFail(ctx, err, ""), nil
		}
		return ctx.Out(pinDone).Set(outText, resp.Text).Fire(), nil
	}

	// 结构化: 声明了输出字段 → 要模型出 JSON, 逐字段 coerce 后 Set。
	resp, fields, err := provider.ChatStructured(ctx.Context(), req, buildSchema(decls), in.String(inMode))
	if err != nil {
		return fireFail(ctx, err, resp.Text), nil // 原文带进 Fail.Text 供下游
	}
	out := ctx.Out(pinDone).Set(outText, resp.Text)
	for _, d := range decls {
		raw, ok := fields[d.Name]
		if !ok {
			continue // 模型没返该字段 → 稀疏不 Set(变量留旧值)
		}
		val, cerr := coerceOutput(raw, d.Type)
		if cerr != nil {
			return fireFail(ctx, &llm.CodedError{Kind: llm.KindBadRequest, Err: cerr}, resp.Text), nil
		}
		out = out.Set(d.Name, val)
	}
	return out.Fire(), nil
}

// fireFail 显式 fire Fail 出口, 带 Error/Code(=llm.ErrKind)/可选原文 Text。
// 用显式 fire(而非返 error)以便携带 Text(结构化失败时原文供下游消费)。
func fireFail(ctx node.Ctx, err error, rawText string) node.Outputs {
	b := ctx.Out(pinFail).
		Set(outError, err.Error()).
		Set(outCode, string(llm.KindOf(err)))
	if rawText != "" {
		b = b.Set(outText, rawText)
	}
	return b.Fire()
}

func collectDynamicInputs(in node.Inputs) map[string]any {
	out := map[string]any{}
	for _, k := range in.Keys() {
		if staticAIKeys[k] {
			continue
		}
		out[k] = in.Raw(k)
	}
	return out
}
