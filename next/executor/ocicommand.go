package executor

// OCICommand is the real OCI runtime: it drives docker or podman on PATH
// with fixed argument vectors — nothing is interpolated into a shell
// (plans/os-083112ac.md D4). Where no runtime is on PATH the declared
// `fake` runtime (executor/fakeoci) provisions instead, so CI runs
// credential-free; a runtime absent in CI is a named reason, never a
// silent skip.

import (
	"fmt"
	"os/exec"
	"strings"
)

// OCICommand shells to Bin (docker or podman).
type OCICommand struct{ Bin string }

// Available reports whether the runtime binary is on PATH.
func (r OCICommand) Available() bool {
	_, err := exec.LookPath(r.Bin)
	return err == nil
}

// Start runs a container over the workspace, bind-mounted at a fixed
// path, and returns its id and the image digest.
func (r OCICommand) Start(image, workspace string) (id, digest string, err error) {
	if image == "" || workspace == "" {
		return "", "", fmt.Errorf("%s: an image and a workspace are required", r.Bin)
	}
	run := exec.Command(r.Bin, "run", "-d", "-v", workspace+":/workspace", image, "sleep", "infinity")
	out, err := run.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("%s run: %v: %s", r.Bin, err, strings.TrimSpace(string(out)))
	}
	id = strings.TrimSpace(string(out))
	inspect := exec.Command(r.Bin, "inspect", "--format", "{{.Image}}", id)
	dout, ierr := inspect.CombinedOutput()
	if ierr != nil {
		_ = r.Stop(id)
		_ = r.Remove(id)
		return "", "", fmt.Errorf("%s inspect: %v: %s", r.Bin, ierr, strings.TrimSpace(string(dout)))
	}
	return id, strings.TrimSpace(string(dout)), nil
}

// Stop stops a running container.
func (r OCICommand) Stop(id string) error {
	if out, err := exec.Command(r.Bin, "stop", id).CombinedOutput(); err != nil {
		return fmt.Errorf("%s stop: %v: %s", r.Bin, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove removes a container.
func (r OCICommand) Remove(id string) error {
	if out, err := exec.Command(r.Bin, "rm", "-f", id).CombinedOutput(); err != nil {
		return fmt.Errorf("%s rm: %v: %s", r.Bin, err, strings.TrimSpace(string(out)))
	}
	return nil
}
