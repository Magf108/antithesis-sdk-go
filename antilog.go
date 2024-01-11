package antilog

import (
  "encoding/json"
  "errors"
  "fmt"
)

type AssertInfo struct {
    Hit bool `json:"hit"`
    MustHit bool `json:"must_hit"`
    ExpectType string `json:"expect_type"`
    Expecting bool `json:"expecting"`
    Category string `json:"category"`
    Message string `json:"message"`
    Condition bool `json:"condition"`
    Id string `json:"id"`
    Location *LocationInfo `json:"location"`
    Details map[string]any `json:"details"`
}

type WrappedAssertInfo struct {
    A *AssertInfo `json:"ant_assert"`
}

type JSONDataInfo struct {
    Any any `json:"."`
}

// --------------------------------------------------------------------------------
// Version
// --------------------------------------------------------------------------------
func Version() string {
  return "0.2.0"
}


// --------------------------------------------------------------------------------
// Assertions
// --------------------------------------------------------------------------------
const was_hit = true
const must_be_hit = true
const optionally_hit = false
const expecting_true = true
const expecting_false = false

const universal_test = "every"
const existential_test = "some"
// const reachability_check "none"

// AlwaysTrue asserts that when this is evaluated
// the condition will always be true, and that this is evaluated at least once.
// Alternative name is Always()
func AlwaysTrue(text string, cond bool, values any) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, cond, values, location_info, was_hit, must_be_hit, expecting_true, universal_test)
}

// AlwaysTrueIfOccurs asserts that when this is evaluated
// the condition will always be true, or that this is never evaluated.
// Alternative name is UnreachableOrAlways()
func AlwaysTrueIfOccurs(text string, cond bool, values any) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, cond, values, location_info, was_hit, optionally_hit, expecting_true, universal_test)
}

// SometimesTrue asserts that when this is evaluated
// the condition will sometimes be true, and that this is evaluated at least once.
// Alternative name is Sometimes()
func SometimesTrue(text string, cond bool, values any) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, cond, values, location_info, was_hit, must_be_hit, expecting_true, existential_test)
}

// NeverOccurs asserts that this is never evaluated.
// This assertion will fail if it is evaluated.
// Alternative name is Unreachable()
func NeverOccurs(values any) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl("", false, values, location_info, was_hit, optionally_hit, expecting_true, universal_test)
}

// SometimesOccurs asserts that this is evaluated at least once.
// This assertion will fail if it is not evaluated, and otherwise will pass.
// Alternative name is Reachable()
func SometimesOccurs(values any) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl("", true, values, location_info, was_hit, must_be_hit, expecting_true, existential_test)
}

func AssertImpl(text string, cond bool, values any, loc *LocationInfo, hit bool, must_hit bool, expecting bool, expect_type string) {
  message_key := makeKey(loc)
  tracker_entry := assert_tracker.get_tracker_entry(message_key)
  details_map := struct_to_map(values)

  assertInfo := AssertInfo{
      Hit: hit,
      MustHit: must_hit,
      ExpectType: expect_type,
      Expecting: expecting,
      Category: "",
      Message: text,
      Condition: cond,
      Id: message_key,
      Location: loc,
      Details: details_map,
  }
  tracker_entry.emit(&assertInfo)
}


func makeKey(loc *LocationInfo) string {
    return fmt.Sprintf("%s|%d|%d", loc.Filename, loc.Line, loc.Column)
}

func struct_to_map(values any) map[string]any {

  var details_map map[string]any

  // Validate and format the details
  var data []byte = nil
  var err error
  if values != nil {
      if data, err = json.Marshal(values); err != nil {
          return details_map
      }
  }

  details_map = make(map[string]any)
  if err = json.Unmarshal(data, &details_map); err != nil {
      details_map = nil
  }
  return details_map
}


// --------------------------------------------------------------------------------
// Emit JSON structured payloads
// --------------------------------------------------------------------------------
func emit_assert(assert_info *AssertInfo) error {
  var data []byte = nil
  var err error

  wrapped_assert := WrappedAssertInfo{assert_info}
  if data, err = json.Marshal(wrapped_assert); err != nil {
      return err
  }
  payload := string(data)
  if err = json_data(payload); errors.Is(err, DSOError) {
      local_info := LocalLogAssertInfo{
        LocalLogInfo: *NewLocalLogInfo("", ""),
        WrappedAssertInfo: wrapped_assert,
      }
      if data, err = json.Marshal(local_info); err != nil {
          return err
      }
      payload = string(data)
      local_output.emit(payload)
      err = nil
  }
  return err
}
