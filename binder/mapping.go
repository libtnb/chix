package binder

import (
	"fmt"
	"maps"
	"mime/multipart"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gofiber/schema"
	utilsstrings "github.com/gofiber/utils/v2/strings"
)

// ParserConfig form decoder config for SetParserDecoder
type ParserConfig struct {
	// SetAliasTag is ignored: each binding source keeps its own alias tag
	// (query/header/cookie/form/uri), so a single global tag is incompatible
	// with per-source binding.
	SetAliasTag       string
	ParserType        []ParserType
	IgnoreUnknownKeys bool
	ZeroEmpty         bool
}

// ParserType require two element, type and converter for register.
// Use ParserType with BodyParser for parsing custom type in form data.
type ParserType struct {
	CustomType any
	Converter  func(string) reflect.Value
}

var (
	// decoderPoolMap helps to improve binders performance. The pools are held
	// behind atomic pointers so SetParserDecoder can swap them without racing
	// against concurrent requests; the map itself is only written during init.
	decoderPoolMap = map[string]*atomic.Pointer[sync.Pool]{}
	// tags is used to classify parser's pool
	tags = []string{bindingHeader, bindingCookie, bindingQuery, bindingForm, bindingURI}
)

// SetParserDecoder allow globally change the option of form decoder, update decoderPool
func SetParserDecoder(parserConfig ParserConfig) {
	for _, tag := range tags {
		pool := &sync.Pool{New: func() any {
			return decoderBuilder(tag, parserConfig)
		}}
		decoderPoolMap[tag].Store(pool)
	}
}

func decoderBuilder(aliasTag string, parserConfig ParserConfig) any {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(parserConfig.IgnoreUnknownKeys)
	// Bake the per-source tag once (pools are keyed per tag); doing it per
	// request would reset the decoder's type cache. ParserConfig.SetAliasTag is
	// not honored here: one global tag is incompatible with per-source binding.
	decoder.SetAliasTag(aliasTag)
	for _, v := range parserConfig.ParserType {
		decoder.RegisterConverter(reflect.ValueOf(v.CustomType).Interface(), v.Converter)
	}
	decoder.ZeroEmpty(parserConfig.ZeroEmpty)
	return decoder
}

func init() {
	for _, tag := range tags {
		pool := &sync.Pool{New: func() any {
			return decoderBuilder(tag, ParserConfig{
				IgnoreUnknownKeys: true,
				ZeroEmpty:         true,
			})
		}}
		ptr := &atomic.Pointer[sync.Pool]{}
		ptr.Store(pool)
		decoderPoolMap[tag] = ptr
	}
}

// parse data into the map or struct
func parse(aliasTag string, out any, data map[string][]string, files ...map[string][]*multipart.FileHeader) error {
	ptrVal := reflect.ValueOf(out)

	// Get pointer value
	if ptrVal.Kind() == reflect.Pointer {
		ptrVal = ptrVal.Elem()
	}

	// Parse into the map
	if ptrVal.Kind() == reflect.Map && ptrVal.Type().Key().Kind() == reflect.String {
		return parseToMap(ptrVal, data)
	}

	// Parse into the struct
	return parseToStruct(aliasTag, out, data, files...)
}

// Parse data into the struct with gofiber/schema
func parseToStruct(aliasTag string, out any, data map[string][]string, files ...map[string][]*multipart.FileHeader) error {
	// Get decoder from pool. Keep the loaded pool in a local so Get and Put
	// always operate on the same pool even if SetParserDecoder swaps it.
	pool := decoderPoolMap[aliasTag].Load()
	schemaDecoder := pool.Get().(*schema.Decoder)
	defer pool.Put(schemaDecoder)

	// The alias tag is baked in at build time (see decoderBuilder); setting it
	// here would reset the decoder's type cache on every request.
	if err := schemaDecoder.Decode(out, data, files...); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}

// Parse data into the map
// thanks to https://github.com/gin-gonic/gin/blob/master/binding/binding.go
func parseToMap(target reflect.Value, data map[string][]string) error {
	if !target.IsValid() {
		return ErrInvalidDestinationValue
	}

	if target.Kind() == reflect.Interface && !target.IsNil() {
		target = target.Elem()
	}

	if target.Kind() != reflect.Map || target.Type().Key().Kind() != reflect.String {
		return nil // nothing to do for non-map destinations
	}

	if target.IsNil() {
		if !target.CanSet() {
			return ErrMapNilDestination
		}
		target.Set(reflect.MakeMap(target.Type()))
	}

	switch target.Type().Elem().Kind() {
	case reflect.Slice:
		newMap, ok := target.Interface().(map[string][]string)
		if !ok {
			return ErrMapNotConvertible
		}

		maps.Copy(newMap, data)
	case reflect.String:
		newMap, ok := target.Interface().(map[string]string)
		if !ok {
			return ErrMapNotConvertible
		}

		for k, v := range data {
			if len(v) == 0 {
				newMap[k] = ""
				continue
			}

			newMap[k] = v[len(v)-1]
		}
	default:
		// Interface element maps (e.g. map[string]any) are left untouched because
		// the binder cannot safely infer element conversions without mutating
		// caller-provided values. These destinations therefore see a successful
		// no-op parse.
		return nil // it's not necessary to check all types
	}

	return nil
}

