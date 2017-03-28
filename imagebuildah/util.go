package imagebuildah

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/chrootarchive"
)

func cloneToDirectory(url, dir string) error {
	if !strings.HasPrefix(url, "git://") {
		url = "git://" + url
	}
	logrus.Debugf("cloning %q to %q", url, dir)
	cmd := exec.Command("git", "clone", url, dir)
	return cmd.Run()
}

func downloadToDirectory(url, dir string) error {
	logrus.Debugf("extracting %q to %q", url, dir)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error getting %q: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.ContentLength == 0 {
		return fmt.Errorf("no contents in %q", url)
	}
	return chrootarchive.Untar(resp.Body, dir, nil)
}

// TempDirForURL checks if the passed-in string looks like a URL.  If it is,
// TempDirForURL creates a temporary directory, arranges for its contents to be
// the contents of that URL, and returns the temporary directory's path, along
// with the name of a subdirectory which should be used as the build context
// (which may be empty or ".").  Removal of the temporary directory is the
// responsibility of the caller.  If the string doesn't look like a URL,
// TempDirForURL returns empty strings and a nil error code.
func TempDirForURL(dir, prefix, url string) (name string, subdir string, err error) {
	if !strings.HasPrefix(url, "http://") &&
		!strings.HasPrefix(url, "https://") &&
		!strings.HasPrefix(url, "git://") &&
		!strings.HasPrefix(url, "github.com/") {
		return "", "", nil
	}
	name, err = ioutil.TempDir(dir, prefix)
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary directory for %q: %v", url, err)
	}
	if strings.HasPrefix(url, "git://") {
		err = cloneToDirectory(url, name)
		if err != nil {
			if err2 := os.Remove(name); err2 != nil {
				logrus.Debugf("error removing temporary directory %q: %v", name, err2)
			}
			return "", "", err
		}
		return name, "", nil
	}
	if strings.HasPrefix(url, "github.com/") {
		ghurl := url
		url = fmt.Sprintf("https://%s/archive/master.tar.gz", ghurl)
		logrus.Debugf("resolving url %q to %q", ghurl, url)
		subdir = path.Base(ghurl) + "-master"
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		err = downloadToDirectory(url, name)
		if err != nil {
			if err2 := os.Remove(name); err2 != nil {
				logrus.Debugf("error removing temporary directory %q: %v", name, err2)
			}
			return "", subdir, err
		}
		return name, subdir, nil
	}
	logrus.Debugf("don't know how to retrieve %q", url)
	if err2 := os.Remove(name); err2 != nil {
		logrus.Debugf("error removing temporary directory %q: %v", name, err2)
	}
	return "", "", fmt.Errorf("unreachable code reached")
}
