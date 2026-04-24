package cli

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/logtrace/logtrace/config"
)

func addConfigGenerateCommand(rootCmd *cobra.Command) {
	configGenCmd := &cobra.Command{
		Use:   "export-config",
		Short: "export configuration to .env.example",
		Long:  "Exports the default configuration to config/.env.example",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := &config.Config{}
			viper.SetEnvPrefix(envPrefix)
			cfg.SetDefaultValues()

			if err := viper.Unmarshal(cfg); err != nil {
				return fmt.Errorf("failed to unmarshal defaults: %w", err)
			}

			envFile, err := os.Create("config/.env.example")
			if err != nil {
				return fmt.Errorf("failed to create config/.env.example: %w", err)
			}
			defer envFile.Close()

			if err := exportEnv(envFile, cfg, envPrefix, ""); err != nil {
				return fmt.Errorf("failed to export env: %w", err)
			}
			fmt.Println("Exported config to config/.env.example")

			return nil
		},
	}

	rootCmd.AddCommand(configGenCmd)
}

func exportEnv(w io.Writer, iface any, prefix string, pathPrefix string) error {
	var sb strings.Builder
	if err := walkStruct(iface, pathPrefix, func(fieldv reflect.Value, _, envKey string) error {
		key := prefix + envKey
		value := fmt.Sprintf("%v", fieldv.Interface())
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(value)
		sb.WriteString("\n")
		return nil
	}); err != nil {
		return err
	}
	_, err := w.Write([]byte(sb.String()))
	return err
}

func walkStruct(iface any, prefix string, fn func(reflect.Value, string, string) error) error {
	v := reflect.ValueOf(iface)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldv := v.Field(i)

		tag := field.Tag.Get("mapstructure")
		tagName := strings.SplitN(tag, ",", 2)[0]
		if tagName == "" || tagName == "-" {
			continue
		}

		path := tagName
		if prefix != "" {
			path = prefix + "." + tagName
		}

		fv := fieldv
		for fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				break
			}
			fv = fv.Elem()
		}

		if fv.Kind() == reflect.Struct {
			if err := walkStruct(fv.Interface(), path, fn); err != nil {
				return err
			}
		} else {
			envKey := strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
			if err := fn(fv, path, envKey); err != nil {
				return err
			}
		}
	}
	return nil
}