func parseParamSquareBrackets(k string) (string, error) {
	var sb strings.Builder

	kbytes := []byte(k)
	openBracketsCount := 0

	for i, b := range kbytes {
		if b == '[' {
			openBracketsCount++
			if i+1 < len(kbytes) && kbytes[i+1] != ']' {
				sb.WriteByte('.')
			}
			continue
		}

		if b == ']' {
			openBracketsCount--
			if openBracketsCount < 0 {
				return "", ErrUnmatchedBrackets
			}
			continue
		}

		sb.WriteByte(b)
	}

	if openBracketsCount > 0 {
		return "", ErrUnmatchedBrackets
	}

	return sb.String(), nil
}

func isStringKeyMap(t reflect.Type) bool {
	return t.Kind() == reflect.Map && t.Key().Kind() == reflect.String
}

func isExported(f *reflect.StructField) bool {
	if f == nil {
		return false
	}
	return f.PkgPath == ""
}

func fieldName(f *reflect.StructField, aliasTag string) string {
	if f == nil {
		return ""
	}

	name := f.Tag.Get(aliasTag)
	if name == "" {
		name = f.Name
	} else if first, _, found := strings.Cut(name, ","); found {
		name = first
	}

	return utilsstrings.ToLower(name)
}

type fieldInfo struct {
	names       map[string]reflect.Kind
	nestedKinds map[reflect.Kind]struct{}
}

func unwrapType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

var (
	headerFieldCache sync.Map
	cookieFieldCache sync.Map
	queryFieldCache  sync.Map
	formFieldCache   sync.Map
	uriFieldCache    sync.Map
)

func getFieldCache(aliasTag string) *sync.Map {
	switch aliasTag {
	case bindingHeader:
		return &headerFieldCache
	case bindingCookie:
		return &cookieFieldCache
	case bindingForm:
		return &formFieldCache
	case bindingURI:
		return &uriFieldCache
	case bindingQuery:
		return &queryFieldCache
	}

	panic("unknown alias tag: " + aliasTag)
}

func buildFieldInfo(t reflect.Type, aliasTag string) fieldInfo {
	info := fieldInfo{
		names:       make(map[string]reflect.Kind),
		nestedKinds: make(map[reflect.Kind]struct{}),
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !isExported(&f) {
			continue
		}
		fieldType := unwrapType(f.Type)
		info.names[fieldName(&f, aliasTag)] = fieldType.Kind()

		if fieldType.Kind() == reflect.Struct {
			for j := 0; j < fieldType.NumField(); j++ {
				sf := fieldType.Field(j)
				if !isExported(&sf) {
					continue
				}
				nestedType := unwrapType(sf.Type)
				info.nestedKinds[nestedType.Kind()] = struct{}{}
			}
		}
	}

	return info
}

func equalFieldType(out any, kind reflect.Kind, key, aliasTag string) bool {
	typ := reflect.TypeOf(out).Elem()
	key = utilsstrings.ToLower(key)

	if isStringKeyMap(typ) {
		return true
	}

	if typ.Kind() != reflect.Struct {
		return false
	}

	cache := getFieldCache(aliasTag)
	val, ok := cache.Load(typ)
	if !ok {
		info := buildFieldInfo(typ, aliasTag)
		val, _ = cache.LoadOrStore(typ, info)
	}

	info, ok := val.(fieldInfo)
	if !ok {
		return false
	}

	if k, ok := info.names[key]; ok && k == kind {
		return true
	}
	if _, ok := info.nestedKinds[kind]; ok {
		return true
	}

	return false
}

// FilterFlags returns the media type value by trimming any parameters from a Content-Type header.
func FilterFlags(content string) string {
	if i := strings.IndexAny(content, " ;"); i >= 0 {
		return content[:i]
	}
	return content
}

func formatBindData[T, K any](aliasTag string, out any, data map[string][]T, key string, value K, enableSplitting, supportBracketNotation bool) error { //nolint:revive // it's okay
	if supportBracketNotation && strings.IndexByte(key, '[') >= 0 {
		parsed, err := parseParamSquareBrackets(key)
		if err != nil {
			return err
		}
		key = parsed
	}

	switch v := any(value).(type) {
	case string:
		dataMap, ok := any(data).(map[string][]string)
		if !ok {
			return fmt.Errorf("unsupported value type: %T", value)
		}

		assignBindData(aliasTag, out, dataMap, key, v, enableSplitting)
	case []string:
		dataMap, ok := any(data).(map[string][]string)
		if !ok {
			return fmt.Errorf("unsupported value type: %T", value)
		}

		for _, val := range v {
			assignBindData(aliasTag, out, dataMap, key, val, enableSplitting)
		}
	case []*multipart.FileHeader:
		for _, val := range v {
			valT, ok := any(val).(T)
			if !ok {
				return fmt.Errorf("unsupported value type: %T", value)
			}
			data[key] = append(data[key], valT)
		}
	default:
		return fmt.Errorf("unsupported value type: %T", value)
	}

	return nil
}

func assignBindData(aliasTag string, out any, data map[string][]string, key, value string, enableSplitting bool) { //nolint:revive // it's okay
	if enableSplitting && strings.IndexByte(value, ',') >= 0 && equalFieldType(out, reflect.Slice, key, aliasTag) {
		for v := range strings.SplitSeq(value, ",") {
			data[key] = append(data[key], v)
		}
	} else {
		data[key] = append(data[key], value)
	}
}
