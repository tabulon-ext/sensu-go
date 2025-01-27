package handler

import (
	"fmt"
	"testing"

	v2 "github.com/sensu/core/v2"
	client "github.com/sensu/sensu-go/cli/client/testing"
	test "github.com/sensu/sensu-go/cli/commands/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateCommand(t *testing.T) {
	testCases := []struct {
		args		[]string
		fetchResponse	error
		updateResponse	error
		expectedOutput	string
		expectError	bool
	}{
		{[]string{}, nil, nil, "Usage", true},
		{[]string{"foo"}, fmt.Errorf("error"), nil, "", true},
		{[]string{"foo"}, nil, fmt.Errorf("error"), "", true},
	}

	for _, tc := range testCases {
		name := ""
		if len(tc.args) > 0 {
			name = tc.args[0]
		}

		testName := fmt.Sprintf(
			"update the handler %s",
			name,
		)
		t.Run(testName, func(t *testing.T) {
			handler := v2.FixtureHandler(name)
			cli := test.NewMockCLI()

			client := cli.Client.(*client.MockClient)
			client.On(
				"FetchHandler",
				name,
			).Return(handler, tc.fetchResponse)

			client.On(
				"CreateHandler",
				mock.Anything,
			).Return(tc.updateResponse)

			cmd := UpdateCommand(cli)
			out, err := test.RunCmd(cmd, tc.args)

			assert.Regexp(t, tc.expectedOutput, out)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
