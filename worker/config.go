package worker

import (
	"fmt"

	"github.com/gotomicro/ego/core/econf"
	_ "github.com/gotomicro/ego/core/econf/file"
	"github.com/gotomicro/ego/core/econf/manager"
)

func InitConfig(filePath string) error {
	provider, parser, tag, err := manager.NewDataSource(filePath, false)
	if err != nil {
		return fmt.Errorf("load config fail: %w", err)
	}
	if err = econf.LoadFromDataSource(provider, parser, econf.WithTagName(tag)); err != nil {
		return fmt.Errorf("data source: load config, unmarshal config err: %w", err)
	}
	return nil
}
