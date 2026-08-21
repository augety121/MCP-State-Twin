package limits

import (
	"fmt"
	"math"
	"reflect"

	"github.com/augety121/mcp-state-twin/internal/canonical"
)

const (
	Format  = "statetwin.dev/resource-profile/v1alpha1"
	Version = "local-preview-v1"

	MaxSpecBytes         = 1 << 20
	MaxToolCount         = 256
	MaxEntityTypeCount   = 256
	MaxEntitiesPerBranch = 100_000
	MaxStateBytes        = 16 << 20
	MaxInputBytes        = 1 << 20
	MaxOutputBytes       = 1 << 20
	MaxJSONDepth         = 64
	MaxJSONMembers       = 1_000_000
	MaxSchemaBytes       = 256 << 10
	MaxSchemaDepth       = 32
	MaxExpressionBytes   = 4096
	MaxExpressionCost    = 10_000
	MaxEffectsPerCall    = 128
	MaxQueryResultItems  = 10_000
	MaxDiffEntries       = 10_000
	MaxDiffBytes         = 8 << 20
	MaxAuditEventBytes   = 2 << 20
	MaxReportBytes       = 32 << 20
	MaxScenarioSteps     = 256
	MaxFaultRules        = 128
	MaxForks             = 1024
	MaxSnapshots         = 1024
	MaxConcurrentCalls   = 1
	MaxBundleFiles       = 0
	MaxBundleCompressed  = 0
	MaxBundleExtracted   = 0
	MaxCassetteBytes     = 0
	MaxScheduledEvents   = 0
	FutureMaxTaskCount   = 0
)

// Profile is part of deterministic environment identity. A zero value means
// the corresponding feature is disabled, not unlimited.
type Profile struct {
	Format               string `json:"format"`
	Version              string `json:"version"`
	MaxSpecBytes         int    `json:"maxSpecBytes"`
	MaxToolCount         int    `json:"maxToolCount"`
	MaxEntityTypeCount   int    `json:"maxEntityTypeCount"`
	MaxEntitiesPerBranch int    `json:"maxEntitiesPerBranch"`
	MaxStateBytes        int    `json:"maxStateBytes"`
	MaxInputBytes        int    `json:"maxInputBytes"`
	MaxOutputBytes       int    `json:"maxOutputBytes"`
	MaxJSONDepth         int    `json:"maxJSONDepth"`
	MaxJSONMembers       int    `json:"maxJSONMembers"`
	MaxSchemaBytes       int    `json:"maxSchemaBytes"`
	MaxSchemaDepth       int    `json:"maxSchemaDepth"`
	MaxExpressionBytes   int    `json:"maxExpressionBytes"`
	MaxExpressionCost    uint64 `json:"maxExpressionCost"`
	MaxEffectsPerCall    int    `json:"maxEffectsPerCall"`
	MaxQueryResultItems  int    `json:"maxQueryResultItems"`
	MaxDiffEntries       int    `json:"maxDiffEntries"`
	MaxDiffBytes         int    `json:"maxDiffBytes"`
	MaxAuditEventBytes   int    `json:"maxAuditEventBytes"`
	MaxReportBytes       int    `json:"maxReportBytes"`
	MaxScenarioSteps     int    `json:"maxScenarioSteps"`
	MaxScheduledEvents   int    `json:"maxScheduledEvents"`
	MaxFaultRules        int    `json:"maxFaultRules"`
	MaxForks             int    `json:"maxForks"`
	MaxSnapshots         int    `json:"maxSnapshots"`
	MaxConcurrentCalls   int    `json:"maxConcurrentCalls"`
	MaxCassetteBytes     int    `json:"maxCassetteBytes"`
	MaxBundleFiles       int    `json:"maxBundleFiles"`
	MaxBundleCompressed  int    `json:"maxBundleCompressedBytes"`
	MaxBundleExtracted   int    `json:"maxBundleExtractedBytes"`
	FutureMaxTaskCount   int    `json:"futureMaxTaskCount"`
}

