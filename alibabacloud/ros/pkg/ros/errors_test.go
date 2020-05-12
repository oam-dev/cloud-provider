package ros

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCodeOfError(t *testing.T) {
	type args struct {
		error error
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSdkError := NewMockError(ctrl)
	mockSdkError.EXPECT().ErrorCode().Return("SdkErrorCode")

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestSdkError",
			args: args{
				error: mockSdkError,
			},
			want: "SdkErrorCode",
		},
		{
			name: "TestBasicError",
			args: args{
				error: errors.New("BasicErrorCode"),
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCode := CodeOfError(tt.args.error)
			assert.Equal(t, tt.want, errorCode)
		})
	}
}

func TestIsStackNotFound(t *testing.T) {
	type args struct {
		errorCode string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestTrueForStackNotFound",
			args: args{
				errorCode: "StackNotFound",
			},
			want: true,
		},
		{
			name: "TestFalseForOtherError",
			args: args{
				errorCode: "OtherError",
			},
			want: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMockError(ctrl)
			err.EXPECT().ErrorCode().Return(tt.args.errorCode)
			assert.Equal(t, tt.want, IsStackNotFound(err))
		})
	}
}

func TestIsStackSame(t *testing.T) {
	type args struct {
		errorCode    string
		errorMessage string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestTrueForNotSupported",
			args: args{
				errorCode:    "NotSupported",
				errorMessage: "completely same stack",
			},
			want: true,
		},
		{
			name: "TestFalseForNotSupported",
			args: args{
				errorCode:    "NotSupported",
				errorMessage: "",
			},
			want: false,
		},
		{
			name: "TestFalseForOtherError",
			args: args{
				errorCode:    "OtherError",
				errorMessage: "",
			},
			want: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMockError(ctrl)
			err.EXPECT().ErrorCode().Return(tt.args.errorCode)
			err.EXPECT().Message().AnyTimes().Return(tt.args.errorMessage)
			assert.Equal(t, tt.want, IsStackSame(err))
		})
	}
}
