//go:build !no_antithesis_sdk

package assert

type GuidepostType int

const (
	GuidepostMaximize GuidepostType = iota
	GuidepostMinimize
	GuidepostExplore
	GuidepostAll
	GuidepostNone
)

func get_guidance_type_string(gt GuidepostType) string {
	switch gt {
	case GuidepostMaximize, GuidepostMinimize:
		return "numeric"
	case GuidepostAll, GuidepostNone:
		return "boolean"
	case GuidepostExplore:
		return "json"
	}
	return ""
}

type numericOperands struct {
	Left  any `json:"left,omitempty"`
	Right any `json:"right,omitempty"`
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

func build_guidance[T Number](gt GuidepostType, message string, left, right T, loc *locationInfo, id string, hit bool) *guidanceInfo {

	operands := numericOperands{
		Left:  left,
		Right: right,
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

func NewPair(first string, second bool) *Pair {
	p := Pair{
		First:  first,
		Second: second,
	}
	return &p
}

type pairMap map[string]bool

func toPairMap(p Pair) pairMap {
	pair_map := pairMap{
		p.First: p.Second,
	}
	return pair_map
}

func build_boolean_guidance(gt GuidepostType, message string, pairs []Pair,
	loc *locationInfo,
	id string, hit bool) *booleanGuidanceInfo {

	// To ensure the sequence and naming for the pairs
	pair_list := []pairMap{}
	for _, pair := range pairs {
		pair_list = append(pair_list, toPairMap(pair))
	}

	bgI := booleanGuidanceInfo{
		GuidanceType: get_guidance_type_string(gt),
		Message:      message,
		Id:           id,
		Location:     loc,
		Maximize:     uses_maximize(gt),
		Data:         pair_list,
		Hit:          hit,
	}

	return &bgI
}

func numericGuidanceImpl[T Number](left, right T, message, id string, loc *locationInfo, guidepost GuidepostType, hit bool) {
	tI := numeric_gp_tracker.getTrackerEntry(id, TrackerTypeForNumber(left), uses_maximize(guidepost))
	gI := build_guidance(guidepost, message, left, right, loc, id, hit)
	tI.send_value(gI)
}

func booleanGuidanceImpl(pairs []Pair, message, id string, loc *locationInfo, guidepost GuidepostType, hit bool) {
	tI := boolean_gp_tracker.getTrackerEntry(id)
	bgI := build_boolean_guidance(guidepost, message, pairs, loc, id, hit)
	tI.send_value(bgI)
}

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

func BooleanGuidanceRaw[T Number](
	pairs []Pair,
	message, id string,
	classname, funcname, filename string,
	line int,
	guidepost GuidepostType,
	hit bool,
) {
	loc := &locationInfo{classname, funcname, filename, line, columnUnknown}
	booleanGuidanceImpl(pairs, message, id, loc, guidepost, hit)
}

func AlwaysGreaterThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left > right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

func AlwaysGreaterThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left >= right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

func SometimesGreaterThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left > right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

func SometimesGreaterThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left >= right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

func AlwaysLessThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left < right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

func AlwaysLessThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left <= right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMaximize, wasHit)
}

func SometimesLessThan[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left < right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

func SometimesLessThanOrEqualTo[T Number](left, right T, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	condition := left <= right
	assertImpl(condition, message, details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	numericGuidanceImpl(left, right, message, id, loc, GuidepostMinimize, wasHit)
}

func AlwaysSome(pairs []Pair, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	disjunction := false
	for _, pair := range pairs {
		if pair.Second {
			disjunction = true
			break
		}
	}
	assertImpl(disjunction, message, details, loc, wasHit, mustBeHit, universalTest, alwaysDisplay, id)

	booleanGuidanceImpl(pairs, message, id, loc, GuidepostNone, wasHit)
}

func SometimesAll(pairs []Pair, message string, details map[string]any) {
	loc := newLocationInfo(offsetAPICaller)
	id := makeKey(message, loc)
	conjunction := true
	for _, pair := range pairs {
		if !pair.Second {
			conjunction = false
			break
		}
	}
	assertImpl(conjunction, message, details, loc, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)

	booleanGuidanceImpl(pairs, message, id, loc, GuidepostAll, wasHit)
}