func Default() Profile {
	return Profile{
		Format: Format, Version: Version,
		MaxSpecBytes: MaxSpecBytes, MaxToolCount: MaxToolCount,
		MaxEntityTypeCount:   MaxEntityTypeCount,
		MaxEntitiesPerBranch: MaxEntitiesPerBranch,
		MaxStateBytes:        MaxStateBytes, MaxInputBytes: MaxInputBytes,
		MaxOutputBytes: MaxOutputBytes, MaxJSONDepth: MaxJSONDepth,
		MaxJSONMembers: MaxJSONMembers, MaxSchemaBytes: MaxSchemaBytes,
		MaxSchemaDepth: MaxSchemaDepth, MaxExpressionBytes: MaxExpressionBytes,
		MaxExpressionCost: MaxExpressionCost, MaxEffectsPerCall: MaxEffectsPerCall,
		MaxQueryResultItems: MaxQueryResultItems, MaxDiffEntries: MaxDiffEntries,
		MaxDiffBytes: MaxDiffBytes, MaxAuditEventBytes: MaxAuditEventBytes,
		MaxReportBytes: MaxReportBytes, MaxScenarioSteps: MaxScenarioSteps,
		MaxScheduledEvents: MaxScheduledEvents, MaxFaultRules: MaxFaultRules,
		MaxForks: MaxForks, MaxSnapshots: MaxSnapshots,
		MaxConcurrentCalls: MaxConcurrentCalls, MaxCassetteBytes: MaxCassetteBytes,
		MaxBundleFiles: MaxBundleFiles, MaxBundleCompressed: MaxBundleCompressed,
		MaxBundleExtracted: MaxBundleExtracted, FutureMaxTaskCount: FutureMaxTaskCount,
	}
}

func Digest() (string, error) {
	return canonical.Digest(Default())
}

// ValidateJSON rejects values outside the current bounded JSON domain. The
// member budget counts each object entry and array element.
func ValidateJSON(value any, maxBytes int) error {
	data, err := canonical.JSON(value)
	if err != nil {
		return fmt.Errorf("canonical JSON: %w", err)
	}
	if len(data) > maxBytes {
		return fmt.Errorf("bytes %d exceed limit %d", len(data), maxBytes)
	}
	members := 0
	if err := walk(value, 0, &members); err != nil {
		return err
	}
	return nil
}

func walk(value any, depth int, members *int) error {
	if depth > MaxJSONDepth {
		return fmt.Errorf("JSON depth exceeds limit %d", MaxJSONDepth)
	}
	if value == nil {
		return nil
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer {
		if reflected.IsNil() {
			return nil
		}
		return walk(reflected.Elem().Interface(), depth, members)
	}
	switch reflected.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil
	case reflect.Float32, reflect.Float64:
		if math.IsNaN(reflected.Float()) || math.IsInf(reflected.Float(), 0) {
			return fmt.Errorf("JSON contains a non-finite number")
		}
		return nil
	case reflect.Struct:
		data, err := canonical.JSON(value)
		if err != nil {
			return err
		}
		var normalized any
		if err := unmarshalCanonical(data, &normalized); err != nil {
			return err
		}
		return walk(normalized, depth, members)
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("JSON object has non-string keys")
		}
		*members += reflected.Len()
		if *members > MaxJSONMembers {
			return fmt.Errorf("JSON members exceed limit %d", MaxJSONMembers)
		}
		iterator := reflected.MapRange()
		for iterator.Next() {
			if err := walk(iterator.Value().Interface(), depth+1, members); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		*members += reflected.Len()
		if *members > MaxJSONMembers {
			return fmt.Errorf("JSON members exceed limit %d", MaxJSONMembers)
		}
		for i := 0; i < reflected.Len(); i++ {
			if err := walk(reflected.Index(i).Interface(), depth+1, members); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported %s outside JSON domain", reflected.Kind())
	}
}
