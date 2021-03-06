package vql

import (
	"context"
	"www.velocidex.com/golang/velociraptor/glob"
	"www.velocidex.com/golang/vfilter"
)

// We also offer a VQL function to manage the upload.
// Example: select upload(file=FullPath) from glob(globs="/bin/*")

// NOTE: Due to the order in which VQL is evaluated, VQL column
// transformations happen _BEFORE_ The where condition is
// applied. This means that an expression like:

// select upload(file=FullPath) from glob(globs="/bin/*") where Size > 100

// Will cause all files to be uploaded, even if their size is smaller
// than 100. You need to instead issue the following query to apply
// the WHERE clause filtering first, then upload the result:

// let files = select * from glob(globs="/bin/*") where Size > 100
// select upload(files=FullPath) from files
type UploadFunction struct{}

func (self *UploadFunction) Call(ctx context.Context,
	scope *vfilter.Scope,
	args *vfilter.Dict) vfilter.Any {

	uploader_obj, ok := scope.Resolve("$uploader")
	if !ok {
		scope.Log("upload: Uploader not configured.")
		return vfilter.Null{}
	}

	uploader, ok := uploader_obj.(Uploader)
	if ok {
		// Extract the glob from the args.
		filename, ok := vfilter.ExtractString("file", args)
		if !ok || filename == nil {
			scope.Log("upload: Expecting a 'file' arg")
			return vfilter.Null{}
		}

		accessor := glob.OSFileSystemAccessor{}
		file, err := accessor.Open(*filename)
		if err != nil {
			scope.Log("upload: Unable to open %s: %s",
				filename, err.Error())
			return &UploadResponse{
				Error: err.Error(),
			}
		}
		defer file.Close()

		upload_response, err := uploader.Upload(
			scope, *filename, file)
		if err != nil {
			return &UploadResponse{
				Error: err.Error(),
			}
		}

		return upload_response
	}
	return vfilter.Null{}

}

func (self UploadFunction) Name() string {
	return "upload"
}

// The upload plugin uploads the files to the server using the
// configured uploader.

// Args:
//   - files: A series of filenames to upload.

// Example:
//   SELECT * from upload(files= { SELECT FullPath FROM glob(globs=['/tmp/*.txt']) })
func uploadPluginFunc(scope *vfilter.Scope, args *vfilter.Dict) []vfilter.Row {
	result := []vfilter.Row{}
	uploader_obj, ok := scope.Resolve("$uploader")
	if !ok {
		scope.Log("upload: Uploader not configured.")
		return result
	}

	uploader, ok := uploader_obj.(Uploader)
	if ok {
		// Extract the glob from the args.
		files, ok := vfilter.ExtractStringArray(
			scope, "files", args)
		if !ok {
			scope.Log("upload: Expecting a 'files' arg")
			return result
		}

		for _, filename := range files {
			accessor := glob.OSFileSystemAccessor{}
			file, err := accessor.Open(filename)
			if err != nil {
				scope.Log("upload: Unable to open %s: %s",
					filename, err.Error())
				continue
			}
			defer file.Close()

			upload_response, err := uploader.Upload(
				scope, filename, file)
			if err != nil {
				continue
			}
			result = append(result, upload_response)
		}
	}
	return result
}

func init() {
	plugin := &vfilter.GenericListPlugin{
		PluginName: "upload",
		RowType:    UploadResponse{},
	}

	plugin.Function = uploadPluginFunc
	exportedPlugins = append(exportedPlugins, plugin)
	exportedFunctions = append(exportedFunctions, &UploadFunction{})
}
