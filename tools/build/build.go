package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/v2ray/v2ray-core/tools/git"
)

var (
	flagTargetOS   = flag.String("os", runtime.GOOS, "Target OS of this build.")
	flagTargetArch = flag.String("arch", runtime.GOARCH, "Target CPU arch of this build.")
	flagArchive    = flag.Bool("zip", false, "Whether to make an archive of files or not.")

	binPath string
)

func createTargetDirectory(version string, goOS GoOS, goArch GoArch) (string, error) {
	suffix := getSuffix(goOS, goArch)

	targetDir := filepath.Join(binPath, "v2ray-"+version+suffix)
	if version != "custom" {
		os.RemoveAll(targetDir)
	}
	err := os.MkdirAll(targetDir, os.ModeDir|0777)
	return targetDir, err
}

func getTargetFile(goOS GoOS) string {
	suffix := ""
	if goOS == Windows {
		suffix += ".exe"
	}
	return "v2ray" + suffix
}

func getBinPath() string {
	GOPATH := os.Getenv("GOPATH")
	return filepath.Join(GOPATH, "bin")
}

func main() {
	flag.Parse()
	binPath = getBinPath()
	build(*flagTargetOS, *flagTargetArch, *flagArchive, "")
}

func build(targetOS, targetArch string, archive bool, version string) {
	v2rayOS := parseOS(targetOS)
	v2rayArch := parseArch(targetArch)

	if len(version) == 0 {
		v, err := git.RepoVersionHead()
		if v == git.VersionUndefined {
			v = "custom"
		}
		if err != nil {
			fmt.Println("Unable to detect V2Ray version: " + err.Error())
			return
		}
		version = v
	}
	fmt.Printf("Building V2Ray (%s) for %s %s\n", version, v2rayOS, v2rayArch)

	targetDir, err := createTargetDirectory(version, v2rayOS, v2rayArch)
	if err != nil {
		fmt.Println("Unable to create directory " + targetDir + ": " + err.Error())
	}

	targetFile := getTargetFile(v2rayOS)
	err = buildV2Ray(filepath.Join(targetDir, targetFile), version, v2rayOS, v2rayArch)
	if err != nil {
		fmt.Println("Unable to build V2Ray: " + err.Error())
	}

	err = copyConfigFiles(targetDir, v2rayOS)
	if err != nil {
		fmt.Println("Unable to copy config files: " + err.Error())
	}

	if archive {
		err := os.Chdir(binPath)
		if err != nil {
			fmt.Printf("Unable to switch to directory (%s): %v\n", binPath, err)
		}
		suffix := getSuffix(v2rayOS, v2rayArch)
		zipFile := "v2ray" + suffix + ".zip"
		root := filepath.Base(targetDir)
		err = zipFolder(root, zipFile)
		if err != nil {
			fmt.Println("Unable to create archive (%s): %v\n", zipFile, err)
		}
	}
}
