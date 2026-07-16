package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/datatype"
)

const (
	TemplateMatchTypeID = "https://schemas.yotta.dev/types/vision/template-match/v1"
	QRCodeTypeID        = "https://schemas.yotta.dev/types/vision/qr-code/v1"
	ColorRangeTypeID    = "https://schemas.yotta.dev/types/vision/color-range/v1"
	ColorBlobTypeID     = "https://schemas.yotta.dev/types/vision/color-blob/v1"
)

type visionTypeSet struct {
	templateMatch datatype.Definition
	qrCode        datatype.Definition
	colorRange    datatype.Definition
	colorBlob     datatype.Definition
}

func sealVisionTypes() (visionTypeSet, error) {
	pointSchema := `{"type":"object","properties":{"x":{"type":"number"},"y":{"type":"number"},"unit":{"const":"px"}},"required":["x","y","unit"],"additionalProperties":false}`
	regionSchema := `{"type":"object","properties":{"x":{"type":"number"},"y":{"type":"number"},"width":{"type":"number","minimum":0},"height":{"type":"number","minimum":0},"unit":{"const":"px"}},"required":["x","y","width","height","unit"],"additionalProperties":false}`
	seal := func(typeID string, schema string, authoring datatype.Authoring) (datatype.Definition, error) {
		return sealStructuredType(typeID, json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",%s
		}`, typeID+"/schema", schema)), authoring)
	}
	templateMatch, err := seal(TemplateMatchTypeID, fmt.Sprintf(`
		"type":"object","properties":{"score":{"type":"number","minimum":-1,"maximum":1},"center":%s,"bounds":%s},
		"required":["score","center","bounds"],"additionalProperties":false`, pointSchema, regionSchema), datatype.Authoring{
		TitleKey: "type.vision.templateMatch.title", DescriptionKey: "type.vision.templateMatch.description", Color: "#06b6d4", Icon: "scan-position",
	})
	if err != nil {
		return visionTypeSet{}, err
	}
	qrCode, err := seal(QRCodeTypeID, fmt.Sprintf(`
		"type":"object","properties":{"text":{"type":"string","maxLength":65536},"points":{"type":"array","items":%s,"maxItems":64}},
		"required":["text","points"],"additionalProperties":false`, pointSchema), datatype.Authoring{
		TitleKey: "type.vision.qrCode.title", DescriptionKey: "type.vision.qrCode.description", Color: "#0ea5e9", Icon: "qrcode",
	})
	if err != nil {
		return visionTypeSet{}, err
	}
	colorRange, err := seal(ColorRangeTypeID, `
		"oneOf":[
			{"type":"object","properties":{"space":{"const":"rgb"},"minimum":{"type":"array","prefixItems":[{"type":"integer","minimum":0,"maximum":255},{"type":"integer","minimum":0,"maximum":255},{"type":"integer","minimum":0,"maximum":255}],"items":false},"maximum":{"type":"array","prefixItems":[{"type":"integer","minimum":0,"maximum":255},{"type":"integer","minimum":0,"maximum":255},{"type":"integer","minimum":0,"maximum":255}],"items":false}},"required":["space","minimum","maximum"],"additionalProperties":false},
			{"type":"object","properties":{"space":{"const":"hsv"},"minimum":{"type":"array","prefixItems":[{"type":"integer","minimum":0,"maximum":360},{"type":"integer","minimum":0,"maximum":100},{"type":"integer","minimum":0,"maximum":100}],"items":false},"maximum":{"type":"array","prefixItems":[{"type":"integer","minimum":0,"maximum":360},{"type":"integer","minimum":0,"maximum":100},{"type":"integer","minimum":0,"maximum":100}],"items":false}},"required":["space","minimum","maximum"],"additionalProperties":false}
		]`, datatype.Authoring{
		TitleKey: "type.vision.colorRange.title", DescriptionKey: "type.vision.colorRange.description", Color: "#f43f5e", Icon: "color-swatch",
		EditorAdapter: "color-range",
		Examples: []json.RawMessage{
			json.RawMessage(`{"space":"rgb","minimum":[220,0,0],"maximum":[255,80,80]}`),
			json.RawMessage(`{"space":"hsv","minimum":[45,30,60],"maximum":[75,100,100]}`),
		},
	})
	if err != nil {
		return visionTypeSet{}, err
	}
	colorBlob, err := seal(ColorBlobTypeID, fmt.Sprintf(`
		"type":"object","properties":{"area":{"type":"integer","minimum":1},"center":%s,"bounds":%s},
		"required":["area","center","bounds"],"additionalProperties":false`, pointSchema, regionSchema), datatype.Authoring{
		TitleKey: "type.vision.colorBlob.title", DescriptionKey: "type.vision.colorBlob.description", Color: "#fb7185", Icon: "box-multiple",
	})
	if err != nil {
		return visionTypeSet{}, err
	}
	return visionTypeSet{templateMatch: templateMatch, qrCode: qrCode, colorRange: colorRange, colorBlob: colorBlob}, nil
}
