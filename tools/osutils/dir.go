package osutils

import (
	"log"
	"os"
	"path/filepath"

	"done/tools/list"
)

// MoveDirContent moves the src dir content, that is not listed in the exclide list,
// to dest dir (it will be created if missing).
//
// The rootExclude argument is used to specify a list of src root entries to exclude.
//
// Note that this method doesn't delete the old src dir.
//
// It is an alternative to os.Rename() for the cases where we can't
// rename/delete the src dir (see https://done/services/datastorage/issues/2519).
func MoveDirContent(src string, dest string, rootExclude ...string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// make sure that the dest dir exist
	manuallyCreatedDestDir := false
	if _, err := os.Stat(dest); err != nil {
		if err := os.Mkdir(dest, os.ModePerm); err != nil {
			return err
		}
		manuallyCreatedDestDir = true
	}

	moved := map[string]string{}

	tryRollback := func() []error {
		errs := []error{}

		for old, new := range moved {
			if err := os.Rename(new, old); err != nil {
				errs = append(errs, err)
			}
		}

		// try to delete manually the created dest dir if all moved files were restored
		if manuallyCreatedDestDir && len(errs) == 0 {
			if err := os.Remove(dest); err != nil {
				errs = append(errs, err)
			}
		}

		return errs
	}

	for _, entry := range entries {
		basename := entry.Name()

		if list.ExistInSlice(basename, rootExclude) {
			continue
		}

		old := filepath.Join(src, basename)
		new := filepath.Join(dest, basename)

		if err := os.Rename(old, new); err != nil {
			if errs := tryRollback(); len(errs) > 0 {
				// currently just log the rollback errors
				// in the future we may require go 1.20+ to use errors.Join()
				log.Println(errs)
			}

			return err
		}

		moved[old] = new
	}

	return nil
}