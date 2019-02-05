/**
 * @author mlabarinas
 */

package godog

import (
	"github.com/jimlawless/cfg"
	"os"
)

var Config = make(map[string]string)

func init() {
	Config["dump_default_interval"] = "10"
	Config["dump_base_path"] = "/tmp/ramdisk/datadog_user"
	Config["dump_use_thread"] = "true"
	Config["max_files"] = "12"
	Config["max_metrics_combinatory"] = "10000"

	if os.Getenv("GO_ENVIRONMENT") == "production" {
		cfg.Load("/melicloud/common/metrics/javadog.conf", Config)
	}
}
