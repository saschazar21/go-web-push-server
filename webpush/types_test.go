package webpush

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEpoch(t *testing.T) {
	type epoch struct {
		Exp Epoch `json:"exp,omitempty"`
	}

	type test struct {
		name    string
		obj     any
		cmp     any
		wantErr bool
	}

	tests := []test{
		{
			"5min",
			epoch{
				Epoch{time.Now().Add(5 * time.Minute)},
			},
			fmt.Sprintf("{\"exp\":%v}", time.Now().Add(5*time.Minute).Unix()),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			enc, err := json.Marshal(tt.obj)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestEpoch err = %v, wantErr = %v", err, tt.wantErr)
			}

			assert.Equal(t, string(enc), tt.cmp)
		})
	}

	tests = []test{
		{
			"5min",
			fmt.Sprintf("{\"exp\":%v}", time.Now().Add(5*time.Minute).Unix()),
			epoch{
				Epoch{time.Now().Add(5 * time.Minute).Truncate(time.Second)},
			},
			false,
		},
		{
			"invalid",
			"{\"exp\":\"absd\"}",
			epoch{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := new(epoch)

			if err := json.Unmarshal([]byte(tt.obj.(string)), parsed); (err != nil) != tt.wantErr {
				t.Errorf("TestEpoch err = %v, wantErr = %v", err, tt.wantErr)
			}

			assert.Equal(t, *parsed, tt.cmp)
		})
	}
}
