package nodes31

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
)

func divideNumbers(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	if b == 0 {
		return nil, &InlineFailure{Code: divisionByZeroCode, Cause: errors.New("division by zero")}
	}
	return finiteNumberResult(a / b)
}

func moduloNumbers(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	if b == 0 {
		return nil, &InlineFailure{Code: divisionByZeroCode, Cause: errors.New("modulo by zero")}
	}
	return finiteNumberResult(math.Mod(a, b))
}

func unaryNumber(evaluate func(float64) float64) InlineEvaluator {
	return func(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
		value, err := decodeFiniteNumber(inputs["value"])
		if err != nil {
			return nil, err
		}
		return finiteNumberResult(evaluate(value))
	}
}

var (
	negateNumber   = unaryNumber(func(value float64) float64 { return -value })
	absoluteNumber = unaryNumber(math.Abs)
	floorNumber    = unaryNumber(math.Floor)
	ceilingNumber  = unaryNumber(math.Ceil)
)

func minimumNumber(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	return finiteNumberResult(math.Min(a, b))
}

func maximumNumber(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	a, b, err := numberPair(inputs)
	if err != nil {
		return nil, err
	}
	return finiteNumberResult(math.Max(a, b))
}

func roundNumber(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	value, err := decodeFiniteNumber(inputs["value"])
	if err != nil {
		return nil, err
	}
	digits, err := decodeInteger(inputs["digits"])
	if err != nil {
		return nil, err
	}
	if digits > 15 {
		digits = 15
	} else if digits < -15 {
		digits = -15
	}
	factor := math.Pow10(int(digits))
	return finiteNumberResult(math.Round(value*factor) / factor)
}

func clampNumber(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	value, err := decodeFiniteNumber(inputs["value"])
	if err != nil {
		return nil, err
	}
	minimum, err := decodeFiniteNumber(inputs["minimum"])
	if err != nil {
		return nil, err
	}
	maximum, err := decodeFiniteNumber(inputs["maximum"])
	if err != nil {
		return nil, err
	}
	if minimum > maximum {
		minimum, maximum = maximum, minimum
	}
	return finiteNumberResult(math.Max(minimum, math.Min(maximum, value)))
}

func powerNumber(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	base, err := decodeFiniteNumber(inputs["base"])
	if err != nil {
		return nil, err
	}
	exponent, err := decodeFiniteNumber(inputs["exponent"])
	if err != nil {
		return nil, err
	}
	value := math.Pow(base, exponent)
	if math.IsNaN(value) {
		return nil, &InlineFailure{Code: mathDomainErrorCode, Cause: errors.New("power is outside the real-number domain")}
	}
	return finiteNumberResult(value)
}

func squareRoot(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	value, err := decodeFiniteNumber(inputs["value"])
	if err != nil {
		return nil, err
	}
	if value < 0 {
		return nil, &InlineFailure{Code: mathDomainErrorCode, Cause: errors.New("square root is outside the real-number domain")}
	}
	return finiteNumberResult(math.Sqrt(value))
}

func equalValues(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	equal, err := canonicalValuesEqual(inputs["a"], inputs["b"])
	if err != nil {
		return nil, err
	}
	return jsonResult(equal)
}

func notEqualValues(ctx context.Context, inputs map[string]json.RawMessage, config map[string]any) (map[string]json.RawMessage, error) {
	result, err := equalValues(ctx, inputs, config)
	if err != nil {
		return nil, err
	}
	var equal bool
	if err := json.Unmarshal(result["result"], &equal); err != nil {
		return nil, err
	}
	return jsonResult(!equal)
}

