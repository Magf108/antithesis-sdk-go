//go:build !no_antithesis_sdk

package assert

// A type for writing raw assertions. 
// GuidepostType allows the assertion to provide guidance to 
// the Antithesis platform when testing in Antithesis. 
// Regular users of the assert package should not use it.
type GuidepostType int

const (
	GuidepostMaximize GuidepostType = iota // Maximize (left - right) values
	GuidepostMinimize                      // Minimize (left - right) values
	GuidepostAll                           // Encourages fuzzing explorations where boolean values are true
	GuidepostNone                          // Encourages fuzzing explorations where boolean values are false
)

// guidepostExplore

func get_guidance_type_string(gt GuidepostType) string {
	switch gt {
	case GuidepostMaximize, GuidepostMinimize:
		return "numeric"
	case GuidepostAll, GuidepostNone:
		return "boolean"
		// case GuidepostExplore:
		// 	return "json"
	}
	return ""
}

type numericOperands[T operandConstraint] struct {
	Left  T `json:"left"`
	Right T `json:"right"`
}

type guidanceInfo struct {
	Data         any           `json:"guidance_data,omitempty"`
	Location     *locationInfo `json:"location"`
	GuidanceType string        `json:"guidance_type"`
	Message      string        `json:"message"`
	Id           string        `json:"id"`
	Maximize     bool          `json:"maximize"`
	Hit          bool          `json:"hit"`
}

type booleanGuidanceInfo struct {
	Data         any           `json:"guidance_data,omitempty"`
	Location     *locationInfo `json:"location"`
	GuidanceType string        `json:"guidance_type"`
	Message      string        `json:"message"`
	Id           string        `json:"id"`
	Maximize     bool          `json:"maximize"`
	Hit          bool          `json:"hit"`
}

func uses_maximize(gt GuidepostType) bool {
	return gt == GuidepostMaximize || gt == GuidepostAll
}

func newOperands[T Number](left, right T) any {
	switch any(left).(type) {
	case int8, int16, int32:
		return numericOperands[int32]{int32(left), int32(right)}
	case int, int64:
		return numericOperands[int64]{int64(left), int64(right)}
	case uint8, uint16, uint32, uint, uint64, uintptr:
		return numericOperands[uint64]{uint64(left), uint64(right)}
	case float32, float64:
		return numericOperands[float64]{float64(left), float64(right)}
	}
	return nil
}

func build_numeric_guidance[T Number](gt GuidepostType, message string, left, right T, loc *locationInfo, id string, hit bool) *guidanceInfo {

	operands := newOperands(left, right)
	if !hit {
		operands = nil
	}

	gI := guidanceInfo{
		GuidanceType: get_guidance_type_string(gt),
		Message:      message,
		Id:           id,
		Location:     loc,
		Maximize:     uses_maximize(gt),
		Data:         operands,
		Hit:          hit,
	}

	return &gI
}

// Convenience function to construct a NamedBool used for boolean assertions
func NewNamedBool(first string, second bool) *NamedBool {
	p := NamedBool{
		First:  first,
		Second: second,
	}
	return &p
}

type namedBoolDictionary map[string]bool

func build_boolean_guidance(gt GuidepostType, message string, named_bools []NamedBool,
	loc *locationInfo,
	id string, hit bool) *booleanGuidanceInfo {

	var guidance_data any

	// To ensure the sequence and naming for the named_bool values
	if hit {
		named_bool_dictionary := namedBoolDictionary{}
		for _, named_bool := range named_bools {
			named_bool_dictionary[named_bool.First] = named_bool.Second
		}
		guidance_data = named_bool_dictionary
	}

	bgI := booleanGuidanceInfo{
		GuidanceType: get_guidance_type_string(gt),
		Message:      message,
		Id:           id,
		Location:     loc,
		Maximize:     uses_maximize(gt),
		Data:         guidance_data,
		Hit:          hit,
	}

	return &bgI
}

func numericGuidanceImpl[T Number](left, right T, message, id string, loc *locationInfo, guidepost GuidepostType, hit bool) {
	tI := numeric_gp_tracker.getTrackerEntry(id, gapTypeForOperand(left), uses_maximize(guidepost))
	gI := build_numeric_guidance(guidepost, message, left, right, loc, id, hit)
	send_value_if_needed(tI, gI)
}

