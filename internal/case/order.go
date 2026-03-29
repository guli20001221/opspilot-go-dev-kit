package cases

import (
	"strconv"
	"strings"
)

func caseSortsAfter(left Case, right Case) bool {
	if !left.UpdatedAt.Equal(right.UpdatedAt) {
		return left.UpdatedAt.After(right.UpdatedAt)
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.After(right.CreatedAt)
	}

	leftSeq, leftOK := caseIDNumericSuffix(left.ID)
	rightSeq, rightOK := caseIDNumericSuffix(right.ID)
	if leftOK && rightOK && leftSeq != rightSeq {
		return leftSeq > rightSeq
	}

	return left.ID > right.ID
}

func caseIDNumericSuffix(id string) (uint64, bool) {
	lastDash := strings.LastIndexByte(id, '-')
	if lastDash < 0 || lastDash == len(id)-1 {
		return 0, false
	}

	value, err := strconv.ParseUint(id[lastDash+1:], 10, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}