func canonicalValuesEqual(a, b json.RawMessage) (bool, error) {
	left, err := artifact.Canonicalize(a)
	if err != nil {
		return false, err
	}
	right, err := artifact.Canonicalize(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(left, right), nil
}

func replaceText(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, err := decodeString(inputs["text"])
	if err != nil {
		return nil, err
	}
	oldValue, err := decodeString(inputs["old"])
	if err != nil {
		return nil, err
	}
	newValue, err := decodeString(inputs["new"])
	if err != nil {
		return nil, err
	}
	var all bool
	if err := json.Unmarshal(inputs["all"], &all); err != nil {
		return nil, err
	}
	if oldValue == "" {
		return jsonResult(text)
	}
	if all {
		return jsonResult(strings.ReplaceAll(text, oldValue, newValue))
	}
	return jsonResult(strings.Replace(text, oldValue, newValue, 1))
}

func substringText(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, err := decodeString(inputs["text"])
	if err != nil {
		return nil, err
	}
	start, err := decodeInteger(inputs["start"])
	if err != nil {
		return nil, err
	}
	length, err := decodeInteger(inputs["length"])
	if err != nil {
		return nil, err
	}
	runes := []rune(text)
	if start < 0 {
		start = 0
	}
	if start > int64(len(runes)) {
		start = int64(len(runes))
	}
	end := int64(len(runes))
	if length == 0 {
		end = start
	} else if length > 0 && length < end-start {
		end = start + length
	}
	return jsonResult(string(runes[int(start):int(end)]))
}

func stringTransform(input string, transform func(string) string) InlineEvaluator {
	return func(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
		value, err := decodeString(inputs[input])
		if err != nil {
			return nil, err
		}
		return jsonResult(transform(value))
	}
}

var (
	trimText      = stringTransform("text", strings.TrimSpace)
	uppercaseText = stringTransform("text", strings.ToUpper)
	lowercaseText = stringTransform("text", strings.ToLower)
)

func indexOfText(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, search, err := stringPair(inputs, "text", "search")
	if err != nil {
		return nil, err
	}
	byteIndex := strings.Index(text, search)
	if byteIndex < 0 {
		return jsonResult(-1)
	}
	return jsonResult(utf8.RuneCountInString(text[:byteIndex]))
}

func startsWithText(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, prefix, err := stringPair(inputs, "text", "prefix")
	if err != nil {
		return nil, err
	}
	return jsonResult(strings.HasPrefix(text, prefix))
}

func endsWithText(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, suffix, err := stringPair(inputs, "text", "suffix")
	if err != nil {
		return nil, err
	}
	return jsonResult(strings.HasSuffix(text, suffix))
}

func regexMatch(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, pattern, err := stringPair(inputs, "text", "pattern")
	if err != nil {
		return nil, err
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &InlineFailure{Code: invalidRegexCode, Cause: errors.New("regular expression is invalid")}
	}
	return jsonResult(compiled.MatchString(text))
}

func regexExtract(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, pattern, err := stringPair(inputs, "text", "pattern")
	if err != nil {
		return nil, err
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &InlineFailure{Code: invalidRegexCode, Cause: errors.New("regular expression is invalid")}
	}
	match := compiled.FindStringSubmatch(text)
	if len(match) == 0 {
		return jsonResult("")
	}
	if len(match) > 1 {
		return jsonResult(match[1])
	}
	return jsonResult(match[0])
}

func valueToString(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	canonical, err := artifact.Canonicalize(inputs["value"])
	if err != nil {
		return nil, err
	}
	var text string
	if json.Unmarshal(canonical, &text) == nil {
		return jsonResult(text)
	}
	return jsonResult(string(canonical))
}

func stringToNumber(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, err := decodeString(inputs["text"])
	if err != nil {
		return nil, err
	}
	if text == "" || text != strings.TrimSpace(text) {
		return nil, &InlineFailure{Code: invalidNumberCode, Cause: errors.New("number string is not canonical")}
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, &InlineFailure{Code: invalidNumberCode, Cause: errors.New("number string is invalid")}
	}
	result, err := finiteNumberResult(value)
	if err != nil {
		return nil, &InlineFailure{Code: invalidNumberCode, Cause: errors.New("number string is outside the interoperable range")}
	}
	return result, nil
}

func stringToBoolean(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, err := decodeString(inputs["text"])
	if err != nil {
		return nil, err
	}
	switch text {
	case "true":
		return jsonResult(true)
	case "false":
		return jsonResult(false)
	default:
		return nil, &InlineFailure{Code: invalidBooleanCode, Cause: errors.New("boolean string must be true or false")}
	}
}

func parseJSONValue(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	text, err := decodeString(inputs["text"])
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, &InlineFailure{Code: invalidJSONCode, Cause: errors.New("JSON document is invalid")}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, &InlineFailure{Code: invalidJSONCode, Cause: errors.New("JSON document contains trailing values")}
	}
	canonical, err := artifact.Canonicalize([]byte(text))
	if err != nil {
		return nil, &InlineFailure{Code: invalidJSONCode, Cause: errors.New("JSON document is outside the interoperable profile")}
	}
	return map[string]json.RawMessage{"result": canonical}, nil
}

func stringifyJSONValue(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	canonical, err := artifact.Canonicalize(inputs["value"])
	if err != nil {
		return nil, err
	}
	return jsonResult(string(canonical))
}

type jsonPathStep struct {
	kind  byte
	field string
	index int
}

func queryJSONPath(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	path, err := decodeString(inputs["path"])
	if err != nil {
		return nil, err
	}
	steps, ok := parseBoundedJSONPath(path)
	if !ok {
		return nil, &InlineFailure{Code: invalidJSONPathCode, Cause: errors.New("JSON path is invalid or exceeds its budget")}
	}
	decoder := json.NewDecoder(bytes.NewReader(inputs["json"]))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	result, found := evaluateJSONPath(value, steps)
	if !found {
		result = nil
	}
	return jsonResult(result)
}

