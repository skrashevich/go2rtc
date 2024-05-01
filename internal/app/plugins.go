package app

import (
	"fmt"

	"os"
	"path/filepath"
	"plugin"

	plg "github.com/AlexxIT/go2rtc/pkg/plugin"
)

func LoadPlugins() []plg.Plugin {
	var plugins []plg.Plugin

	// Перебираем файлы в директории plugins
	filepath.Walk("plugins", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Ошибка при доступе к пути", path, ":", err)
			return err
		}
		if filepath.Ext(path) == ".so" {
			plug, err := plugin.Open(path)
			if err != nil {
				fmt.Println("Ошибка загрузки плагина:", err)
				return nil
			}
			symInit, err := plug.Lookup("Init")
			if err != nil {
				fmt.Println("Init не найден:", err)
				return nil
			}
			initFunc, ok := symInit.(func() plg.Plugin)
			if !ok {
				fmt.Println("Неверная сигнатура функции Init")
				return nil
			}
			p := initFunc()
			p.RegisterHandlers()
			plugins = append(plugins, initFunc())
		}
		return nil
	})

	return plugins
}
