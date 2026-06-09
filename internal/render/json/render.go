package json

import (
	"github.com/SanD94/redline/internal/model"
)

func Render(result *model.RevealResult) ([]byte, error) {
	return result.JSON()
}
