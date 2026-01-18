package skills

import (
	"go.yaml.in/yaml/v4"
)

type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func ParseSkill(skillBytes []byte) (*Skill, error) {
	var skill Skill
	if err := yaml.Unmarshal(skillBytes, &skill); err != nil {
		return nil, err
	}

	return &skill, nil
}
