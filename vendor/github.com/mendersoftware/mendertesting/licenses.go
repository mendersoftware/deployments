// Copyright 2019 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package mendertesting

import "errors"
import "os/exec"
import "fmt"
import "os"
import "path"
import "strings"

const packageLocation string = "github.com/mendersoftware/mendertesting"

type TSubset interface {
	// What we want is actually a subclass of testing.T, with Fatal
	// overriden, but Go doesn't like that, so make this small interface
	// instead. More methods may need to be added here if we expect to use
	// more of them.

	Fatal(args ...interface{})
	Log(args ...interface{})
}

var known_license_files []string = []string{}

// Specify a license file for a dependency explicitly, avoiding the check for
// common license file names.
func SetLicenseFileForDependency(license_file string) {
	known_license_files = append(known_license_files, "--add-license="+license_file)
}

var firstEnterpriseCommit = ""

// This should be set to the oldest commit that is not part of Open Source, only
// part of Enterprise, if any. IOW it should be the very first commit after the
// fork point, on the Enterprise branch.
func SetFirstEnterpriseCommit(sha string) {
	firstEnterpriseCommit = sha
}

func CheckMenderCompliance(t TSubset) {
	pathToTool, err := locatePackage()
	if err != nil {
		t.Fatal(err.Error())
	}

	cmdString := []string{path.Join(pathToTool, "check_license_go_code.sh")}
	if firstEnterpriseCommit != "" {
		cmdString = append(cmdString, "--ent-start-commit", firstEnterpriseCommit)
	}
	cmd := exec.Command(cmdString[0], cmdString[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(err.Error())
		t.Fatal(string(output[:]))
	}

	cmdString = []string{path.Join(pathToTool, "check_commits.sh")}
	cmd = exec.Command(cmdString[0], cmdString[1:]...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(err.Error())
		t.Fatal(string(output[:]))
	}

	cmdString = []string{path.Join(pathToTool, "check_license.sh")}
	cmdString = append(cmdString, known_license_files...)
	cmd = exec.Command(cmdString[0], cmdString[1:]...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(err.Error())
		t.Fatal(string(output[:]))
	}
}

func locatePackage() (string, error) {
	finalpath := path.Join("vendor", packageLocation)
	_, err := os.Stat(finalpath)
	if err == nil {
		return finalpath, nil
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return "", errors.New("Cannot check for licenses if " +
			"mendertesting is not vendored and GOPATH is unset.")
	}

	paths := strings.Split(gopath, ":")
	for i := 0; i < len(paths); i++ {
		finalpath = path.Join(paths[i], "src", packageLocation)
		_, err := os.Stat(finalpath)
		if err == nil {
			return finalpath, nil
		}
	}

	return "", fmt.Errorf("Package '%s' could not be located anywhere in GOPATH (%s)",
		packageLocation, gopath)
}
