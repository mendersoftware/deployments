package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

func MissingRequiredQueryMsg(name string) string {
	return fmt.Sprintf("Required query parameter missing: '%s'", name)
}

// RestErrorMsg testing function, don,t handle errors
// Used to generate equivalent body as rest.Error (from ant0ine/go-json-rest package) call would
func RestErrorMsg(status error) string {
	msg, _ := json.Marshal(map[string]string{"Error": status.Error()})
	return string(msg)
}

// ParseAndValidateUIntQuery parse and validate uint input as string min and max are included.
func ParseAndValidateUIntQuery(name, value string, min, max uint64) (uint64, error) {

	str := value
	if str == "" {
		return 0, errors.New(MissingRequiredQueryMsg(name))
	}

	uintValue, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}

	if uintValue < min || uintValue > max {
		return 0, errors.New(fmt.Sprintf("Invalid query '%s' value '%d'. Min='%d' Max='%d'.",
			name, uintValue, min, max))
	}

	return uintValue, nil
}
