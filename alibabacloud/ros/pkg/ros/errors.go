package ros

import (
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"strings"
)

// CodeForError returns the Code for a particular error.
func CodeForError(error error) string {
	switch t := error.(type) {
	case sdkerrors.Error:
		return t.ErrorCode()
	}
	return ""
}

// IsStackNotFound returns true if the stack not found.
func IsStackNotFound(error error) bool {
	return CodeForError(error) == "StackNotFound"
}

// IsStackSame returns true if the stack is completely same.
func IsStackSame(error error) bool {
	switch t := error.(type) {
	case sdkerrors.Error:
		return t.ErrorCode() == "NotSupported" && strings.Contains(t.Message(), "completely same stack")
	}
	return false
}
