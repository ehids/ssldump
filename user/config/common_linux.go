//go:build !androidgki
// +build !androidgki

package config

import "log"

const (
	LD_LOAD_PATH = "/etc/ld.so.conf"
)

/*
   1, the RPATH binary header (set at build-time) of the library causing the lookup (if any)
   2, the RPATH binary header (set at build-time) of the executable
   3, the LD_LIBRARY_PATH environment variable (set at run-time)
   4, the RUNPATH binary header (set at build-time) of the executable
   5, /etc/ld.so.cache
   6, base library directories (/lib and /usr/lib)
   ref: http://blog.tremily.us/posts/rpath/
*/
var (
	default_so_paths = []string{
		"/lib",
		"/usr/lib",
		"/usr/lib64",
		"/lib64",
	}
)

func GetDynLibDirs() []string {
	dirs, err := ParseDynLibConf(LD_LOAD_PATH)
	if err != nil {
		log.Println(err.Error())
		return default_so_paths
	}
	return append(dirs, "/lib64", "/usr/lib64")
}
