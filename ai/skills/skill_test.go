package skills_test

import (
	"testing"

	"github.com/donovandicks/chatter/ai/skills"
	"github.com/stretchr/testify/assert"
)

const sample = `
name: Hello World
description: Print hello world
command: echo
args:
	- "Hello, world"
`

func TestSkill_Parse(t *testing.T) {
	cases := map[string]struct {
		skillBytes    []byte
		expectedSkill *skills.Skill
		expectedErr   error
	}{
		"basic skill": {
			skillBytes: []byte(`
name: Read File
description: Simple read file ability. Use to read a file.`),
		},
	}

	for tname, tc := range cases {
		t.Run(tname, func(t *testing.T) {
			skill, err := skills.ParseSkill(tc.skillBytes)
			assert.Equal(t, tc.expectedErr, err)

			if tc.expectedErr == nil && tc.expectedSkill != nil {
				assert.NotNil(t, skill)
				assert.Equal(t, *tc.expectedSkill, *skill)
			}
		})
	}
}