func booleanGuidanceImpl(named_bools []NamedBool, message, id string, loc *locationInfo, guidepost GuidepostType, hit bool) {
	tI := boolean_gp_tracker.getTrackerEntry(id)
	bgI := build_boolean_guidance(guidepost, message, named_bools, loc, id, hit)
	tI.send_value(bgI)
}

// NumericGuidanceRaw is a low-level method designed to be used by third-party frameworks. Regular users of the assert package should not call it.
func NumericGuidanceRaw[T Number](
	left, right T,
	message, id string,
	classname, funcname, filename string,
	line int,
	guidepost GuidepostType,
	hit bool,
) {
	loc := &locationInfo{classname, funcname, filename, line, columnUnknown}
	numericGuidanceImpl(left, right, message, id, loc, guidepost, hit)
}

// BooleanGuidanceRaw is a low-level method designed to be used by third-party frameworks. Regular users of the assert package should not call it.
func BooleanGuidanceRaw(
	named_bools []NamedBool,
	message, id string,
	classname, funcname, filename string,
	line int,
	guidepost GuidepostType,
	hit bool,
) {
	loc := &locationInfo{classname, funcname, filename, line, columnUnknown}
	booleanGuidanceImpl(named_bools, message, id, loc, guidepost, hit)
}

func add_numeric_details[T Number](details map[string]any, left, right T) map[string]any {
	// ----------------------------------------------------
	// Can not use maps.Clone() until go 1.21.0 or above
	// enhancedDetails := maps.Clone(details)
	// ----------------------------------------------------
	enhancedDetails := map[string]any{}
	for k, v := range details {
		enhancedDetails[k] = v
	}
	enhancedDetails["left"] = left
	enhancedDetails["right"] = right
	return enhancedDetails
}

func add_boolean_details(details map[string]any, named_bools []NamedBool) map[string]any {
	// ----------------------------------------------------
	// Can not use maps.Clone() until go 1.21.0 or above
	// enhancedDetails := maps.Clone(details)
	// ----------------------------------------------------
	enhancedDetails := map[string]any{}
	for k, v := range details {
		enhancedDetails[k] = v
	}
	for _, named_bool := range named_bools {
		enhancedDetails[named_bool.First] = named_bool.Second
	}
	return enhancedDetails
}

// Equivalent to asserting Always(left > right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysGreaterThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left > right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

// Equivalent to asserting Always(left >= right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysGreaterThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left >= right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

// Equivalent to asserting Sometimes(T left > T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesGreaterThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left > right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

// Equivalent to asserting Sometimes(T left >= T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesGreaterThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left >= right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

// Equivalent to asserting Always(left < right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysLessThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left < right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

// Equivalent to asserting Always(left <= right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysLessThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left <= right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

// Equivalent to asserting Sometimes(T left < T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesLessThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left < right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

// Equivalent to asserting Sometimes(T left <= T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesLessThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left <= right
	all_details := add_numeric_details(details, left, right)
	assertImpl(condition, message, all_details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

// Asserts that every time this is called, at least one bool in named_bools is true. Equivalent to Always(named_bools[0].second || named_bools[1].second || ..., message, details). If you use this for assertions about the behavior of booleans, you may help Antithesis find more bugs. Information about named_bools will automatically be added to the details parameter, and the keys will be the names of the bools.
func AlwaysSome(named_bools []NamedBool, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	disjunction := false
	for _, named_bool := range named_bools {
		if named_bool.Second {
			disjunction = true
			break
		}
	}
	all_details := add_boolean_details(details, named_bools)
	assertImpl(disjunction, message, all_details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	booleanGuidanceImpl(named_bools, message, id, loc, GuidepostNone, wasHit)
}

// Asserts that at least one time this is called, every bool in named_bools is true. Equivalent to Sometimes(named_bools[0].second && named_bools[1].second && ..., message, details). If you use this for assertions about the behavior of booleans, you may help Antithesis find more bugs. Information about named_bools will automatically be added to the details parameter, and the keys will be the names of the bools.
func SometimesAll(named_bools []NamedBool, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	conjunction := true
	for _, named_bool := range named_bools {
		if !named_bool.Second {
			conjunction = false
			break
		}
	}
	all_details := add_boolean_details(details, named_bools)
	assertImpl(conjunction, message, all_details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	booleanGuidanceImpl(named_bools, message, id, loc, GuidepostAll, wasHit)
}