func parseBoundedJSONPath(path string) ([]jsonPathStep, bool) {
	if len(path) == 0 || len(path) > 1_024 || path[0] != '$' {
		return nil, false
	}
	steps := []jsonPathStep{}
	for index := 1; index < len(path); {
		if len(steps) >= 64 {
			return nil, false
		}
		switch path[index] {
		case '.':
			index++
			start := index
			for index < len(path) && path[index] != '.' && path[index] != '[' {
				index++
			}
			if start == index {
				return nil, false
			}
			steps = append(steps, jsonPathStep{kind: 'f', field: path[start:index]})
		case '[':
			end := strings.IndexByte(path[index:], ']')
			if end < 0 {
				return nil, false
			}
			raw := path[index+1 : index+end]
			if raw == "*" {
				steps = append(steps, jsonPathStep{kind: '*'})
			} else {
				item, err := strconv.Atoi(raw)
				if err != nil || item < 0 {
					return nil, false
				}
				steps = append(steps, jsonPathStep{kind: 'i', index: item})
			}
			index += end + 1
		default:
			return nil, false
		}
	}
	return steps, true
}

func evaluateJSONPath(value any, steps []jsonPathStep) (any, bool) {
	if len(steps) == 0 {
		return value, true
	}
	step, remaining := steps[0], steps[1:]
	switch step.kind {
	case 'f':
		object, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := object[step.field]
		if !ok {
			return nil, false
		}
		return evaluateJSONPath(next, remaining)
	case 'i':
		array, ok := value.([]any)
		if !ok || step.index >= len(array) {
			return nil, false
		}
		return evaluateJSONPath(array[step.index], remaining)
	case '*':
		array, ok := value.([]any)
		if !ok {
			return nil, false
		}
		result := make([]any, len(array))
		for index, item := range array {
			selected, found := evaluateJSONPath(item, remaining)
			if found {
				result[index] = selected
			}
		}
		return result, true
	default:
		return nil, false
	}
}

func selectValue(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	var condition bool
	if err := json.Unmarshal(inputs["condition"], &condition); err != nil {
		return nil, err
	}
	selected := inputs["when_false"]
	if condition {
		selected = inputs["when_true"]
	}
	return map[string]json.RawMessage{"result": append(json.RawMessage(nil), selected...)}, nil
}

type pointValue struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Unit string  `json:"unit"`
}

type regionValue struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

func makePoint(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	x, err := decodeFiniteNumber(inputs["x"])
	if err != nil {
		return nil, err
	}
	y, err := decodeFiniteNumber(inputs["y"])
	if err != nil {
		return nil, err
	}
	unit, err := decodeString(inputs["unit"])
	if err != nil {
		return nil, err
	}
	return jsonResult(pointValue{X: x, Y: y, Unit: unit})
}

func offsetPoint(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	point, err := decodePoint(inputs["point"])
	if err != nil {
		return nil, err
	}
	offsetX, err := decodeFiniteNumber(inputs["offset_x"])
	if err != nil {
		return nil, err
	}
	offsetY, err := decodeFiniteNumber(inputs["offset_y"])
	if err != nil {
		return nil, err
	}
	point.X += offsetX
	point.Y += offsetY
	if point.Unit == "ratio" {
		point.X = clampUnit(point.X)
		point.Y = clampUnit(point.Y)
	}
	if !finite(point.X) || !finite(point.Y) {
		return nil, &InlineFailure{Code: unrepresentableResultCode, Cause: errors.New("offset point is not representable")}
	}
	return jsonResult(point)
}

func pointDistance(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	begin, err := decodePoint(inputs["begin"])
	if err != nil {
		return nil, err
	}
	end, err := decodePoint(inputs["end"])
	if err != nil {
		return nil, err
	}
	if begin.Unit != end.Unit {
		return nil, &InlineFailure{Code: geometryUnitMismatchCode, Cause: errors.New("point units do not match")}
	}
	return finiteNumberResult(math.Hypot(end.X-begin.X, end.Y-begin.Y))
}

func regionAroundPoint(_ context.Context, inputs map[string]json.RawMessage, _ map[string]any) (map[string]json.RawMessage, error) {
	center, err := decodePoint(inputs["center"])
	if err != nil {
		return nil, err
	}
	width, err := decodeFiniteNumber(inputs["width"])
	if err != nil {
		return nil, err
	}
	height, err := decodeFiniteNumber(inputs["height"])
	if err != nil {
		return nil, err
	}
	width, height = math.Max(0, width), math.Max(0, height)
	region := regionValue{X: center.X - width/2, Y: center.Y - height/2, Width: width, Height: height, Unit: center.Unit}
	if center.Unit == "ratio" {
		region.Width, region.Height = clampUnit(region.Width), clampUnit(region.Height)
		region.X = math.Max(0, math.Min(1-region.Width, region.X))
		region.Y = math.Max(0, math.Min(1-region.Height, region.Y))
	}
	if !finite(region.X) || !finite(region.Y) || !finite(region.Width) || !finite(region.Height) {
		return nil, &InlineFailure{Code: unrepresentableResultCode, Cause: errors.New("region is not representable")}
	}
	return jsonResult(region)
}

func decodePoint(raw json.RawMessage) (pointValue, error) {
	var point pointValue
	if err := json.Unmarshal(raw, &point); err != nil {
		return pointValue{}, err
	}
	return point, nil
}

func decodeString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func clampUnit(value float64) float64 { return math.Max(0, math.Min(1, value)) }

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
